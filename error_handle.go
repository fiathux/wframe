/* General Web framework
 * default error handler
 * Qujie Tech 2019-06-04
 * Fiathux Su
 */

package wframe

import (
	"fmt"
	"io"
	"net/http"
)

var httpErrorTitles = map[int]string{
	http.StatusBadRequest:                    "Bad Request",
	http.StatusUnauthorized:                  "Unauthorized",
	http.StatusPaymentRequired:               "Payment Required",
	http.StatusForbidden:                     "Forbidden",
	http.StatusNotFound:                      "Not Found",
	http.StatusMethodNotAllowed:              "Method Not Allowed",
	http.StatusNotAcceptable:                 "Not Acceptable",
	http.StatusProxyAuthRequired:             "Proxy Auth Required",
	http.StatusRequestTimeout:                "Request Timeout",
	http.StatusConflict:                      "Conflict",
	http.StatusGone:                          "Gone",
	http.StatusLengthRequired:                "Length Required",
	http.StatusPreconditionFailed:            "Precondition Failed",
	http.StatusRequestEntityTooLarge:         "Request Entity Too Large",
	http.StatusRequestURITooLong:             "Status Request URI Too Long",
	http.StatusUnsupportedMediaType:          "Unsupported Media Type",
	http.StatusRequestedRangeNotSatisfiable:  "Requested Range Not Satisfiable",
	http.StatusExpectationFailed:             "Expectation Failed",
	http.StatusTeapot:                        "Teapot",
	http.StatusMisdirectedRequest:            "Misdirected Request",
	http.StatusUnprocessableEntity:           "Unprocessable Entity",
	http.StatusLocked:                        "Locked",
	http.StatusFailedDependency:              "Failed Dependency",
	http.StatusUpgradeRequired:               "Upgrade Required",
	http.StatusPreconditionRequired:          "Precondition Required",
	http.StatusTooManyRequests:               "Too Many Requests",
	http.StatusRequestHeaderFieldsTooLarge:   "Request Header Fields Too Large",
	http.StatusUnavailableForLegalReasons:    "Unavailable For Legal Reasons",
	http.StatusInternalServerError:           "Internal Server Error",
	http.StatusNotImplemented:                "Not Implemented",
	http.StatusBadGateway:                    "Bad Gateway",
	http.StatusServiceUnavailable:            "Service Unavailable",
	http.StatusGatewayTimeout:                "Gateway Timeout",
	http.StatusHTTPVersionNotSupported:       "HTTP Version Not Supported",
	http.StatusVariantAlsoNegotiates:         "Variant Also Negotiates",
	http.StatusInsufficientStorage:           "Insufficient Storage",
	http.StatusLoopDetected:                  "Loop Detected",
	http.StatusNotExtended:                   "Not Extended",
	http.StatusNetworkAuthenticationRequired: "Network Authentication Required",
}

//Error handle and it work as a session
type errHandle struct {
	stateCode int
	msg       []byte
}

// general error init
func (hnd *errHandle) InitHandler(
	inst QInstance, rte RouteHandle, ptree *RouteTree) {
	return
}

// handle work as a session
func (hnd *errHandle) BeginSession(
	req SvrReq, env interface{}) QSession {
	return hnd
}

// general error response
func (hnd *errHandle) EnterServer() (redirect string, err error) {
	return "", nil
}

// general error response
func (hnd *errHandle) BeginResponse(header http.Header) (
	status int) {
	header.Add("Content-Type", "text/html;charset=utf-8")
	return hnd.stateCode
}

// general error output
func (hnd *errHandle) WriteResponse(rsp io.Writer) []byte {
	return hnd.msg
}

// CreateErrSession make a default http error session
func CreateErrSession(code int, msg string, debug *string) QSession {
	title, ok := httpErrorTitles[code]
	if !ok {
		return nil
	}
	if debug != nil {
		msg = fmt.Sprintf("<h1>%d %s</h1><p>%s</p>%s", code, title, msg, *debug)
	} else {
		msg = fmt.Sprintf("<h1>%d %s</h1><p>%s</p>", code, title, msg)
	}
	return &errHandle{code, []byte(msg)}
}
