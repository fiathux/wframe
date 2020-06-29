/* General Web framework
 * service instance core
 * Qujie Tech 2019-06-01
 * Fiathux Su
 */

package wframe

import (
	"net/http"
	"os"
)

// QInstance defined interface for service master instance
type QInstance interface {
	http.Handler
	LoadConfig(name string, refobj interface{}) error // load sub-config file
	InstConf() InstConfig                             // get instance configure
	Env() QEnv                                        // get envronment object
	WorkPath(relpath string) string                   // get work path
	//get log sender
	Log(name string) func(level QLogLevel, msg string)
	ServiceName() string // get service name
	Terminate()          // terminate instance
}

// QEnv is basic environment object interface
type QEnv interface {
	InitEnv(inst QInstance)
	ReqForEnv(req SvrReq) interface{}
	Terminate()
}

// framework instance object, it need implement QInstance interface
type svrInstance struct {
	// service init handle
	initHandle  func(http.ResponseWriter, *http.Request)
	serviceName string                  // service name identification
	workPath    string                  // service instance work path
	conf        *InstConfig             // main configure data
	env         QEnv                    // service envronment manager
	logs        map[string]*LogInstance // logger
	discard     bool                    // a tag mark service discard
}

// CreateInstance create a basic instance
func CreateInstance(path string, inithnd QHandle, env QEnv) (
	QInstance, error) {
	//read basic configure
	if err := os.Chdir(path); err != nil {
		return nil, err
	}
	conf := newFrmConf()
	if err := loadConf(defaultConfFile, conf); err != nil {
		return nil, err
	}
	servname := conf.ServiceName
	allLogger := make(map[string]*LogInstance)
	if conf.Logs != nil {
		for i, v := range conf.Logs {
			allLogger[i] = initLog(i, &v)
		}
	}
	// make instance object
	inst := &svrInstance{
		nil,
		servname,
		path,
		conf,
		nil,
		allLogger,
		false,
	}
	inst.initHandle = QHandle2HandlerFunc(inithnd, inst)
	if env != nil {
		env.(QEnv).InitEnv(inst)
		inst.env = env
	}
	return inst, nil
}

////////////////// methods //////////////////

// implement 'Handler': ServeHTTP
func (s *svrInstance) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	s.initHandle(rsp, req)
}

// load sub-config file
func (s *svrInstance) LoadConfig(name string, refobj interface{}) error {
	if s.conf.Includes == nil {
		return nil
	}
	if fname, ok := s.conf.Includes[name]; ok {
		err := loadConf(fname, refobj)
		if err != nil {
			return err
		}
	}
	return nil
}

// get envronment object
func (s *svrInstance) Env() QEnv {
	return s.env
}

// get work path. add relative path after work path if 'relpath' specified
func (s *svrInstance) WorkPath(relpath string) string {
	if relpath == "" || relpath == "/" {
		return s.workPath
	}
	wp := s.workPath
	if wp == "" {
		return wp
	}
	if wp == "/" || wp[len(wp)-1:len(wp)] == "/" {
		if relpath[0:1] == "/" {
			return wp + relpath[1:len(relpath)]
		}
		return wp + relpath
	}
	if relpath[0:1] == "/" {
		return wp + relpath
	}
	return wp + "/" + relpath
}

// get service name
func (s *svrInstance) ServiceName() string {
	return s.serviceName
}

// get log instance
func (s *svrInstance) logInst(name string) *LogInstance {
	if logobj, ok := s.logs[name]; ok {
		return logobj
	}
	return nil
}

// get log sender
func (s *svrInstance) Log(name string) func(level QLogLevel, msg string) {
	if logobj, ok := s.logs[name]; ok {
		return logobj.sender
	}
	panic("no logger named: " + name)
}

// get instance configure
func (s *svrInstance) InstConf() InstConfig {
	return *s.conf
}

// terminate service
func (s *svrInstance) Terminate() {
	s.discard = true
	for _, v := range s.logs {
		v.term()
	}
}
