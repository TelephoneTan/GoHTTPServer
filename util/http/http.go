package http

import (
	"github.com/TelephoneTan/GoHTTPServer/net/http/header"
	"github.com/TelephoneTan/GoHTTPServer/net/http/method"
	"github.com/TelephoneTan/GoHTTPServer/net/mime"
	"net/http"
	"strings"
)

func SetAllow(w http.ResponseWriter, method ...method.Method) {
	methods := make([]string, 0, len(method))
	for _, m := range method {
		methods = append(methods, m.String())
	}
	w.Header().Set(header.Allow, strings.Join(methods, ","))
}
func SetContentType(w http.ResponseWriter, t mime.Type) {
	w.Header().Set(header.ContentType, t)
}
func SetLocation(w http.ResponseWriter, location string) {
	w.Header().Set(header.Location, location)
}
