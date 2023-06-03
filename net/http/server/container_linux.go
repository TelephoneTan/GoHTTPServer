package server

import (
	"github.com/TelephoneTan/GoLog/log"
	"syscall"
)

func setrlimit() {
	limit := 1000000
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{Cur: uint64(limit), Max: uint64(limit)}); err != nil {
		log.FF("尝试设置 最大文件数 限制为 (%d) 失败", limit)
	} else {
		log.SF("已将 最大文件数 限制设为：%d", limit)
	}
}
