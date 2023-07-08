package server

import (
	"crypto/tls"
	"github.com/TelephoneTan/GoHTTPGzipServer/gzip"
	httpUtil "github.com/TelephoneTan/GoHTTPServer/util/http"
	"github.com/TelephoneTan/GoLog/log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Service struct {
	Network string
	Address string
	UseTLS  bool
	UseGzip bool
}

type PickSSLCertFunc = func(info *tls.ClientHelloInfo) (*tls.Certificate, error)
type HandleFunc = func(http.ResponseWriter, *http.Request)

type _Container struct {
	GetServices                func() []Service
	GetHandleFunc              func() HandleFunc
	GetPickSSLCertFunc         func() PickSSLCertFunc
	ShouldListenOnDefaultPorts func() bool
	wg                         sync.WaitGroup
}
type Container = *_Container

func NewContainer(getServices func() []Service, getHandleFunc func() HandleFunc, getPickSSLCertFunc func() PickSSLCertFunc, init ...func(container Container)) Container {
	container := &_Container{
		GetServices:        getServices,
		GetHandleFunc:      getHandleFunc,
		GetPickSSLCertFunc: getPickSSLCertFunc,
	}
	if len(init) > 0 {
		init[0](container)
	}
	return container
}

func (c Container) await() {
	c.wg.Wait()
}

func (c Container) goHttp(service Service, pickSSLCertFunc PickSSLCertFunc, handler http.Handler) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if service.UseGzip {
			handler = &gzip.Handler{Handler: handler}
		}
		var e error
		server := http.Server{
			//这里注意啦：
			//
			//请求中一般不会包含大量数据，因此可以设置较小的请求读取超时
			//
			//但是，回复中可能会包含大量数据，因此必须设置较大的回复写入超时
			WriteTimeout: 120 * time.Second,
			ReadTimeout:  30 * time.Second,
			Handler:      handler,
		}
		var listener net.Listener
		var tag string
		netAddr := service.Network + " " + service.Address
		if service.UseTLS && pickSSLCertFunc != nil {
			tag = "【HTTPS】"
			listener, e = tls.Listen(service.Network, service.Address, &tls.Config{GetCertificate: pickSSLCertFunc})
		} else {
			if service.UseTLS {
				log.W("【警告】没有 SSL 证书，【", netAddr, "】上的 SSL/TLS 功能不会被启用")
			}
			tag = "【HTTP】"
			listener, e = net.Listen(service.Network, service.Address)
		}
		if e == nil {
			log.S(tag, netAddr, " √")
			e = server.Serve(listener)
		} else {
			log.E(tag, netAddr, " ×")
		}
		log.E(e)
	}()
}

func (c Container) Boot() {
	{ // 设置最大文件数
		setrlimit()
	}
	handler := http.NewServeMux()
	var hf HandleFunc
	if c.GetHandleFunc != nil {
		hf = c.GetHandleFunc()
	}
	if hf != nil {
		handler.HandleFunc("/", hf)
	}
	var pickSSL PickSSLCertFunc
	if c.GetPickSSLCertFunc != nil {
		pickSSL = c.GetPickSSLCertFunc()
	}
	if c.GetServices != nil {
		for _, service := range c.GetServices() {
			c.goHttp(service, pickSSL, handler)
		}
	}
	if c.ShouldListenOnDefaultPorts == nil || c.ShouldListenOnDefaultPorts() {
		httpHandler := handler
		if pickSSL != nil {
			httpHandler = http.NewServeMux()
			httpHandler.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
				httpUtil.SetLocation(writer, "https://"+request.Host+request.RequestURI)
				writer.WriteHeader(http.StatusTemporaryRedirect)
			})
		}
		// HTTP
		c.goHttp(Service{Network: "tcp4", Address: "0.0.0.0:80", UseTLS: false, UseGzip: true}, pickSSL, httpHandler)
		c.goHttp(Service{Network: "tcp6", Address: "[::]:80", UseTLS: false, UseGzip: true}, pickSSL, httpHandler)
		// HTTPS
		c.goHttp(Service{Network: "tcp4", Address: "0.0.0.0:443", UseTLS: true, UseGzip: true}, pickSSL, handler)
		c.goHttp(Service{Network: "tcp6", Address: "[::]:443", UseTLS: true, UseGzip: true}, pickSSL, handler)
	}
	c.await()
}
