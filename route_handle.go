/* Program error code verson manager
 * route handler
 * Qujie Tech 2019-06-04
 * Fiathux Su
 */

package wframe

import (
	"fmt"
	"net/http"
)

const pathDefaultCapcity = 16
const restMethodCount = 7

// supported method
const (
	MethodDEL   string = "DELETE"
	MethodGET          = "GET"
	MethodHEAD         = "HEAD"
	MethodOPT          = "OPTIONS"
	MethodPATCH        = "PATCH"
	MethodPOST         = "POST"
	MethodPUT          = "PUT"
)

// RouteTree is a inherted route tree info
type RouteTree struct {
	BindMethod string
	BindPath   []string
	Exten      interface{} // reserve for extension
}

// mounted route node object
type routeNode struct {
	RouteTree
	hnd QHandle
}

// RouteHandle is framework HTTP route interface
type RouteHandle interface {
	QHandle
	Handle(pattern string, handler QHandle) // mount http.Handler object
	// mount http.HandleFunc
	RawHandleFunc(pattern string,
		handler func(http.ResponseWriter, *http.Request))
	// mount genenral framework handle object
	RawHandle(pattern string, handler http.Handler)
	Parent() RouteHandle
	Gwtree() RouteTree
}

// basic route struct
type frmRteBase struct {
	inst     QInstance
	treeinfo *RouteTree
	parent   RouteHandle
	nodes    []routeNode
}

// path route struct
type frmRtePath struct {
	frmRteBase
	allpath  map[string]interface{}
	rootnode QHandle
}

// RESTful style route struct
type frmRteREST struct {
	frmRteBase
	allmth map[string]QHandle
}

// supported REST method check table
var supportedRESTTab = map[string]bool{
	MethodDEL:   true,
	MethodGET:   true,
	MethodHEAD:  true,
	MethodOPT:   true,
	MethodPATCH: true,
	MethodPOST:  true,
	MethodPUT:   true,
}

// check REST action supported
func checkRESTMethod(method string) bool {
	_, ok := supportedRESTTab[method]
	return ok
}

// CreatePathHandle create path route handle
func CreatePathHandle() RouteHandle {
	return &frmRtePath{
		frmRteBase: frmRteBase{nodes: make([]routeNode, 0, pathDefaultCapcity)},
		allpath:    make(map[string]interface{}),
	}
}

// CreateRESTHandle create RESTful style route handle
func CreateRESTHandle() RouteHandle {
	return &frmRteREST{
		frmRteBase: frmRteBase{nodes: make([]routeNode, 0, restMethodCount)},
		allmth:     make(map[string]QHandle),
	}
}

////////////////////////// common methods //////////////////////////
// initialization, include all sub handles
func (rhnd *frmRteBase) initHandlerBase(
	inst QInstance, rte RouteHandle, ptree *RouteTree, self RouteHandle) {
	var subtree func(mnt *RouteTree) *RouteTree
	if ptree != nil { // combine parent node
		rhnd.treeinfo = ptree
		subtree = func(mnt *RouteTree) *RouteTree {
			//check method
			var method string
			if ptree.BindMethod != "" {
				method = ptree.BindMethod
			} else {
				method = mnt.BindMethod
			}
			// check path
			var spath []string
			if ptree.BindPath != nil || mnt.BindPath != nil {
				spath = make([]string, 0, pathDefaultCapcity)
				if ptree.BindPath != nil {
					for _, v := range ptree.BindPath {
						spath = append(spath, v)
					}
				}
				if mnt.BindPath != nil {
					for _, v := range mnt.BindPath {
						spath = append(spath, v)
					}
				}
			}
			// check extension
			var ext interface{}
			if mnt.Exten != nil {
				ext = mnt.Exten
			} else {
				ext = ptree.Exten
			}
			return &RouteTree{method, spath, ext}
		}
	} else { // non parent
		subtree = func(mnt *RouteTree) *RouteTree {
			ptreenode := *mnt
			ptreenode.BindPath = make([]string, len(mnt.BindPath))
			copy(ptreenode.BindPath, mnt.BindPath)
			return &ptreenode
		}
	}
	rhnd.inst = inst
	rhnd.parent = rte
	for _, v := range rhnd.nodes {
		v.hnd.InitHandler(inst, self, subtree(&v.RouteTree))
	}
}

// get parent route handle
func (rhnd *frmRteBase) Parent() RouteHandle {
	return rhnd.parent
}

// get a copy of current tree info
func (rhnd *frmRteBase) Gwtree() RouteTree {
	if rhnd.treeinfo == nil {
		return RouteTree{"", nil, nil}
	}
	var bpath []string
	if rhnd.treeinfo.BindPath != nil {
		bpath = make([]string, len(rhnd.treeinfo.BindPath))
		copy(bpath, rhnd.treeinfo.BindPath)
	}
	return RouteTree{
		rhnd.treeinfo.BindMethod,
		bpath,
		rhnd.treeinfo.Exten,
	}
}

// a decorator for apply debug tag. to decide message output or not
func (rhnd *frmRteBase) DebugMsg(msgf func() string) func() *string {
	if rhnd.inst.InstConf().Debuging {
		return func() *string {
			msg := msgf()
			return &msg
		}
	}
	return func() *string { return nil }
}

////////////////////////// path route methods //////////////////////////

func (rhnd *frmRtePath) InitHandler(
	inst QInstance, rte RouteHandle, ptree *RouteTree) {
	if rhnd.rootnode != nil {
		rhnd.rootnode.InitHandler(inst, rte, ptree)
	}
	rhnd.frmRteBase.initHandlerBase(inst, rte, ptree, rhnd)
}

// get root node
func (rhnd *frmRtePath) getRoot() (interface{}, bool, []string) {
	if rhnd.rootnode != nil {
		return interface{}(rhnd.rootnode), false, nil
	}
	return interface{}(rhnd.allpath), false, nil
}

// set root node
func (rhnd *frmRtePath) setRoot(hnd QHandle) {
	if rhnd.rootnode != nil {
		panic("failed add handle to path. sepcify locate already exists")
	} else {
		rhnd.rootnode = hnd
	}
}

// found node from path tree
func (rhnd *frmRtePath) findPath(
	pattern []string, newpath bool) (interface{}, bool, []string) {
	if pattern == nil || len(pattern) < 1 {
		return rhnd.getRoot()
	}
	innerPath := false
	pathstep := rhnd.allpath
	pathstack := make([]string, 0, pathDefaultCapcity)
	for _, l := range pattern {
		if l == "" {
			continue
		}
		if l[0:1] == "@" {
			innerPath = true
		}
		nextstep, ok := pathstep[l]
		if !ok {
			if newpath {
				nextstep = make(map[string]interface{})
				pathstep[l] = interface{}(nextstep)
			} else {
				return pathstep, innerPath, pathstack
			}
		}
		pathstack = append(pathstack, l)
		switch nextstep.(type) {
		case map[string]interface{}:
			pathstep = nextstep.(map[string]interface{})
		case QHandle:
			expHnd := nextstep.(QHandle)
			return expHnd, innerPath, pathstack
		default:
			return nil, false, nil
		}
	}
	return pathstep, innerPath, pathstack
}

// check path available
func (rhnd *frmRtePath) checkPath(pattern []string) bool {
	hnd, _, _ := rhnd.findPath(pattern, false)
	switch hnd.(type) {
	case map[string]interface{}:
		return true
	default:
		return false
	}
}

// find QHandle from path
func (rhnd *frmRtePath) findHandle(
	pattern []string) (QHandle, bool, []string) {
	hnd, inner, step := rhnd.findPath(pattern, false)
	switch hnd.(type) {
	case QHandle:
		expHnd := hnd.(QHandle)
		return expHnd, inner, step
	default:
		return nil, false, nil
	}
}

// insert QHandle into route
func (rhnd *frmRtePath) insertHandle(pattern []string, handler QHandle) {
	if handler == nil {
		panic("handle can not set nil")
	}
	if pattern == nil {
		rhnd.setRoot(handler)
		return
	}
	if len(pattern) > 1 {
		hnd, _, _ := rhnd.findPath(pattern[:len(pattern)-1], true)
		if hnd == nil {
			panic("failed add handle to path. specify locate not available")
		}
		pmap, ok := hnd.(map[string]interface{})
		if !ok {
			panic("failed add handle to path. sepcify locate already exists")
		}
		_, ok = pmap[pattern[len(pattern)-1]]
		if ok {
			panic("failed add handle to path. sepcify locate already exists")
		}
		pmap[pattern[len(pattern)-1]] = interface{}(handler)
	} else {
		rhnd.allpath[pattern[0]] = interface{}(handler)
	}
	// combine all route path
	fwdpath := make([]string, len(pattern), restMethodCount)
	copy(fwdpath, pattern)
	rhnd.nodes = append(rhnd.nodes, routeNode{RouteTree{"", fwdpath, nil}, handler})
}

// implement Handle
func (rhnd *frmRtePath) RawHandle(pattern string, handler http.Handler) {
	rhnd.insertHandle(splitePath(pattern), Handle2QHandle(handler))
}

// implement HandleFunc
func (rhnd *frmRtePath) RawHandleFunc(
	pattern string, handler func(http.ResponseWriter, *http.Request)) {
	rhnd.insertHandle(splitePath(pattern), HandleFunc2QHandle(handler))
}

// implement HandleFrame
func (rhnd *frmRtePath) Handle(pattern string, handler QHandle) {
	rhnd.insertHandle(splitePath(pattern), handler)
}

// implement BeginSession in QHandle
func (rhnd *frmRtePath) BeginSession(req SvrReq, env interface{}) QSession {
	hnd, inner, step := rhnd.findHandle(req.GetPath(true))
	dbgmsg := rhnd.DebugMsg(func() string { return req.String() })
	if hnd == nil {
		if rhnd.rootnode != nil {
			return rhnd.rootnode.BeginSession(req, env)
		}
		return CreateErrSession(http.StatusNotFound, "You known", dbgmsg())
	}
	if inner && !req.isRedir() {
		return CreateErrSession(
			http.StatusForbidden, "Unavailable this locate", dbgmsg())
	}
	req.trimPath(step)
	return hnd.BeginSession(req, env)
}

////////////////////////// REST route methods //////////////////////////

//
func (rhnd *frmRteREST) InitHandler(
	inst QInstance, rte RouteHandle, ptree *RouteTree) {
	if ptree != nil {
		if ptree.BindMethod != "" {
			panic("can not initialize REST api, it already binded in parent.")
		}
	}
	rhnd.frmRteBase.initHandlerBase(inst, rte, ptree, rhnd)
}

// check handle name enabled
func (rhnd *frmRteREST) checkHandleName(name string) {
	if !(checkRESTMethod(name)) {
		panic(fmt.Sprintf("unsupported method %s", name))
	}
	if _, ok := rhnd.allmth[name]; ok {
		panic(fmt.Sprintf("method %s already registed", name))
	}
}

// implement Handle
func (rhnd *frmRteREST) RawHandle(pattern string, handler http.Handler) {
	rhnd.Handle(pattern, Handle2QHandle(handler))
}

// implement HandleFunc
func (rhnd *frmRteREST) RawHandleFunc(pattern string,
	handler func(http.ResponseWriter, *http.Request)) {
	rhnd.Handle(pattern, HandleFunc2QHandle(handler))
}

// implement HandleFrame
func (rhnd *frmRteREST) Handle(pattern string, handler QHandle) {
	rhnd.checkHandleName(pattern)
	rhnd.allmth[pattern] = handler
	rhnd.nodes = append(rhnd.nodes, routeNode{
		RouteTree{pattern, nil, nil},
		handler,
	})
}

// implement BeginSession in QHandle
func (rhnd *frmRteREST) BeginSession(req SvrReq, env interface{}) QSession {
	dbgmsg := rhnd.DebugMsg(func() string { return req.String() })
	if !checkRESTMethod(req.Method()) {
		return CreateErrSession(
			http.StatusMethodNotAllowed, "unsupported method", dbgmsg())
	}
	hnd, ok := rhnd.allmth[req.Method()]
	if !ok {
		return CreateErrSession(
			http.StatusMethodNotAllowed, "method not implement", dbgmsg())
	}
	return hnd.BeginSession(req, env)
}
