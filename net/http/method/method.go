package method

import "strings"

// Method 是 HTTP 方法的全大写形式
type Method string

const (
	OPTIONS Method = "OPTIONS"
	GET     Method = "GET"
	HEAD    Method = "HEAD"
	POST    Method = "POST"
	PUT     Method = "PUT"
	DELETE  Method = "DELETE"
	CONNECT Method = "CONNECT"
	TRACE   Method = "TRACE"
	PATCH   Method = "PATCH"
)

func Parse(s string) Method {
	return Method(strings.ToUpper(s))
}

func (t Method) String() string {
	return string(t)
}
