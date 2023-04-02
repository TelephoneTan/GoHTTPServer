package base

import (
	"github.com/TelephoneTan/GoHTTPServer/net/http/method"
	"github.com/TelephoneTan/GoHTTPServer/types"
	"github.com/TelephoneTan/GoHTTPServer/util"
	httpUtil "github.com/TelephoneTan/GoHTTPServer/util/http"
	"github.com/TelephoneTan/GoLog/log"
	"net/http"
)

type ResourceRequestHandler[PACK any] struct {
	// 用于解析请求并决定是否要拦截请求
	Peek func(r *http.Request, paths PathPack) (pack PACK, hijacked bool)
	// 如果请求被拦截，此函数会被调用用于作出回复
	Reply func(w http.ResponseWriter, pack PACK)
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

type _ResourceManager interface {
	GetWordList() *types.WordList
	Handle(w http.ResponseWriter, r *http.Request, paths PathPack, relativeRootDirList []string)
}

type ResourceManager[PACK any] struct {
	server              Server
	WordList            types.WordList
	GetRelativeRootDir  func() string
	GetHomepageFileName func() string
	// 用于决定该请求是否需要自动重定向，以及如果需要的话，提供自动重定向的状态码和 Location
	GetRedirect func(r *http.Request, paths PathPack) (redirect bool, statusCode int, location string)
	// 用于记录请求，所有情况下均会被调用
	Record func(r *http.Request, paths PathPack)
	Guide  map[method.Method]*ResourceRequestHandler[PACK]
	nodes  []_ResourceManager
}

func NewResourceManager[PACK any](server Server, init ...func(*ResourceManager[PACK])) *ResourceManager[PACK] {
	rm := util.New(&ResourceManager[PACK]{
		server: server,
	}, init...)
	server.nodes = append(server.nodes, rm)
	return rm
}

func NewChildResourceManager[PackParent, PackChild any](parent *ResourceManager[PackParent], init ...func(*ResourceManager[PackChild])) *ResourceManager[PackChild] {
	rm := util.New(&ResourceManager[PackChild]{
		server: parent.server,
	}, init...)
	parent.nodes = append(parent.nodes, rm)
	return rm
}

func (rm *ResourceManager[PACK]) getRelativeRootDir() (relativeRootDir string) {
	goto start
end:
	return relativeRootDir
start:
	if rm.GetRelativeRootDir != nil {
		relativeRootDir = rm.GetRelativeRootDir()
	}
	goto end
}

func (rm *ResourceManager[PACK]) getRootDir(r *http.Request, relativeRootDirList []string) string {
	relativeDir := util.JoinPath(util.JoinPath(relativeRootDirList...), rm.getRelativeRootDir())
	if relativeDir == "" {
		relativeDir = "."
	}
	root := rm.server.GetRoot(r)
	return util.AppendPathDelimiter(util.JoinPath(root, relativeDir))
}

func (rm *ResourceManager[PACK]) getHomepageFileName() (homepageFileName string) {
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
func (rm *ResourceManager[PACK]) calHandler(r *http.Request, paths PathPack) *ResourceRequestHandler[PACK] {
	if h, has := rm.Guide[method.Parse(r.Method)]; has {
		return h
	}
	return nil
}

// 计算用于处理该请求的动作
func (rm *ResourceManager[PACK]) calActions(r *http.Request, w http.ResponseWriter, paths PathPack) (
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
	handler := rm.calHandler(r, paths)
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
			handler.Reply(w, pack)
		}
	}
	if handler.Monitor != nil {
		monitor = func(hijacked bool) {
			handler.Monitor(pack, hijacked)
		}
	}
	goto end
}

// 处理请求
func (rm *ResourceManager[PACK]) handle(r *http.Request, w http.ResponseWriter, paths PathPack) (hijacked bool) {
	goto start
end:
	return hijacked
start:
	hijacked, reply, monitor, record, redirect := rm.calActions(r, w, paths)
	if record != nil {
		record()
	}
	if redirect != nil {
		redirect()
		hijacked = true
		goto end
	}
	if method.Parse(r.Method) == method.OPTIONS { // OPTIONS 方法的处理比较特殊，会改写 hijacked、reply
		supportedMethodList := []method.Method{
			method.OPTIONS,
		}
		for m := range rm.Guide {
			if m != method.OPTIONS {
				func() {
					originalMethod := r.Method
					r.Method = m.String()
					defer func() { r.Method = originalMethod }()
					if hijacked, _, _, _, _ := rm.calActions(r, w, paths); hijacked {
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
		reply()
	}
	goto end
}

func (rm *ResourceManager[PACK]) Handle(
	w http.ResponseWriter,
	r *http.Request,
	paths PathPack,
	relativeRootDirList []string,
) {
	if len(paths.SuffixPath) < 1 {
		log.E("len(SuffixPath) < 1")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hijacked := rm.handle(r, w, paths)
	if hijacked {
		return
	}
	filePath := rm.getRootDir(r, relativeRootDirList)
	if len(paths.SuffixPath) == 1 {
		filePath = util.JoinPath(filePath, rm.getHomepageFileName())
	} else {
		paths.PrefixPath = paths.PrefixPath[:len(paths.PrefixPath)+1]
		paths.SuffixPath = paths.SuffixPath[1:]
		for _, manager := range rm.nodes {
			if manager.GetWordList().Match(paths.SuffixPath[0]) {
				manager.Handle(w, r, paths, append(relativeRootDirList, rm.getRelativeRootDir()))
				return
			}
		}
		nodes := make([]string, 0, len(paths.SuffixPath)+1)
		nodes = append(nodes, filePath)
		nodes = append(nodes, paths.SuffixPath...)
		filePath = util.JoinPath(nodes...)
	}
	HandleFile(w, r, filePath, false)
}

func (rm *ResourceManager[PACK]) GetWordList() *types.WordList {
	return &rm.WordList
}
