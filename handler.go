/* General Web framework
 * framework handle and convert
 * Qujie Tech 2019-06-27
 * Fiathux Su
 */

package wframe

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
)

// QSession defined basic session interface in framework
type QSession interface {
	EnterServer() (redirect string, err error)
	BeginResponse(header http.Header) (status int)
	WriteResponse(rsp io.Writer) []byte
}

// QHandle defined basic service handler interface in framework
type QHandle interface {
	InitHandler(inst QInstance, rte RouteHandle, ptree *RouteTree)
	BeginSession(req SvrReq, env interface{}) QSession
}

// raw ResponseWriter adapter for framework. it implement ResponseWriter
type rawRspAdaper struct {
	rawrsp http.ResponseWriter
	//frmhandle *rawHandler
	startrep  chan bool // tell framework start response
	allowrep  chan bool // tell handle to write response
	endrep    chan bool //tell write complete to write response
	started   bool      // mark response is started
	writeable bool      // mrak response is writeable
	status    int
	err       error
}

// Raw handle adapter. it implement QHandle
type rawHandler struct {
	rawfunc func(http.ResponseWriter, *http.Request)
}

// Session of raw handle adapter, it implement QSession
type rawSession struct {
	rawfunc  func(http.ResponseWriter, *http.Request)
	req      SvrReq
	rspstart *rawRspAdaper
}

// simple handle, it only create specify session
type simpHandle struct {
	inst QInstance
	ses  func(QInstance, SvrReq, interface{}) QSession
}

// get enviropnment object for current session
func getSessionEnv(inst QInstance, req SvrReq) interface{} {
	if inst.Env() == nil {
		return nil
	}
	if renv := inst.Env(); renv != nil {
		return renv.ReqForEnv(req)
	}
	return nil
}

// QHandle2HandlerFunc convert QHandle to HandlerFunc
func QHandle2HandlerFunc(hnd QHandle, inst QInstance) func(
	rsp http.ResponseWriter, req *http.Request) {
	hnd.InitHandler(inst, nil, nil)
	var sndlog func(QLogLevel, string)
	sndlog = inst.Log("error")
	if sndlog == nil {
		sndlog = func(level QLogLevel, msg string) {
			fmt.Println(msg)
		}
	}
	// create session and process redirect
	maxRdir := inst.InstConf().MaxRedirect
	createSession := func(reqobj SvrReq) (ret QSession) {
		defer (func() {
			if err := recover(); err != nil {
				sndlog(LQLogERROR, fmt.Sprintf(
					"A big error - %q\n%s", err, string(debug.Stack())))
				ret = CreateErrSession(
					http.StatusInternalServerError, "Big Errorrrrrrr", nil)
			}
		})()
		var reqinfoFunc func() *string
		if inst.InstConf().Debuging {
			reqinfoFunc = func() *string {
				msg := reqobj.String()
				return &msg
			}
		} else {
			reqinfoFunc = func() *string { return nil }
		}
		for i := uint(0); i < maxRdir; i++ {
			ses := hnd.BeginSession(reqobj, getSessionEnv(inst, reqobj))
			if ses == nil {
				sndlog(LQLogERROR, fmt.Sprint("except session"))
				return CreateErrSession(
					http.StatusInternalServerError, "an error occured", reqinfoFunc())
			}
			rdir, err := ses.EnterServer()
			if rdir != "" {
				reqobj.redirect(splitePath(rdir))
				continue
			} else if err != nil {
				sndlog(LQLogERROR, fmt.Sprintf("handler error - %q", err))
				return CreateErrSession(
					http.StatusInternalServerError, "an error occured", reqinfoFunc())
			}
			return ses
		}
		sndlog(LQLogERROR, fmt.Sprint("max redirect detected"))
		return CreateErrSession(
			http.StatusInternalServerError, "over limited redirect", reqinfoFunc())
	}
	// export
	return func(rsp http.ResponseWriter, req *http.Request) {
		reqobj := createReqObj(inst, rsp, req)
		ses := createSession(reqobj)
		state := ses.BeginResponse(rsp.Header())
		rsp.WriteHeader(state)
		ret := ses.WriteResponse(rsp)
		if ret != nil {
			_, err := rsp.Write(ret)
			if err != nil {
				sndlog(LQLogERROR, fmt.Sprintf("handler error - %q", err))
			}
		}
	}
}

// HandleFunc2QHandle convert HandleFunc to QHandle
func HandleFunc2QHandle(
	hndf func(http.ResponseWriter, *http.Request)) QHandle {
	return &rawHandler{hndf}
}

// Handle2QHandle convert Handle object to QHandle
func Handle2QHandle(hnd http.Handler) QHandle {
	return &rawHandler{func(rsp http.ResponseWriter, req *http.Request) {
		hnd.ServeHTTP(rsp, req)
	}}
}

// CreateSimpHandle create a simple hanlde with a session factory function
func CreateSimpHandle(
	gen func(QInstance, SvrReq, interface{}) QSession) QHandle {
	return &simpHandle{nil, gen}
}

////////////////////////// raw handle methods //////////////////////////

// rawHandler: init
func (hnd *rawHandler) InitHandler(
	inst QInstance, rte RouteHandle, ptree *RouteTree) {
	return
}

// rawHandler: create session
func (hnd *rawHandler) BeginSession(
	req SvrReq, env interface{}) QSession {
	return &rawSession{hnd.rawfunc, req, nil}
}

// rawHandler: access
func (ses *rawSession) EnterServer() (redirect string, err error) {
	adapter := &rawRspAdaper{
		ses.req.rawRsp(),
		//ses,
		make(chan bool),
		make(chan bool),
		make(chan bool),
		false,
		false,
		http.StatusOK,
		nil,
	}
	go (func() {
		defer (func() {
			if err := recover(); err != nil {
				if errobj, ok := err.(error); ok {
					adapter.err = errobj
				} else {
					adapter.err = fmt.Errorf("%q", err)
				}
				close(adapter.endrep)
				close(adapter.startrep)
			} else {
				if !adapter.writeable {
					adapter.Write(nil)
				}
			}
		})()
		ses.rawfunc(adapter, ses.req.RawReq())
		close(adapter.endrep)
	})()
	<-adapter.startrep //wait reponse
	ses.rspstart = adapter
	if adapter.err != nil {
		close(adapter.allowrep)
		return "", adapter.err
	}
	return "", nil
}

// rawHandler: start response
func (ses *rawSession) BeginResponse(header http.Header) (status int) {
	return ses.rspstart.status
}

// rawHandler: write data
func (ses *rawSession) WriteResponse(rsp io.Writer) []byte {
	ses.rspstart.allowrep <- true
	<-ses.rspstart.endrep
	return nil
}

// rawRspAdaper: ResponseWriter.Header
func (adp *rawRspAdaper) Header() http.Header {
	return adp.rawrsp.Header()
}

// rawRspAdaper: ResponseWriter.Write
func (adp *rawRspAdaper) Write(data []byte) (int, error) {
	if !adp.started {
		adp.startrep <- true
		adp.started = true
	}
	if !adp.writeable {
		_, ok := <-adp.allowrep
		if !ok && adp.err == nil {
			adp.err = errors.New("Can not write data. an error occured in framework")
		}
		adp.writeable = true
	}
	if adp.err != nil {
		return 0, adp.err
	}
	if data != nil {
		return adp.rawrsp.Write(data)
	}
	return 0, nil
}

// rawRspAdaper: ResponseWriter.WriteHeader
func (adp *rawRspAdaper) WriteHeader(statusCode int) {
	adp.status = statusCode
	adp.startrep <- true
	adp.started = true
}

////////////////////////// simpHandle methods //////////////////////////

// rawHandler: init
func (hnd *simpHandle) InitHandler(
	inst QInstance, rte RouteHandle, ptree *RouteTree) {
	hnd.inst = inst
}

// rawHandler: create session
func (hnd *simpHandle) BeginSession(
	req SvrReq, env interface{}) QSession {
	return hnd.ses(hnd.inst, req, env)
}
