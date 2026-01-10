package handlers

import (
	"errors"
	"net/netip"
	"os"

	"github.com/legiz-ru/prizrak-box/api"
	"github.com/legiz-ru/prizrak-box/api/job"
	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/pkg/cache"
	"github.com/legiz-ru/prizrak-box/pkg/constant"
	sys "github.com/legiz-ru/prizrak-box/pkg/sys/proxy"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/chi"
	"github.com/metacubex/chi/render"
	"github.com/metacubex/http"
	"github.com/metacubex/mihomo/component/process"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel/statistic"
	"strings"
)

func Prizrak(r chi.Router) {
	r.Get("/version", getPrizrakVersion)
	r.Mount("/prizrak", PrizrakRouter())
}

func PrizrakRouter() chi.Router {
	r := chi.NewRouter()
	// 代理相关
	r.Put("/enableProxy", enableProxy)
	r.Get("/disableProxy", disableProxy)

	// 地址相关
	r.Put("/checkAddressPort", checkAddressPort)

	// 配置目录
	r.Get("/configDir", configDir)

	// 退出px
	r.Get("/exit", exitPx)

	// 更新HTTP客户端配置
	r.Put("/httpClientConfig", updateHTTPClientConfig)

	return r
}

func getPrizrakVersion(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, render.M{"version": api.Version})
}

func enableProxy(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	mi := struct {
		BindAddress string `json:"bindAddress"`
		Port        int    `json:"port"`
	}{}
	if err := render.DecodeJSON(r.Body, &mi); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	// 开启
	_ = sys.EnableProxy(mi.BindAddress, mi.Port)

	render.NoContent(w, r)
}

func disableProxy(w http.ResponseWriter, r *http.Request) {
	sys.DisableProxy()
	log.Warnln("System proxy disabled")
	if !executor.GetGeneral().Tun.Enable {
		statistic.DefaultManager.Range(func(c statistic.Tracker) bool {
			_ = c.Close()
			return true
		})
	}
	log.Warnln("All connections disconnected")
	render.NoContent(w, r)
}

func checkAddressPort(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	mi := struct {
		BindAddress string `json:"bindAddress"`
		MixedPort   int    `json:"port"`
	}{}
	if err := render.DecodeJSON(r.Body, &mi); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	// 检测到Px相同地址端口则跳过
	var mc models.Mihomo
	_ = cache.Get(constant.Mihomo, &mc)
	if mc.BindAddress == mi.BindAddress && mc.Port == mi.MixedPort {
		render.NoContent(w, r)
		return
	}

	// 增强校验，如果是px程序占用的，那直接返回
	addr, err := netip.ParseAddr(mi.BindAddress)
	if err != nil {
		ErrorResponse(w, r, errors.New("invalid address"))
		return
	}
	_, s, err := process.FindProcessName(process.TCP, addr, mi.MixedPort)
	if err == nil {
		if strings.HasSuffix(s, "px") || strings.HasSuffix(s, "px.exe") {
			if mc.BindAddress == mi.BindAddress && api.ControllerPort == mi.MixedPort {
				ErrorResponse(w, r, errors.New("invalid address or port"))
				return
			}
			render.NoContent(w, r)
			return
		}
	}

	// 检测地址端口是否可用
	err = utils.IsPortAvailable(mi.BindAddress, mi.MixedPort)
	if err != nil {
		ErrorResponse(w, r, err)
		return
	}

	render.NoContent(w, r)
}

func configDir(w http.ResponseWriter, r *http.Request) {
	render.PlainText(w, r, utils.GetUserHomeDir())
}

func exitPx(w http.ResponseWriter, r *http.Request) {
	job.Exit(false)
	render.PlainText(w, r, "ok")
	os.Exit(0)
}

func updateHTTPClientConfig(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	config := struct {
		EnableHWID  bool   `json:"enableHWID"`
		Version     string `json:"version"`
		DeviceOS    string `json:"deviceOS"`
		DeviceOSVer string `json:"deviceOSVer"`
		DeviceModel string `json:"deviceModel"`
	}{}
	if err := render.DecodeJSON(r.Body, &config); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	// 构建用户代理字符串
	userAgent := ""
	if config.EnableHWID && config.Version != "" {
		userAgent = "prizrak-box/" + config.Version
	} else {
		userAgent = "clash-verge/v2.3.0"
	}

	// 更新配置
	httpConfig := &utils.HTTPClientConfig{
		EnableHWID:  config.EnableHWID,
		Version:     config.Version,
		DeviceOS:    config.DeviceOS,
		DeviceOSVer: config.DeviceOSVer,
		DeviceModel: config.DeviceModel,
		UserAgent:   userAgent,
	}

	utils.UpdateHTTPClientConfig(httpConfig)

	details := utils.GetResolvedDeviceDetails()
	render.JSON(w, r, details)
}
