/* General Web framework
 * normal prepare handler
 * Qujie Tech 2019-07-09
 * Fiathux Su
 */

package wframe

import (
	"io"
	"net/http"
	"strings"
)

// RedirCode is a domian redirct code
type RedirCode int

// supported redirct code
const (
	RdirMovePermanently     RedirCode = http.StatusMovedPermanently  //301
	RdirMove                          = http.StatusFound             //302
	RdirRedirectPermanently           = http.StatusPermanentRedirect //308
	RdirRedirect                      = http.StatusTemporaryRedirect //307
)

// static file server
type fileHandle struct {
	filepath string
	rawhnd   http.Handler
}

type ht3xxSession struct {
	code int
}

// redirect session
type redirSession struct {
	ht3xxSession
	url string
}

// inner redirect session
type aliasSession struct {
	route string
}

// CreateFilesystemHandle create a static file handle
func CreateFilesystemHandle(filepath string) QHandle {
	return &fileHandle{filepath, nil}
}

// CreateRedirectSession create a redirect session
func CreateRedirectSession(url string, redirtype RedirCode) QSession {
	return &redirSession{ht3xxSession{int(redirtype)}, url}
}

// CreateNochangeSession create HTTP 304 not change session
func CreateNochangeSession() QSession {
	return &ht3xxSession{http.StatusNotModified}
}

// CreateAliasSession create a alias session
func CreateAliasSession(path string) QSession {
	if path == "" {
		path = "/"
	}
	return &aliasSession{path}
}

//////////////////// fileHandle methods ////////////////////

// fileHandle: init
func (hnd *fileHandle) InitHandler(
	inst QInstance, rte RouteHandle, ptree *RouteTree) {
	fhnd := http.FileServer(http.Dir(hnd.filepath))
	if ptree != nil && ptree.BindPath != nil {
		strpath := "/" + strings.Join(ptree.BindPath, "/")
		fhnd = http.StripPrefix(strpath, fhnd)
	}
	hnd.rawhnd = fhnd
}

// fileHandle: create session
func (hnd *fileHandle) BeginSession(
	req SvrReq, env interface{}) QSession {
	return &rawSession{hnd.rawhnd.ServeHTTP, req, nil}
}

//////////////////// ht3xxSession methods ////////////////////

// default 3xx EnterServer
func (ses *ht3xxSession) EnterServer() (redirect string, err error) {
	return "", nil
}

// default 3xx BeginResponse
func (ses *ht3xxSession) BeginResponse(header http.Header) (status int) {
	return ses.code
}

// default 3xx no content
func (ses *ht3xxSession) WriteResponse(rsp io.Writer) []byte {
	return nil
}

// redirect BeginResponse
func (ses *redirSession) BeginResponse(header http.Header) (status int) {
	header.Add("Location", ses.url)
	return ses.ht3xxSession.BeginResponse(header)
}

//////////////////// aliasSession methods ////////////////////

// aliasSession EnterServer
func (ses *aliasSession) EnterServer() (redirect string, err error) {
	return ses.route, nil
}

// aliasSession BeginResponse
func (ses *aliasSession) BeginResponse(header http.Header) (status int) {
	return http.StatusForbidden
}

// aliasSession WriteResponse
func (ses *aliasSession) WriteResponse(rsp io.Writer) []byte {
	return []byte("Nothing content in Alias")
}
