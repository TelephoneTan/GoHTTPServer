package server

import (
	"fmt"
	"github.com/TelephoneTan/GoHTTPServer/net/http/header"
	"github.com/TelephoneTan/GoHTTPServer/net/http/method"
	"github.com/TelephoneTan/GoHTTPServer/util"
	httpUtil "github.com/TelephoneTan/GoHTTPServer/util/http"
	"github.com/TelephoneTan/GoLog/log"
	"golang.org/x/net/idna"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type HostPack struct {
	Host     string
	HostPort *uint16
	IP       net.IP
	IPPort   *uint16
}

type _Server struct {
	GetRoot           func(HostPack) string
	GetRootRelative   func(HostPack) string
	GetHosts          func() []string
	GetHostPorts      func() []uint16
	GetIPs            func() []net.IP
	GetIPPorts        func() []uint16
	Guard             func(http.ResponseWriter, *http.Request, *PathPack) bool
	HasRootFileServer func() bool
	GetCDNHost        func() string
	GetCDNOriginHost  func() string
	nodes             []ResourceManagerI
}

type Server = *_Server

func defaultRoot(_ HostPack) string {
	return "data-" + strconv.FormatInt(time.Now().UnixMilli(), 10) + strconv.Itoa(rand.Int())
}

func defaultRootRelative(_ HostPack) string {
	return "root-" + strconv.Itoa(rand.Int())
}

func NewServer(getRoot func(HostPack) string, getRootRelative func(HostPack) string, init ...func(Server)) Server {
	if getRoot == nil {
		getRoot = defaultRoot
	}
	if getRootRelative == nil {
		getRootRelative = defaultRootRelative
	}
	return util.New(&_Server{
		GetRoot:         getRoot,
		GetRootRelative: getRootRelative,
	}, init...)
}

func (s Server) Use(child ...ResourceManagerI) Server {
	s.nodes = append(s.nodes, child...)
	return s
}

func extractHost(s string) string {
	if s[0] == '[' {
		return s[:strings.IndexRune(s, ']')+1]
	} else if colonIndex := strings.IndexRune(s, ':'); colonIndex != -1 {
		return s[:colonIndex]
	} else {
		return s
	}
}

func extractPort(s string) (portNum *uint16, portStr string) {
	if s[0] == '[' {
		s = s[strings.IndexRune(s, ']')+1:]
	}
	if colonIndex := strings.IndexRune(s, ':'); colonIndex != -1 {
		portStr = s[colonIndex+1:]
		if port, err := strconv.ParseUint(portStr, 10, 16); err == nil {
			port16 := uint16(port)
			portNum = &port16
		}
	}
	return portNum, portStr
}

func getIPPort(r *http.Request) (ip net.IP, port *uint16) {
	return util.AddrToIPPort(r.Context().Value(http.LocalAddrContextKey).(net.Addr))
}

func getHostPort(r *http.Request) (host string, port *uint16) {
	host = extractHost(r.Host)
	port, _ = extractPort(r.Host)
	return host, port
}

func getHostInfo(r *http.Request) HostPack {
	ip, ipPort := getIPPort(r)
	host, hostPort := getHostPort(r)
	if hostPort == nil {
		hostPort = ipPort
	}
	return HostPack{
		Host:     host,
		HostPort: hostPort,
		IP:       ip,
		IPPort:   ipPort,
	}
}

func withoutBrackets(s string) string {
	if s[0] == '[' {
		if tailIndex := strings.IndexRune(s, ']'); tailIndex != -1 {
			s = s[1:tailIndex]
		}
	}
	return s
}

func matchHost(host string, getValidHosts func() []string) bool {
	if getValidHosts == nil {
		return true
	}
	if host == "" {
		return false
	}
	host, err := idna.ToUnicode(host)
	if err != nil {
		return false
	}
	host = withoutBrackets(host)
	validHosts := getValidHosts()
	for _, validHost := range validHosts {
		validHost, err = idna.ToUnicode(validHost)
		if err == nil && strings.EqualFold(host, withoutBrackets(validHost)) {
			return true
		}
	}
	return false
}

func matchIP(ip net.IP, getValidIPs func() []net.IP) bool {
	if getValidIPs == nil {
		return true
	}
	if ip == nil {
		return false
	}
	validIPs := getValidIPs()
	for _, validIP := range validIPs {
		if validIP.Equal(ip) {
			return true
		}
	}
	return false
}

func matchPort(port *uint16, getValidPorts func() []uint16) bool {
	if getValidPorts == nil {
		return true
	}
	if port == nil {
		return false
	}
	validPorts := getValidPorts()
	for _, validPort := range validPorts {
		if *port == validPort {
			return true
		}
	}
	return false
}

func (s Server) match(hostInfo HostPack) bool {
	return matchHost(hostInfo.Host, s.GetHosts) &&
		matchPort(hostInfo.HostPort, s.GetHostPorts) &&
		matchIP(hostInfo.IP, s.GetIPs) &&
		matchPort(hostInfo.IPPort, s.GetIPPorts)
}

func HandlePanic(w http.ResponseWriter, r *http.Request, panicArgument any) {
	var id string
	var statusCode int
	var stackTrace []byte
	switch e := panicArgument.(type) {
	case Exception:
		e.SetHeader(w)
		w.Header().Add("Reason", mime.QEncoding.Encode("utf-8", e.TipZH()))
		w.Header().Add("Error-Code", mime.QEncoding.Encode("utf-8", e.Code()))
		w.Header().Add("Error-ID", mime.QEncoding.Encode("utf-8", e.ID()))
		statusCode = e.HTTPCode()
		stackTrace = []byte(e.StackTrace())
	default:
		id = fmt.Sprintf("E#%v#%v ", time.Now().UnixNano(), rand.Int())
		w.Header().Add("Error-ID", id)
		statusCode = http.StatusInternalServerError
		stackTrace = debug.Stack()
	}
	if r.Method != "HEAD" {
		w.Header().Set("Content-Length", "0")
	}
	w.WriteHeader(statusCode)
	log.EF(
		"\n发生了错误：%v%v\n"+
			"\n======================================\n"+
			"\n%s\n"+
			"\n======================================\n",
		id,
		panicArgument,
		stackTrace,
	)
}

func (s Server) Handle(w http.ResponseWriter, r *http.Request) (handled bool) {
	normal := false
	goto start
end:
	normal = true
	return handled
handle:
	handled = true
	goto end
notHandle:
	handled = false
	goto end
start:
	defer func() {
		if !normal {
			HandlePanic(w, r, recover())
			handled = true
		}
	}()
	hostInfo := getHostInfo(r)
	if !s.match(hostInfo) {
		goto notHandle
	}
	if !s.handle(w, r, hostInfo) {
		goto notHandle
	}
	goto handle
}

func (s Server) toCDN(w http.ResponseWriter, r *http.Request) bool {
	var cdnHost, cdnOriginHost string
	if s.GetCDNHost != nil {
		cdnHost = s.GetCDNHost()
	}
	if s.GetCDNOriginHost != nil {
		cdnOriginHost = s.GetCDNOriginHost()
	}
	if cdnHost != "" && cdnOriginHost != "" {
		clientHost, _ := idna.ToASCII(r.Host)
		cdnOriginHost, _ := idna.ToASCII(cdnOriginHost)
		if !strings.EqualFold(clientHost, cdnOriginHost) {
			cdnHost, _ := idna.ToASCII(cdnHost)
			w.Header().Set("Content-Length", "0")
			w.Header().Set("Location", "//"+cdnHost+r.RequestURI)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return true
		}
	}
	return false
}

func (s Server) HandleFile(w http.ResponseWriter, r *http.Request, filePath string, privateOrNoCDN bool) {
	tag := "FileServer"
	fileInfo, err := os.Stat(filePath)
	ff, err2 := os.Open(filePath)
	if err2 == nil { // 如果成功打开了文件，记得关闭文件
		defer func() {
			_ = ff.Close()
		}()
	}
	if err != nil {
		log.EF("%s : 发生了错误 (%v) 在文件 (%s) 上", tag, err, filePath)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err2 != nil {
		log.EF("%s : 发生了错误 (%v) 在文件 (%s) 上", tag, err2, filePath)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch method.Parse(r.Method) {
	// 方法查询
	case method.OPTIONS:
		httpUtil.SetAllow(w,
			// 方法查询
			method.OPTIONS,
			// 查
			method.GET,
			// 头部预览
			method.HEAD,
		)
		w.WriteHeader(http.StatusOK)
	case
		// 头部预览
		method.HEAD,
		// 查
		method.GET:
		if fileInfo.IsDir() {
			log.WF("%s : 文件 (%s) 是个目录", tag, filePath)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if privateOrNoCDN {
			// 私有缓存
			w.Header().Set(header.CacheControl, "private")
		} else {
			if s.toCDN(w, r) {
				return
			}
			// 2023年9月13日15点37分 之前：
			//   缓存，但是每次都要验证
			// 2023年9月13日15点37分 之后：
			//   由于阿里云 CDN 和腾讯云 CDN 不遵循 no-cache 规范，改为使用 max-age 并指定尽可能长的缓存周期
			//   max-age 不会导致需要身份认证才能访问的请求回复被缓存在公有缓存上
			httpUtil.CacheForever(w)
		}
		http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), ff)
	// 不支持其他方法
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s Server) handle(w http.ResponseWriter, r *http.Request, hostInfo HostPack) bool {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/") { // 确保路径以 '/' 开头，否则路径分割会不一致
		path = "/" + path
	}
	if len(path) > 1 && strings.HasSuffix(path, "/") { // 确保路径不以 '/' 结尾，否则路径分割会不一致
		pathRS := []rune(path)
		pathLen := len(pathRS)
		path = string(pathRS[:pathLen-1])
	}
	subPath := strings.Split(path, "/")[1:]
	{ // 请求不允许包含相对路径
		for _, p := range subPath {
			if p == "." || p == ".." {
				log.W("request contains relative path")
				w.WriteHeader(http.StatusBadRequest)
				return true
			}
		}
	}
	{ // 【注意】比较路径时不要区分大小写
		// 资源路径，例如：http://www.x.com/a/b -> path：["a", "b"]
		path := subPath
		// 当前处理节点之前的路径（包括当前节点），例如：http://www.x.com/a/b + 正在处理 b 节点 -> prefixPath：["a", "b"]
		prefixPath := subPath[:1]
		// 当前处理节点之后的路径（包括当前节点），例如：http://www.x.com/a/b + 正在处理 b 节点 -> suffixPath：["b"]
		suffixPath := subPath
		paths := PathPack{
			Path:       path,
			PrefixPath: prefixPath,
			SuffixPath: suffixPath,
		}
		{
			// 守卫优先
			if s.Guard != nil && s.Guard(w, r, &paths) {
				return true
			}
			// 匹配子节点
			token := paths.SuffixPath[0]
			for _, handler := range s.nodes {
				if handler.WordList().Match(token) {
					handler.Handle(w, r, hostInfo, paths, s, nil)
					return true
				}
			}
			if s.HasRootFileServer != nil && s.HasRootFileServer() {
				// 文件服务器
				s.HandleFile(
					w,
					r,
					util.JoinPath(
						s.GetRoot(hostInfo),
						util.JoinPath(append([]string{s.GetRootRelative(hostInfo)}, paths.SuffixPath...)...),
					),
					false,
				)
				return true
			} else {
				return false
			}
		}
	}
}
