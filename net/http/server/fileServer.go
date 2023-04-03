package server

import (
	"github.com/TelephoneTan/GoHTTPServer/net/http/header"
	"github.com/TelephoneTan/GoHTTPServer/net/http/method"
	httpUtil "github.com/TelephoneTan/GoHTTPServer/util/http"
	"github.com/TelephoneTan/GoLog/log"
	httpGo "net/http"
	"os"
)

func HandleFile(w httpGo.ResponseWriter, r *httpGo.Request, filePath string, isPrivateFile bool) {
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
		w.WriteHeader(httpGo.StatusNotFound)
		return
	}
	if err2 != nil {
		log.EF("%s : 发生了错误 (%v) 在文件 (%s) 上", tag, err2, filePath)
		w.WriteHeader(httpGo.StatusNotFound)
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
		w.WriteHeader(httpGo.StatusOK)
	case
		// 头部预览
		method.HEAD,
		// 查
		method.GET:
		if fileInfo.IsDir() {
			log.WF("%s : 文件 (%s) 是个目录", tag, filePath)
			w.WriteHeader(httpGo.StatusBadRequest)
			return
		}
		if isPrivateFile {
			// 私有缓存
			w.Header().Set(header.CacheControl, "private")
		} else {
			// 缓存，但是每次都要验证
			w.Header().Set(header.CacheControl, "no-cache")
		}
		httpGo.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), ff)
	// 不支持其他方法
	default:
		w.WriteHeader(httpGo.StatusMethodNotAllowed)
	}
}
