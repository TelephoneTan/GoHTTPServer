package server

import "net/http"

type Exception interface {
	Code() string
	ID() string
	HTTPCode() int
	SetHeader(http.ResponseWriter)
	TipZH() string
}
