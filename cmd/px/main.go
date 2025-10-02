package main

import (
	"flag"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/log"
	"github.com/legiz-ru/prizrak-box/prizrak"
	sys "github.com/legiz-ru/prizrak-box/pkg/sys/proxy"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"go.uber.org/automaxprocs/maxprocs"
	"net/url"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	// 优化线程资源配置
	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	// 回调地址
	addr := flag.String("addr", "", "callback address")
	home := flag.String("home", "", "home directory")

	// 解析命令行参数
	flag.Parse()

	if addr == nil || *addr == "" {
		panic("callback address is required")
	}

	if home == nil || *home == "" {
		panic("home directory is required")
	}

	homeDir, err := url.QueryUnescape(*home)
	if err != nil {
		panic(err)
	}

	// 设置工作目录
	utils.InitHomeDir(homeDir)

	// 保持单例
	if utils.NotSingleton("px-server.pid") {
		os.Exit(1)
	}

	// 初始化工作目录
	prizrak.Init()

	// 开启后端api
	prizrak.StartCore(*addr)

	termSign := make(chan os.Signal, 1)
	signal.Notify(termSign, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-termSign:
		log.Warnln("received termination signal")
		prizrak.Release()
		utils.UnlockSingleton()
		executor.Shutdown()
		sys.DisableProxy()
	}

}
