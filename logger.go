/* General Web framework
 * log file support
 * Qujie Tech 2019-06-18
 * Fiathux Su
 */

package wframe

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// QLogLevel defined framework log level
type QLogLevel uint

// all log level
const (
	LQLogDEBUG  QLogLevel = iota //full debug info level
	LQLogTALK                    //domain talk info level
	LQLogINFO                    //global message level
	LQLogWARN                    //warning level
	LQLogEXCEPT                  //function exception level
	LQLogERROR                   //error level
	LQLogFATAL                   //fatal error level
)

const checkInterval int = 60 // log status check timer interval (second)
const retryLogError uint = 3 // retry log create error
const fileLife uint = 30     // log file life(day)
const backupDir string = "recent_log"

// file handler type
const (
	FTypeNone int = iota
	FTypeFile
	FTypePipe
)

// log control signal
const (
	ctrLogRenew int = iota
	ctrLogFork
)

// log message struct
type frmLogUnit struct {
	level QLogLevel
	msg   string
	ts    int64
}

// LogInstance is way to control log instance object
type LogInstance struct {
	sender func(level QLogLevel, msg string)
	term   func()
}

// log message format
type frmLogConf struct {
	FileName   string `yaml:"file_name,omitempty"`
	TimeFmt    string `yaml:"time_fmt,omitempty"`
	MsgFmt     string `yaml:"msg_fmt,omitempty"`
	QueueSize  uint   `yaml:"queue_size,omitempty"`
	ReserveDay uint   `yaml:"reserve_day,omitempty"`
}

type frmLogger struct {
	frmLogConf
	logname  string
	fileType int
	rawFile  *os.File    // raw file handle
	rawlog   *log.Logger // raw log object
	chanTerm chan bool   // terminate controller
	lock     sync.Mutex  //sync locker
}

// get file handle from filename
func fname2handle(fname string) (io.Writer, int, error) {
	// fixed file name
	fname = strings.Trim(fname, " ")
	if fname == "" {
		fname = "-"
	}
	if fname == "-" {
		return os.Stdout, FTypePipe, nil
	} else if fname == "=" {
		return os.Stderr, FTypePipe, nil
	}
	fp, err := os.OpenFile(fname, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return nil, FTypeNone, err
	}
	return fp, FTypeFile, nil
}

// fork a new log file
func (l *frmLogger) forklog() {
	if l.fileType != FTypeFile {
		return
	}
	// replace log file
	var mvpath string
	for i := 0; ; i++ {
		mvpath = fmt.Sprintf("%s/%s%02d.%s", backupDir,
			time.Now().Format("20060102"), i, l.FileName)
		_, err := os.Stat(mvpath)
		if err != nil && os.IsNotExist(err) {
			break
		}
	}
	os.Rename(l.FileName, mvpath)
	//renew log handle
	if err := l.renewlog(); err != nil {
		panic(fmt.Sprintf("failed renew Log when fork it - %q", err))
	}
	//TODO: do backup async
}

// renew log
func (l *frmLogger) renewlog() error {
	if l.fileType == FTypeFile && l.rawFile != nil {
		l.rawFile.Close()
	}
	fp, ftype, err := fname2handle(l.FileName)
	if err != nil {
		l.rawFile = nil
		l.rawlog = nil
		l.fileType = FTypeNone
		return err
	}
	var logPrefix string
	if ftype == FTypePipe {
		logPrefix = fmt.Sprintf("%s: ", l.logname)
	} else {
		logPrefix = ""
	}
	l.rawFile = fp.(*os.File)
	l.rawlog = log.New(fp, logPrefix, 0)
	l.fileType = ftype
	return nil
}

// log level to string
func logLevelMap(level QLogLevel) string {
	switch level {
	case LQLogDEBUG:
		return "[DEBUG]"
	case LQLogTALK:
		return "[TALK]"
	case LQLogINFO:
		return "[INFO]"
	case LQLogWARN:
		return "[WARNING]"
	case LQLogEXCEPT:
		return "[EXCEPT]"
	case LQLogERROR:
		return "[ERROR]"
	case LQLogFATAL:
		return "[FATAL!]"
	default:
		return "[NOLEVEL]"
	}
}

// log send processor
func loggerProc(logobj *frmLogger, datach chan *frmLogUnit, ctrl chan int) {
	// Proc loop
	for {
		select {
		case d := <-datach:
			// write log
			timstr := time.Unix(d.ts, 0).Format(logobj.TimeFmt)
			logobj.rawlog.Printf(logobj.MsgFmt, logLevelMap(d.level), timstr, d.msg)
		case ctr, ctrOk := <-ctrl:
			if !ctrOk {
				return
			}
			// log control
			switch ctr {
			case ctrLogRenew:
				if err := logobj.renewlog(); err != nil {
					panic(fmt.Sprintf("failed renew Log handler - %q", err))
				}
			case ctrLogFork:
				logobj.forklog()
			}
		}
	}
}

// log check timer
func loggerCtrl(logobj *frmLogger, ctrl chan int) {
	dayts := func() int64 {
		ts := time.Now().Unix()
		return ts - (ts % 86400)
	}
	dayTs := dayts()
	// Proc loop
	for {
		select {
		case <-time.After(time.Second * time.Duration(checkInterval)):
			updts := dayts()
			if updts != dayTs {
				dayTs = updts
				ctrl <- ctrLogFork
			} else {
				ctrl <- ctrLogRenew
			}
		case <-logobj.chanTerm:
			close(ctrl)
			return
		}
	}
}

// create log instance and start log threads
func initLog(name string, conf *frmLogConf) *LogInstance {
	{
		// prepare a backup directory for legacy log
		fstat, err := os.Stat(backupDir)
		if err != nil {
			if os.IsNotExist(err) {
				os.Mkdir(backupDir, 0755)
			} else {
				panic(fmt.Sprintf("failed check log backup directory - %q", err))
			}
		}
		if err == nil && !fstat.IsDir() {
			panic(fmt.Sprintf("failed check log backup directory - "+
				"%s is not a directory", backupDir))
		}
	}
	logobj := frmLogger{
		frmLogConf: *conf,
		logname:    name,
		chanTerm:   make(chan bool),
	}
	if err := logobj.renewlog(); err != nil {
		panic(fmt.Sprintf("failed to create Log handler - %q", err))
	}
	//create cycle msg list
	flagExec := true
	msgchan := make(chan *frmLogUnit, conf.QueueSize)
	ctrchan := make(chan int)
	// log sender
	procLog := func(level QLogLevel, msg string) {
		if !flagExec {
			return
		}
		logobj.lock.Lock()
		defer logobj.lock.Unlock()
		if !flagExec { // double check
			return
		}
		ts := time.Now().Unix()
		msgchan <- &frmLogUnit{level, msg, ts}
	}
	// log termnator
	procTerm := func() {
		logobj.lock.Lock()
		defer logobj.lock.Unlock()
		if !flagExec {
			return
		}
		flagExec = false
		close(logobj.chanTerm)
	}
	go loggerProc(&logobj, msgchan, ctrchan)
	go loggerCtrl(&logobj, ctrchan)
	return &LogInstance{procLog, procTerm}
}
