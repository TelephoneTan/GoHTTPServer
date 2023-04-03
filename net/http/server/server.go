package base

import (
	"github.com/TelephoneTan/GoHTTPServer/util"
	"github.com/TelephoneTan/GoLog/log"
	"math/rand"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type _Server struct {
	GetRoot         func(*http.Request) string
	Guard           func(http.ResponseWriter, *http.Request, PathPack) bool
	GetRootRelative func(*http.Request) string
	nodes           []_ResourceManager
}

type Server = *_Server

func defaultRoot(_ *http.Request) string {
	return "data-" + strconv.FormatInt(time.Now().UnixMilli(), 10) + strconv.Itoa(rand.Int())
}

func defaultRootRelative(_ *http.Request) string {
	return "root-" + strconv.Itoa(rand.Int())
}

func NewServer(getRoot func(*http.Request) string, init ...func(Server)) Server {
	if getRoot == nil {
		getRoot = defaultRoot
	}
	return util.New(&_Server{
		GetRoot: getRoot,
	}, init...)
}

func (s Server) Handle(w http.ResponseWriter, r *http.Request) {
	normalReturn := false
	defer func() {
		if !normalReturn {
			panicArgument := recover()
			log.EF(
				"\n发生了错误：%v\n"+
					"\n======================================\n"+
					"\n%s\n"+
					"\n======================================\n",
				panicArgument,
				debug.Stack(),
			)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()
	s.handle(w, r)
	normalReturn = true
}

func (s Server) handle(w http.ResponseWriter, r *http.Request) {
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
				return
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
			if s.Guard != nil && s.Guard(w, r, paths) {
				return
			}
			// 匹配子节点
			token := paths.SuffixPath[0]
			for _, handler := range s.nodes {
				if handler.GetWordList().Match(token) {
					handler.Handle(w, r, paths, nil)
					return
				}
			}
			// 文件服务器
			getRootRelative := defaultRootRelative
			if s.GetRootRelative != nil {
				getRootRelative = s.GetRootRelative
			}
			HandleFile(
				w,
				r,
				util.JoinPath(s.GetRoot(r), util.JoinPath(append([]string{getRootRelative(r)}, paths.SuffixPath...)...)),
				false,
			)
		}
	}
}
