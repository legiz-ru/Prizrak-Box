package prizrak

import (
	"fmt"
	"github.com/metacubex/mihomo/hub/route"
	"github.com/metacubex/mihomo/log"
	"github.com/legiz-ru/prizrak-box/api"
	"github.com/legiz-ru/prizrak-box/api/handlers"
	"github.com/legiz-ru/prizrak-box/api/job"
	"github.com/legiz-ru/prizrak-box/internal"
	"github.com/legiz-ru/prizrak-box/pkg/cache"
	"github.com/legiz-ru/prizrak-box/pkg/constant"
	"github.com/legiz-ru/prizrak-box/pkg/cron"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"time"
)

func Init() {
	internal.Init()
}

func Release() {
	cache.Close()
}

func StartCore(server string) (port int, secret string) {

	route.Register(handlers.Profile)
	route.Register(handlers.WebTest)
	route.Register(handlers.Rule)
	route.Register(handlers.DNS)
	route.Register(handlers.Mihomo)
	route.Register(handlers.Prizrak)
	route.Register(handlers.Age)

	// 设置地址
	host := "127.0.0.1"

	// 获取端口
	if utils.IsPortAvailable(host, 9686) == nil {
		port = 9686
	} else {
		port, _ = utils.GetRandomPort(host)
	}
	api.ControllerPort = port

	// 获取密钥
	_ = cache.Get(constant.SecretKey, &secret)
	if secret == "" {
		secret = utils.RandString(16)
		_ = cache.Put(constant.SecretKey, secret)
	}

	cors := route.Cors{AllowOrigins: []string{"*"}, AllowPrivateNetwork: true}
	route.StartByPandoraBox(host, port, secret, cors)
	log.Infoln("Routing startup completed")

	// 开启mihomo
	internal.SwitchProfile(false)

	// 进行回调
	if server != "" {
		callbackUrl := fmt.Sprintf("http://%s/pxStore?port=%v&secret=%v", server, port, secret)
		for {
			log.Infoln("向地址发送数据：%s", callbackUrl)
			body, _, err := utils.SendGet(callbackUrl, map[string]string{}, "")
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if body == "ok" {
				break
			}
		}
	}

	// Restore EnableHWID from cache before starting cron jobs.
	// gocron fires tasks immediately on StartAsync(), so the HTTP client config
	// must reflect the persisted user preference before DoRefresh() runs.
	var enableHWID bool
	if err := cache.Get(constant.EnableHWIDKey, &enableHWID); err != nil {
		// No persisted value yet — default to true (matches frontend default).
		enableHWID = true
	}
	utils.UpdateHTTPClientConfig(&utils.HTTPClientConfig{EnableHWID: enableHWID})

	// 开启定时任务
	job.LogJob("px-server.log")
	job.RefreshJob()
	job.AliveJob("alive", server)
	go cron.Start()

	return port, secret
}
