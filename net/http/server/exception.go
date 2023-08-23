package server

import "net/http"

type Exception interface {
	HTTPCode() int
	SetHeader(http.ResponseWriter)
}
