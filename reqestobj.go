/* General Web framework
 * request object define
 * Qujie Tech 2019-06-03
 * Fiathux Su
 */

package wframe

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// parse http `token` string(reference RFC2616)
var matchHttpToken = regexp.MustCompile("^[a-zA-Z0-9!#\\$%&'\\*\\+-\\.\\^_`\\|~]+$")

// ContentReadHardLimit set body content length limit for native reader
const ContentReadHardLimit uint = 8388608

// ContentReaderType is type of content reader tags
type ContentReaderType uint

// supported content reader tags
const (
	CntReaderNone ContentReaderType = iota
	CntReaderRAW
	CntReaderJSON
	CntReaderForm
	CntReaderExtern
)

// SvrReq defined server request object
type SvrReq interface {
	//environment
	RemoteAddr() string          // client address
	RawReq() *http.Request       // raw Request object
	rawRsp() http.ResponseWriter // raw ResponseWrite obejct
	// URL
	RawURL() string               // URL string
	RawQuery() string             // query string
	Method() string               // request method
	Fragment() string             // URL fragment
	HostName() string             // request hostname
	FullPath() string             // full path after hostname
	RelPath() string              // relative path in current gateway
	BasePath() string             // base path about current gateway
	GetPath(rel bool) []string    // get splited path
	trimPath(path []string) error // move relative path
	redirect(path []string)       // set path for redirect
	isRedir() bool                // check request been redirected
	Query() url.Values            // parse query parameter
	// cookies reader
	Cookie(name string) (*http.Cookie, error) // get Cookie by cookie name
	Cookies() []*http.Cookie                  // Cookies list
	//Header
	ContentLength() int64 // length of request body
	Header() http.Header  // request headers
	// body reader
	ReadBodyRaw(buff []byte) (int, error) //get body as bytes buffer
	ReadBodyJSON(ref interface{}) error   // get body as a JSON object
	ReadBodyForm() (url.Values, error)    // get body as URI form
	// Multi-part body
	ReadBodyMtPart(boundary string) *multipart.Reader
	// To string
	String() string
}

// response object
type svrRspObj struct {
	inst       QInstance           // framework instance
	req        *http.Request       // raw-request
	mrkReaded  bool                // mark body content already readed
	mrkReader  ContentReaderType   // mark readr alreay exists
	rawReadlen uint                // read body raw data length
	rawContent []byte              // raw data for native struct reader
	rsp        http.ResponseWriter // raw-response
	fullpath   []string            // full require path
	relpath    []string            // relative path
	postform   url.Values          // form data from POST content
	redir      bool                // mark gateway been redirected
}

// splite path string to a slice
func splitePath(pathstr string) []string {
	if pathstr == "" || pathstr == "/" {
		return nil
	}
	if pathstr[0:1] == "/" {
		pathstr = pathstr[1:]
	}
	sppath := strings.Split(pathstr, "/")
	outSp := make([]string, 0, len(sppath))
	for _, sp := range sppath {
		if sp == "" {
			continue
		}
		outSp = append(outSp, sp)
	}
	if len(outSp) < 1 {
		return nil
	}
	return outSp
}

// create response object
func createReqObj(
	inst QInstance, rsp http.ResponseWriter, req *http.Request) SvrReq {
	readed := !(req.ContentLength > 0)
	// path splite
	var relpath []string
	fullpath := splitePath(req.URL.Path)
	if fullpath == nil {
		relpath = nil
	} else {
		relpath = make([]string, len(fullpath))
		copy(relpath, fullpath)
	}
	// create object
	return &svrRspObj{
		inst, req, readed, CntReaderNone,
		0, nil, rsp, fullpath,
		relpath, nil, false,
	}
}

////////////////////// method //////////////////////

// client address
func (srq *svrRspObj) RemoteAddr() string {
	return srq.req.RemoteAddr
}

// raw Request object
func (srq *svrRspObj) RawReq() *http.Request {
	return srq.req
}

// raw ResponseWrite obejct
func (srq *svrRspObj) rawRsp() http.ResponseWriter {
	return srq.rsp
}

// URL string
func (srq *svrRspObj) RawURL() string {
	return srq.req.URL.String()
}

// query string
func (srq *svrRspObj) RawQuery() string {
	return srq.req.URL.RawQuery
}

// request method
func (srq *svrRspObj) Method() string {
	return srq.req.Method
}

// URL fragment
func (srq *svrRspObj) Fragment() string {
	return srq.req.URL.Fragment
}

// request hostname
func (srq *svrRspObj) HostName() string {
	return srq.req.Host
}

// parse query parameter
func (srq *svrRspObj) Query() url.Values {
	return srq.req.URL.Query()
}

// get Cookie by cookie name
func (srq *svrRspObj) Cookie(name string) (*http.Cookie, error) {
	return srq.req.Cookie(name)
}

// Cookies list
func (srq *svrRspObj) Cookies() []*http.Cookie {
	return srq.req.Cookies()
}

// get HTTP head 'Content-Length'
func (srq *svrRspObj) ContentLength() int64 {
	return srq.req.ContentLength
}

// get HTTP head object
func (srq *svrRspObj) Header() http.Header {
	return srq.req.Header
}

// safety rewrite request header
func (srq *svrRspObj) RewriteHeader(name string, values []string) error {
	if !matchHttpToken.MatchString(name) {
		return fmt.Errorf("Invalid HTTP header named %s", name)
	}
	srq.req.Header[name] = values
	return nil
}

// check body reader
func (srq *svrRspObj) checkBodyReader(readtype ContentReaderType) error {
	if srq.mrkReaded {
		return errors.New("No content")
	}
	if srq.mrkReader != CntReaderNone {
		if srq.mrkReader != CntReaderRAW || readtype != CntReaderRAW {
			return errors.New("Another reader already exists")
		}
	}
	return nil
}

// raw reader for native structure
func (srq *svrRspObj) nativeBodyRawReader() error {
	totalLen := srq.ContentLength()
	if totalLen > int64(ContentReadHardLimit) ||
		totalLen > int64(srq.inst.InstConf().LimitPost) {
		return errors.New("out of limited content length for parse")
	}
	if srq.rawContent == nil {
		srq.rawContent = make([]byte, totalLen)
		srq.rawReadlen = 0
	}
	for int64(srq.rawReadlen) < totalLen {
		rndLen, err := srq.req.Body.Read(srq.rawContent[srq.rawReadlen:])
		if rndLen > 0 {
			srq.rawReadlen += uint(rndLen)
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
	}
	srq.rawContent = srq.rawContent[:srq.rawReadlen]
	return nil
}

// read raw body data
func (srq *svrRspObj) ReadBodyRaw(buff []byte) (int, error) {
	if err := srq.checkBodyReader(CntReaderRAW); err != nil {
		return 0, err
	}
	srq.mrkReader = CntReaderRAW
	rlen, err := srq.req.Body.Read(buff)
	if rlen > 0 {
		srq.rawReadlen += uint(rlen)
		if int64(rlen) >= srq.ContentLength() {
			srq.mrkReaded = true
		}
	}
	return rlen, err
}

// read body as JSON object
func (srq *svrRspObj) ReadBodyJSON(ref interface{}) error {
	if srq.mrkReaded && srq.rawContent != nil {
		return json.Unmarshal(srq.rawContent, ref)
	}
	if err := srq.checkBodyReader(CntReaderJSON); err != nil {
		return err
	}
	srq.mrkReader = CntReaderJSON
	srq.mrkReaded = true
	if err := srq.nativeBodyRawReader(); err != nil {
		return err
	}
	return json.Unmarshal(srq.rawContent, ref)
}

// read body as URI form
func (srq *svrRspObj) ReadBodyForm() (url.Values, error) {
	if srq.postform != nil {
		return srq.postform, nil
	}
	if err := srq.checkBodyReader(CntReaderForm); err != nil {
		return nil, err
	}
	srq.mrkReader = CntReaderForm
	srq.mrkReaded = true
	if err := srq.nativeBodyRawReader(); err != nil {
		return nil, err
	}
	val, err := url.ParseQuery(string(srq.rawContent))
	if err != nil {
		return nil, err
	}
	srq.postform = val
	return val, nil
}

// read body as multipart reference RFC 2046
func (srq *svrRspObj) ReadBodyMtPart(boundary string) *multipart.Reader {
	if err := srq.checkBodyReader(CntReaderExtern); err != nil {
		return nil
	}
	srq.mrkReader = CntReaderExtern
	srq.mrkReaded = true
	return multipart.NewReader(srq.req.Body, boundary)
}

// full path after hostname
func (srq *svrRspObj) FullPath() string {
	if srq.fullpath == nil {
		return ""
	}
	return "/" + strings.Join(srq.fullpath, "/")
}

// current relative path in last level of gateway
func (srq *svrRspObj) RelPath() string {
	if srq.relpath == nil {
		return "./"
	}
	return strings.Join(srq.relpath, "/")
}

// move relative path
func (srq *svrRspObj) trimPath(path []string) error {
	rellen := len(srq.relpath)
	//check path
	for i, v := range path {
		if i+1 > rellen {
			return errors.New("specify path is longer then current path")
		}
		if srq.relpath[i] != v {
			return errors.New("specify path is not found")
		}
	}
	newpath := srq.relpath[len(path):]
	if len(newpath) < 1 {
		srq.relpath = nil
	}
	srq.relpath = newpath
	return nil
}

// base path about current gateway
func (srq *svrRspObj) BasePath() string {
	if srq.relpath == nil || len(srq.relpath) == 0 {
		if srq.fullpath != nil {
			return "/" + strings.Join(srq.fullpath, "/")
		}
		return "/"
	}
	if len(srq.relpath) == len(srq.fullpath) {
		return "/" + strings.Join(srq.fullpath, "/")
	}
	return "/" + strings.Join(srq.fullpath[0:len(srq.relpath)], "/")
}

// get splited path
func (srq *svrRspObj) GetPath(rel bool) []string {
	var swpath []string
	if rel {
		swpath = srq.relpath
	} else {
		swpath = srq.fullpath
	}
	path := make([]string, len(swpath))
	copy(path, swpath)
	return path
}

// set path for redirect
func (srq *svrRspObj) redirect(path []string) {
	srq.fullpath = make([]string, len(path))
	srq.relpath = make([]string, len(path))
	srq.redir = true
	copy(srq.fullpath, path)
	copy(srq.relpath, path)
	srq.req.URL.Path = "/" + strings.Join(path, "/")
}

// check redirect
func (srq *svrRspObj) isRedir() bool {
	return srq.redir
}

// convert to string report
func (srq *svrRspObj) String() string {
	// escape string list
	mapescape := func(origi []string, deco func(string) string) []string {
		ret := make([]string, 0, len(origi))
		for _, s := range origi {
			if deco != nil {
				ret = append(ret, deco(html.EscapeString(s)))
			} else {
				ret = append(ret, html.EscapeString(s))
			}
		}
		return ret
	}
	// headers part
	headerstr := func() string {
		headers := make([]string, 0)
		for k, v := range srq.Header() {
			headers = append(headers, fmt.Sprintf("<li>%s: %s</li>", k, v))
		}
		return fmt.Sprintf("<div><p>all Header</p><ul>%s</ul></div>",
			strings.Join(headers, ""))
	}
	// cookies part
	cookiestr := func() string {
		coo := srq.Cookies()
		if coo == nil || len(coo) < 1 {
			return ""
		}
		tagli := make([]string, 0, len(coo))
		for _, cone := range coo {
			tagli = append(
				tagli,
				fmt.Sprintf(
					"<dt style=\"border-bottom:1px solid #000;padding-bottom:3px;\">"+
						"%s</dt><dd>%s</dd><dd><ul>"+
						"<li>Domain: %s</li>"+
						"<li>Path: %s</li>"+
						"<li>Expires: %q</li>"+
						"</ul></dd>",
					html.EscapeString(cone.Name), html.EscapeString(cone.Value),
					html.EscapeString(cone.Domain), html.EscapeString(cone.Path),
					cone.Expires,
				),
			)
		}
		return fmt.Sprintf("<div><p>Cookies</p>"+
			"<dl style=\"border:1px solid #000;padding:5px;width:525px;\">%s</dl></div>",
			strings.Join(tagli, ""))
	}
	// query part
	querystr := func() string {
		vals := srq.Query()
		if len(vals) < 1 {
			return ""
		}
		qsli := make([]string, 0, len(vals))
		for k, v := range vals {
			qsli = append(
				qsli,
				fmt.Sprintf(
					"<tr><td>%s</td><td>%s</td></tr>",
					html.EscapeString(k), strings.Join(
						mapescape(v, func(s string) string {
							return fmt.Sprintf("<div>%s</div>", s)
						}), ""),
				),
			)
		}
		return fmt.Sprintf(
			"<div><p>Query parameters:</p>"+
				"<table style=\"border-collapse:collapse;\" border=1><thead><tr>"+
				"<th style=\"width:180px\">Key</th>"+
				"<th style=\"width:350px\">Value</th>"+
				"</tr></thead><tbody>%s</tbody><table></div>",
			strings.Join(qsli, ""))
	}
	// maim part
	temp := "<table style=\"border-collapse:collapse;\" border=1>" +
		"<thead><tr><th style=\"width:180px;\">Field</th>" +
		"<th style=\"width:350px\">Value</th></tr></thead>" +
		"<tbody>" +
		"<tr><td>raw url</td><td>%s</td></tr>" +
		"<tr><td>raw query string</td><td>%s</td></tr>" +
		"<tr><td>method</td><td>%s</td></tr>" +
		"<tr><td>fragment</td><td>%s</td></tr>" +
		"<tr><td>hostname</td><td>%s</td></tr>" +
		"<tr><td>full path</td><td>%s</td></tr>" +
		"<tr><td>relative path</td><td>%s</td></tr>" +
		"<tr><td>base path</td><td>%s</td></tr>" +
		"<tr><td>sp. full path</td><td>%q</td></tr>" +
		"<tr><td>sp. relative path</td><td>%q</td></tr>" +
		"<tr><td>is redirect</td><td>%t</td></tr>" +
		"<tr><td>content length</td><td>%d</td></tr>" +
		"</tbody>" +
		"</table>" + querystr() + cookiestr() + headerstr()

	return fmt.Sprintf(
		temp, html.EscapeString(srq.RawURL()),
		html.EscapeString(srq.RawQuery()), srq.Method(),
		html.EscapeString(srq.Fragment()), srq.HostName(),
		html.EscapeString(srq.FullPath()), html.EscapeString(srq.RelPath()),
		html.EscapeString(srq.BasePath()), mapescape(srq.GetPath(false), nil),
		mapescape(srq.GetPath(true), nil), srq.isRedir(),
		srq.ContentLength(),
	)
}
