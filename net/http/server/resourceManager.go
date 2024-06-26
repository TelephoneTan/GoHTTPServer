package server

import (
	"github.com/TelephoneTan/GoHTTPServer/net/http/method"
	"github.com/TelephoneTan/GoHTTPServer/types"
	"github.com/TelephoneTan/GoHTTPServer/util"
	httpUtil "github.com/TelephoneTan/GoHTTPServer/util/http"
	"github.com/TelephoneTan/GoLog/log"
	"golang.org/x/net/idna"
	"net/http"
	"strings"
)

type ResourceRequestHandler[PACK any] struct {
	// 用于解析请求并决定是否要拦截请求
	Peek func(r *http.Request, paths PathPack) (pack PACK, hijacked bool)
	// 如果请求被拦截，此函数会被调用用于作出回复
	Reply func(w http.ResponseWriter, toCDN func() bool, pack PACK)
	// 用于统计业务，此函数相比于 ResourceManager.Record 有如下差异：
	//
	// * Monitor 能够拿到请求解析结果用于详细分析
	//
	// * 请求被自动重定向时 ResourceManager.Record 仍会被调用但 Monitor 不会
	//
	// 当请求使用的是 OPTIONS 方法时，参数 hijacked 表示该请求是否可以使用除 OPTIONS 方法之外的其他方法。
	//
	// 当请求使用的是非 OPTIONS 方法时，参数 hijacked 表示该请求是否会被拦截。
	Monitor func(pack PACK, hijacked bool)
}

type ResourceManagerI interface {
	WordList() *types.WordList
	Handle(w http.ResponseWriter, r *http.Request, hostInfo HostPack, paths PathPack, server Server, relativeRootDirList []string)
}

type _ResourceManager[PACK any] struct {
	GetWordList         func() types.WordList
	GetRelativeRootDir  func() string
	GetHomepageFileName func() string
	// 用于决定该请求是否需要自动重定向，以及如果需要的话，提供自动重定向的状态码和 Location
	GetRedirect func(r *http.Request, paths PathPack) (redirect bool, statusCode int, location string)
	// 用于记录请求，所有情况下均会被调用
	Record           func(r *http.Request, paths PathPack)
	Guide            map[method.Method]ResourceRequestHandler[PACK]
	CORSAllowOrigins func() []string
	nodes            []ResourceManagerI
}

type ResourceManager[PACK any] struct {
	*_ResourceManager[PACK]
}

func NewResourceManager[PACK any](getWordList func() types.WordList, getRelativeRootDir func() string, init ...func(b ResourceManager[PACK])) ResourceManager[PACK] {
	return ResourceManager[PACK]{
		util.New(&_ResourceManager[PACK]{
			GetWordList:        getWordList,
			GetRelativeRootDir: getRelativeRootDir,
		}, func(i *_ResourceManager[PACK]) {
			if len(init) > 0 {
				init[0](ResourceManager[PACK]{i})
			}
		}),
	}
}

func (rm ResourceManager[PACK]) Use(child ...ResourceManagerI) ResourceManager[PACK] {
	rm.nodes = append(rm.nodes, child...)
	return rm
}

func (rm ResourceManager[PACK]) getRelativeRootDir() (relativeRootDir string) {
	goto start
end:
	return relativeRootDir
start:
	if rm.GetRelativeRootDir != nil {
		relativeRootDir = rm.GetRelativeRootDir()
	}
	goto end
}

func (rm ResourceManager[PACK]) getRootDir(hostInfo HostPack, server Server, relativeRootDirList []string) string {
	relativeDir := util.JoinPath(util.JoinPath(relativeRootDirList...), rm.getRelativeRootDir())
	if relativeDir == "" {
		relativeDir = "."
	}
	root := server.GetRoot(hostInfo)
	return util.AppendPathDelimiter(util.JoinPath(root, relativeDir))
}

func (rm ResourceManager[PACK]) getHomepageFileName() (homepageFileName string) {
	goto start
end:
	return homepageFileName
start:
	if rm.GetHomepageFileName != nil {
		homepageFileName = rm.GetHomepageFileName()
	}
	goto end
}

// 计算用于处理该请求的 Handler
func (rm ResourceManager[PACK]) calHandler(r *http.Request) *ResourceRequestHandler[PACK] {
	if h, has := rm.Guide[method.Parse(r.Method)]; has {
		return &h
	}
	return nil
}

// 计算用于处理该请求的动作
func (rm ResourceManager[PACK]) calActions(s Server, r *http.Request, w http.ResponseWriter, paths PathPack) (
	hijacked bool,
	reply func(),
	monitor func(hijacked bool),
	record func(),
	redirect func(),
) {
	goto start
end:
	return hijacked, reply, monitor, record, redirect
start:
	if rm.Record != nil {
		record = func() {
			rm.Record(r, paths.Clone())
		}
	}
	if rm.GetRedirect != nil {
		redirected, statusCode, location := rm.GetRedirect(r, paths.Clone())
		if redirected {
			redirect = func() {
				httpUtil.SetLocation(w, location)
				w.WriteHeader(statusCode)
			}
			goto end
		}
	}
	handler := rm.calHandler(r)
	if handler == nil {
		goto end
	}
	var pack PACK
	if handler.Peek != nil {
		pack, hijacked = handler.Peek(r, paths.Clone())
	} else {
		hijacked = true
	}
	hijacked = hijacked && handler.Reply != nil
	if hijacked {
		reply = func() {
			handler.Reply(w, func() bool {
				return s.toCDN(w, r)
			}, pack)
		}
	}
	if handler.Monitor != nil {
		monitor = func(hijacked bool) {
			handler.Monitor(pack, hijacked)
		}
	}
	goto end
}

func retrieveHostName(origin string) string {
	byColon := strings.Split(origin, ":")
	if len(byColon) < 2 {
		return ""
	}
	origin = byColon[1]
	bySlash := strings.Split(origin, "/")
	if len(bySlash) < 3 {
		return ""
	}
	origin = bySlash[2]
	var err error
	origin, err = idna.ToASCII(origin)
	if err != nil {
		return ""
	}
	return origin
}

func retrieveScheme(origin string) string {
	byColon := strings.Split(origin, ":")
	if len(byColon) < 2 {
		return ""
	}
	return byColon[0]
}

func retrievePort(origin string) string {
	byColon := strings.Split(origin, ":")
	if len(byColon) < 3 {
		return ""
	}
	return byColon[2]
}

func matchOrigin(origin1 string, origin2 string) bool {
	return strings.EqualFold(retrieveScheme(origin1), retrieveScheme(origin2)) &&
		strings.EqualFold(retrieveHostName(origin1), retrieveHostName(origin2)) &&
		strings.EqualFold(retrievePort(origin1), retrievePort(origin2))
}

// 处理请求
func (rm ResourceManager[PACK]) handle(s Server, r *http.Request, w http.ResponseWriter, paths PathPack) (hijacked bool) {
	goto start
end:
	return hijacked
start:
	hijacked, reply, monitor, record, redirect := rm.calActions(s, r, w, paths)
	if record != nil {
		record()
	}
	if redirect != nil {
		redirect()
		hijacked = true
		goto end
	}
	supportedMethodList := []method.Method{
		method.OPTIONS,
	}
	if method.Parse(r.Method) == method.OPTIONS { // OPTIONS 方法的处理比较特殊，会改写 hijacked、reply
		for m := range rm.Guide {
			if m != method.OPTIONS {
				func() {
					originalMethod := r.Method
					r.Method = m.String()
					defer func() { r.Method = originalMethod }()
					if hijacked, _, _, _, _ := rm.calActions(s, r, w, paths); hijacked {
						supportedMethodList = append(supportedMethodList, m)
					}
				}()
			}
		}
		hijacked = len(supportedMethodList) > 1
		if hijacked {
			reply = func() {
				httpUtil.SetAllow(w, supportedMethodList...)
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}
	if monitor != nil {
		monitor(hijacked)
	}
	if hijacked {
		var corsOrigins []string
		if rm.CORSAllowOrigins != nil {
			corsOrigins = rm.CORSAllowOrigins()
		}
		var origin string
		if len(corsOrigins) > 0 {
			origin = r.Header.Get("Origin")
		}
		if origin != "" {
			corsMethod := r.Header.Get("Access-Control-Request-Method")
			corsHeaders := r.Header.Get("Access-Control-Request-Headers")
			for _, allow := range corsOrigins {
				if !matchOrigin(allow, origin) {
					continue
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Expose-Headers", "*")
				if corsMethod != "" {
					httpUtil.SetAccessControlAllowMethods(w, supportedMethodList...)
					if corsHeaders != "" {
						w.Header().Set("Access-Control-Allow-Headers", corsHeaders)
					}
					w.Header().Set("Access-Control-Max-Age", "0")
				}
			}
		}
		reply()
	}
	goto end
}

func (rm ResourceManager[PACK]) Handle(
	w http.ResponseWriter,
	r *http.Request,
	hostInfo HostPack,
	paths PathPack,
	server Server,
	relativeRootDirList []string,
) {
	if len(paths.SuffixPath) < 1 {
		log.E("len(SuffixPath) < 1")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hijacked := rm.handle(server, r, w, paths)
	if hijacked {
		return
	}
	filePath := rm.getRootDir(hostInfo, server, relativeRootDirList)
	var noCDN bool
	if len(paths.SuffixPath) == 1 {
		filePath = util.JoinPath(filePath, rm.getHomepageFileName())
		noCDN = true
	} else {
		paths.PrefixPath = paths.PrefixPath[:len(paths.PrefixPath)+1]
		paths.SuffixPath = paths.SuffixPath[1:]
		for _, manager := range rm.nodes {
			if manager.WordList().Match(paths.SuffixPath[0]) {
				manager.Handle(w, r, hostInfo, paths, server, append(relativeRootDirList, rm.getRelativeRootDir()))
				return
			}
		}
		nodes := make([]string, 0, len(paths.SuffixPath)+1)
		nodes = append(nodes, filePath)
		nodes = append(nodes, paths.SuffixPath...)
		filePath = util.JoinPath(nodes...)
		noCDN = false
	}
	server.HandleFile(w, r, filePath, noCDN)
}

func (rm ResourceManager[PACK]) WordList() *types.WordList {
	wl := rm.GetWordList()
	return &wl
}
