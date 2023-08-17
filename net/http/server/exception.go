package server

type Exception interface {
	HTTPCode() int
}
