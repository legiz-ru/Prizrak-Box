package internal

import (
	"fmt"
	"github.com/legiz-ru/prizrak-box/pkg/constant"
	sysProxy "github.com/legiz-ru/prizrak-box/pkg/sys/proxy"
	"github.com/metacubex/mihomo/hub/executor"
	RC "github.com/metacubex/mihomo/rules/common"
	"github.com/metacubex/mihomo/tunnel"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/pkg/cache"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/mihomo/config"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
	plog "github.com/sirupsen/logrus"
)

// Init meta 启动前的初始化
func Init() {
	// 设置工作目录
	C.SetHomeDir(utils.GetUserHomeDir())

	// 设置日志输出目录
	logName := "px-server.log"
	logFilePath := utils.GetUserHomeDir("logs", logName)
	f, err := utils.CreateFileForAppend(logFilePath)
	if err != nil {
		return
	}

	// 组合一下即可，os.Stdout代表标准输出流
	if runtime.GOOS != "windows" {
		// 组合一下即可，os.Stdout代表标准输出流
		multiWriter := io.MultiWriter(os.Stdout, f)
		plog.SetOutput(multiWriter)
	} else {
		plog.SetOutput(f)
	}

	// 设置cache db
	db := cache.GetDBInstance()
	if db == nil {
		os.Exit(1)
	}
	cache.GetMetaDB()

	// 输出日志
	log.Infoln("[CacheDB] initialized")
	log.Infoln("[HomePath] is %s", utils.GetUserHomeDir())

	// 修改权限
	pathTemp := utils.GetUserHomeDir("logs", "px-client.log")
	_ = utils.SetPermissions(pathTemp)
	pathTemp = utils.GetUserHomeDir("px-electron.db")
	_ = utils.SetPermissions(pathTemp)
	pathTemp = utils.GetUserHomeDir("px-electron.db/config.json")
	pathTempDst := utils.GetUserHomeDir("px-electron.db/config_temp.json")
	_ = utils.ModifyFilePermissions(pathTemp, pathTempDst)
	log.Infoln("[Permission] is ok")

	// 释放资源文件
	_, _ = utils.SaveFile(utils.GetUserHomeDir("geoip.metadb"), GeoIp)
	_, _ = utils.SaveFile(utils.GetUserHomeDir("GeoSite.dat"), GeoSite)
	_, _ = utils.SaveFile(utils.GetUserHomeDir("ASN.mmdb"), ASN)

	// 释放大模型
	bin := utils.GetUserHomeDir("Model.bin")
	if !utils.FileExists(bin) {
		_, _ = utils.SaveFile(bin, ModelBin)
	}

	GeoIp = nil
	GeoSite = nil
	ASN = nil
	ModelBin = nil

	EnsureBuiltinTemplates()
}

var NowConfig *config.Config
var havaStartCore bool
var StartLock = sync.Mutex{}

func loadProfileRawConfig(profile models.Profile) (*config.RawConfig, error) {
	providerBuf, err := os.ReadFile(utils.GetUserHomeDir(profile.Path))
	if err != nil {
		return nil, err
	}

	rawCfg, err := config.UnmarshalRawConfig(providerBuf)
	if err != nil {
		return nil, err
	}

	return rawCfg, nil
}

func profileTag(profile models.Profile, used map[string]int) string {
	base := strings.TrimSpace(profile.HeaderTitle)
	if base == "" {
		base = strings.TrimSpace(profile.Title)
	}
	if base == "" {
		base = profile.Id
	}

	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, ":", "_")
	base = strings.ReplaceAll(base, "[", "_")
	base = strings.ReplaceAll(base, "]", "_")
	base = strings.TrimSpace(base)
	if base == "" {
		base = profile.Id
	}

	if count, ok := used[base]; ok {
		used[base] = count + 1
		return fmt.Sprintf("%s-%d", base, count+1)
	}

	used[base] = 1
	return base
}

func ensureUniqueKey(base string, used map[string]int) string {
	if base == "" {
		base = "item"
	}

	if count, ok := used[base]; ok {
		count++
		used[base] = count
		return fmt.Sprintf("%s-%d", base, count)
	}

	used[base] = 1
	return base
}

func formatProxyName(name string, tag string, applyTag bool, used map[string]int) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "proxy"
	}
	if !applyTag || tag == "" {
		return ensureUniqueKey(trimmed, used)
	}

	base := fmt.Sprintf("%s [%s]", trimmed, tag)
	if _, ok := used[base]; !ok {
		used[base] = 1
		return base
	}

	count := used[base] + 1
	for {
		candidate := fmt.Sprintf("%s-%d [%s]", trimmed, count, tag)
		if _, ok := used[candidate]; !ok {
			used[candidate] = 1
			used[base] = count
			return candidate
		}
		count++
	}
}

func providerSuffix(tag string, key string, providerCount int) string {
	if tag == "" {
		return ""
	}
	key = strings.ReplaceAll(key, "[", "_")
	key = strings.ReplaceAll(key, "]", "_")
	if providerCount > 1 && key != "" {
		return fmt.Sprintf(" [%s-%s]", tag, key)
	}
	return fmt.Sprintf(" [%s]", tag)
}

func applyProviderSuffix(provider map[string]any, suffix string) {
	if suffix == "" {
		return
	}

	override, ok := provider["override"].(map[string]any)
	if !ok {
		if legacy, ok := provider["override"].(map[string]string); ok {
			override = map[string]any{}
			for key, value := range legacy {
				override[key] = value
			}
		} else {
			override = map[string]any{}
		}
		provider["override"] = override
	}

	if existing, ok := override["additional-suffix"].(string); ok {
		if existing != "" && !strings.Contains(existing, suffix) {
			override["additional-suffix"] = existing + suffix
		}
		return
	}

	override["additional-suffix"] = suffix
}

func applyProxyNameMapping(rawCfg *config.RawConfig, mapping map[string]string) {
	if len(mapping) == 0 {
		return
	}

	for _, proxy := range rawCfg.Proxy {
		if dialer, ok := proxy["dialer-proxy"].(string); ok {
			if newName, ok := mapping[dialer]; ok {
				proxy["dialer-proxy"] = newName
			}
		}
	}

	updateProxyGroupProxies(rawCfg.ProxyGroup, mapping)
	rawCfg.Rule = updateRuleList(rawCfg.Rule, mapping)
	if rawCfg.SubRules != nil {
		for name, rules := range rawCfg.SubRules {
			rawCfg.SubRules[name] = updateRuleList(rules, mapping)
		}
	}

	if updated, changed := updateNameServerList(rawCfg.DNS.NameServer, mapping); changed {
		rawCfg.DNS.NameServer = updated
	}
	if updated, changed := updateNameServerList(rawCfg.DNS.Fallback, mapping); changed {
		rawCfg.DNS.Fallback = updated
	}
	if updated, changed := updateNameServerList(rawCfg.DNS.DefaultNameserver, mapping); changed {
		rawCfg.DNS.DefaultNameserver = updated
	}
	if updated, changed := updateNameServerList(rawCfg.DNS.ProxyServerNameserver, mapping); changed {
		rawCfg.DNS.ProxyServerNameserver = updated
	}
	if updated, changed := updateNameServerList(rawCfg.DNS.DirectNameServer, mapping); changed {
		rawCfg.DNS.DirectNameServer = updated
	}
	if rawCfg.DNS.NameServerPolicy != nil {
		for pair := rawCfg.DNS.NameServerPolicy.Oldest(); pair != nil; pair = pair.Next() {
			updated, changed := updateNameServerPolicyValue(pair.Value, mapping)
			if changed {
				rawCfg.DNS.NameServerPolicy.Set(pair.Key, updated)
			}
		}
	}

	for i := range rawCfg.Tunnels {
		if newName, ok := mapping[rawCfg.Tunnels[i].Proxy]; ok {
			rawCfg.Tunnels[i].Proxy = newName
		}
	}
	if newName, ok := mapping[rawCfg.NTP.DialerProxy]; ok {
		rawCfg.NTP.DialerProxy = newName
	}
}

func updateProxyGroupProxies(groups []map[string]any, mapping map[string]string) {
	for _, group := range groups {
		value, ok := group["proxies"]
		if !ok {
			continue
		}

		switch proxies := value.(type) {
		case []string:
			changed := false
			for i, name := range proxies {
				if newName, ok := mapping[name]; ok {
					proxies[i] = newName
					changed = true
				}
			}
			if changed {
				group["proxies"] = proxies
			}
		case []any:
			changed := false
			for i, item := range proxies {
				name, ok := item.(string)
				if !ok {
					continue
				}
				if newName, ok := mapping[name]; ok {
					proxies[i] = newName
					changed = true
				}
			}
			if changed {
				group["proxies"] = proxies
			}
		}
	}
}

func updateRuleList(rules []string, mapping map[string]string) []string {
	if len(rules) == 0 {
		return rules
	}

	changed := false
	updated := make([]string, len(rules))
	for i, rule := range rules {
		next := rewriteRuleTarget(rule, mapping)
		if next != rule {
			changed = true
		}
		updated[i] = next
	}
	if !changed {
		return rules
	}
	return updated
}

func rewriteRuleTarget(rule string, mapping map[string]string) string {
	tp, payload, target, params := RC.ParseRulePayload(rule, true)
	if target == "" {
		return rule
	}
	newTarget, ok := mapping[target]
	if !ok {
		return rule
	}
	return buildRule(tp, payload, newTarget, params)
}

func buildRule(tp string, payload string, target string, params []string) string {
	parts := []string{tp}
	if payload != "" {
		parts = append(parts, payload)
	}
	if target != "" {
		parts = append(parts, target)
	}
	if len(params) > 0 {
		parts = append(parts, params...)
	}
	return strings.Join(parts, ",")
}

func updateNameServerList(list []string, mapping map[string]string) ([]string, bool) {
	if len(list) == 0 {
		return list, false
	}

	changed := false
	updated := make([]string, len(list))
	for i, server := range list {
		next := updateNameServerEntry(server, mapping)
		if next != server {
			changed = true
		}
		updated[i] = next
	}
	if !changed {
		return list, false
	}
	return updated, true
}

func updateNameServerEntry(server string, mapping map[string]string) string {
	parts := strings.SplitN(server, "#", 2)
	if len(parts) < 2 {
		return server
	}

	fragment := parts[1]
	if fragment == "" {
		return server
	}

	segments := strings.Split(fragment, "&")
	changed := false
	for i, segment := range segments {
		if strings.Contains(segment, "=") {
			continue
		}
		if newName, ok := mapping[segment]; ok {
			segments[i] = newName
			changed = true
		}
	}
	if !changed {
		return server
	}
	return parts[0] + "#" + strings.Join(segments, "&")
}

func updateNameServerPolicyValue(value any, mapping map[string]string) (any, bool) {
	switch typed := value.(type) {
	case string:
		updated := updateNameServerEntry(typed, mapping)
		if updated != typed {
			return updated, true
		}
	case []string:
		updated, changed := updateNameServerList(typed, mapping)
		if changed {
			return updated, true
		}
	case []any:
		changed := false
		updated := make([]any, len(typed))
		for i, item := range typed {
			name, ok := item.(string)
			if !ok {
				updated[i] = item
				continue
			}
			next := updateNameServerEntry(name, mapping)
			updated[i] = next
			if next != name {
				changed = true
			}
		}
		if changed {
			return updated, true
		}
	}
	return value, false
}

func buildMergedRawConfig(primary models.Profile, profiles []models.Profile) (*config.RawConfig, error) {
	primaryRaw, err := loadProfileRawConfig(primary)
	if err != nil {
		return nil, err
	}

	useTemplate, templateId, templateBuf := getTemplate(primary)
	multi := len(profiles) > 1

	extras := make([]models.Profile, 0, len(profiles))
	for _, p := range profiles {
		if p.Id != primary.Id {
			extras = append(extras, p)
		}
	}

	profileKeys := map[string]int{}
	profileTags := map[string]string{}
	primaryTag := profileTag(primary, profileKeys)
	profileTags[primary.Id] = primaryTag
	for _, extra := range extras {
		profileTags[extra.Id] = profileTag(extra, profileKeys)
	}
	proxyNameMap := map[string]string{}

	if useTemplate || len(primaryRaw.Rule) == 0 {
		rawCfg, err := config.UnmarshalRawConfig(templateBuf)
		if err != nil {
			return nil, err
		}

		changeProvidersPath("template", templateId, rawCfg)

		providers := map[string]map[string]any{}
		proxies := make([]map[string]any, 0)
		usedProviderKeys := map[string]int{}
		usedProxyNames := map[string]int{}

		if len(primaryRaw.ProxyProvider) > 0 {
			providerCount := len(primaryRaw.ProxyProvider)
			for key, value := range primaryRaw.ProxyProvider {
				if multi {
					applyProviderSuffix(value, providerSuffix(primaryTag, key, providerCount))
				}
				providers[key] = value
				if multi {
					usedProviderKeys[key] = 1
				}
			}
		}

		if len(primaryRaw.Proxy) > 0 {
			for _, proxy := range primaryRaw.Proxy {
				if name, ok := proxy["name"].(string); ok && name != "" {
					if multi {
						newName := formatProxyName(name, primaryTag, true, usedProxyNames)
						if newName != name {
							proxy["name"] = newName
							if _, exists := proxyNameMap[name]; !exists {
								if _, exists := proxyNameMap[name]; !exists {
									proxyNameMap[name] = newName
								}
							}
						}
					}
				}
				proxies = append(proxies, proxy)
			}
		}

		if multi {
			for _, extra := range extras {
				extraRaw, err := loadProfileRawConfig(extra)
				if err != nil {
					log.Warnln("Read config error for %s: %s", extra.Id, err.Error())
					continue
				}

				tag := profileTags[extra.Id]

				if len(extraRaw.ProxyProvider) > 0 {
					providerCount := len(extraRaw.ProxyProvider)
					for key, value := range extraRaw.ProxyProvider {
						if multi {
							applyProviderSuffix(value, providerSuffix(tag, key, providerCount))
						}
						newKey := fmt.Sprintf("%s-%s", tag, key)
						if providerCount == 1 {
							newKey = tag
						}
						newKey = ensureUniqueKey(newKey, usedProviderKeys)
						providers[newKey] = value
					}
				}

				if len(extraRaw.Proxy) > 0 {
					for _, proxy := range extraRaw.Proxy {
						if name, ok := proxy["name"].(string); ok && name != "" {
							newName := formatProxyName(name, tag, multi, usedProxyNames)
							if newName != name {
								proxy["name"] = newName
								proxyNameMap[name] = newName
							}
						}
						proxies = append(proxies, proxy)
					}
				}
			}
		}

		if len(providers) > 0 {
			if len(rawCfg.ProxyProvider) > 0 {
				for key, value := range providers {
					rawCfg.ProxyProvider[key] = value
				}
			} else {
				rawCfg.ProxyProvider = providers
			}
		}

		if len(proxies) > 0 {
			if len(rawCfg.Proxy) > 0 {
				rawCfg.Proxy = append(rawCfg.Proxy, proxies...)
			} else {
				rawCfg.Proxy = proxies
			}
		}

		if len(proxyNameMap) > 0 {
			applyProxyNameMapping(rawCfg, proxyNameMap)
		}

		return rawCfg, nil
	}

	rawCfg := primaryRaw
	if !multi || len(extras) == 0 {
		return rawCfg, nil
	}

	providers := map[string]map[string]any{}
	proxies := make([]map[string]any, 0)
	usedProviderKeys := map[string]int{}
	usedProxyNames := map[string]int{}

	for _, proxy := range rawCfg.Proxy {
		if name, ok := proxy["name"].(string); ok && name != "" {
			newName := formatProxyName(name, primaryTag, true, usedProxyNames)
			if newName != name {
				proxy["name"] = newName
				if _, exists := proxyNameMap[name]; !exists {
					proxyNameMap[name] = newName
				}
			}
		}
	}

	if len(rawCfg.ProxyProvider) > 0 {
		providerCount := len(rawCfg.ProxyProvider)
		for key, value := range rawCfg.ProxyProvider {
			usedProviderKeys[key] = 1
			applyProviderSuffix(value, providerSuffix(primaryTag, key, providerCount))
		}
	}

	for _, extra := range extras {
		extraRaw, err := loadProfileRawConfig(extra)
		if err != nil {
			log.Warnln("Read config error for %s: %s", extra.Id, err.Error())
			continue
		}

		tag := profileTags[extra.Id]

		if len(extraRaw.ProxyProvider) > 0 {
			providerCount := len(extraRaw.ProxyProvider)
			for key, value := range extraRaw.ProxyProvider {
				if multi {
					applyProviderSuffix(value, providerSuffix(tag, key, providerCount))
				}
				newKey := fmt.Sprintf("%s-%s", tag, key)
				if providerCount == 1 {
					newKey = tag
				}
				newKey = ensureUniqueKey(newKey, usedProviderKeys)
				providers[newKey] = value
			}
		}

		if len(extraRaw.Proxy) > 0 {
			for _, proxy := range extraRaw.Proxy {
				if name, ok := proxy["name"].(string); ok && name != "" {
					newName := formatProxyName(name, tag, multi, usedProxyNames)
					if newName != name {
						proxy["name"] = newName
						if _, exists := proxyNameMap[name]; !exists {
							proxyNameMap[name] = newName
						}
					}
				}
				proxies = append(proxies, proxy)
			}
		}
	}

	if len(providers) > 0 {
		if rawCfg.ProxyProvider != nil {
			for key, value := range providers {
				rawCfg.ProxyProvider[key] = value
			}
		} else {
			rawCfg.ProxyProvider = providers
		}
	}

	if len(proxies) > 0 {
		rawCfg.Proxy = append(rawCfg.Proxy, proxies...)
	}

	if len(proxyNameMap) > 0 {
		applyProxyNameMapping(rawCfg, proxyNameMap)
	}

	return rawCfg, nil
}

// startCore 函数用于启动核心功能
func startCore(profile models.Profile, profiles []models.Profile, reload bool) {
	rawCfg, err := buildMergedRawConfig(profile, profiles)
	if err != nil {
		log.Warnln("Read config error: %s", err.Error())
		return
	}

	if len(rawCfg.ProxyProvider) > 1 {
		for key, value := range rawCfg.ProxyProvider {
			override, ok := value["override"].(map[string]any)
			if !ok {
				if legacy, ok := value["override"].(map[string]string); ok {
					override = map[string]any{}
					for k, v := range legacy {
						override[k] = v
					}
				} else {
					override = map[string]any{}
				}
				value["override"] = override
			}
			if _, exists := override["additional-suffix"]; exists {
				continue
			}
			override["additional-suffix"] = "-" + key
		}
	}

	// Prizrak-Box 默认配置
	rawCfg.Port = 0
	rawCfg.SocksPort = 0
	rawCfg.TProxyPort = 0
	rawCfg.RedirPort = 0
	rawCfg.ExternalController = ""
	rawCfg.ExternalUI = ""
	rawCfg.ExternalUIURL = ""
	rawCfg.Tun.DNSHijack = []string{"any:53"}
	rawCfg.Tun.AutoRoute = true
	rawCfg.Tun.AutoDetectInterface = true
	rawCfg.Tun.Device = "Prizrak"
	rawCfg.UnifiedDelay = true

	// 从数据库中获取 mihomo 配置,进行 rawCfg 赋值
	var mi models.Mihomo
	_ = cache.Get(constant.Mihomo, &mi)
	if mi.BindAddress == "" {
		mi = models.Mihomo{
			Mode:        "rule",
			Proxy:       false,
			Tun:         false,
			Port:        9697,
			BindAddress: "127.0.0.1",
			Stack:       "Mixed",
			Dns:         false,
			Ipv6:        false,
		}
	}
	rawCfg.Mode = tunnel.ModeMapping[mi.Mode]
	rawCfg.AllowLan = true
	rawCfg.MixedPort = mi.Port
	rawCfg.BindAddress = mi.BindAddress
	rawCfg.Tun.Stack = C.StackTypeMapping[strings.ToLower(mi.Stack)]
	rawCfg.IPv6 = mi.Ipv6

	// 保存规则数
	_ = cache.Put("Rule_No", len(rawCfg.Rule))

	// 解析配置文件2
	NowConfig, err = config.ParseRawConfig(rawCfg)
	if err != nil {
		log.Errorln("ParseRawConfig error: %v", err)
		return
	}

	// 覆盖dns
	if mi.Dns {
		var dns models.Dns
		_ = cache.Get(constant.Dns, &dns)

		if dns.Content == "" {
			dns.Content = DefaultDNS
		}

		cfg, _ := executor.ParseWithBytes([]byte(dns.Content))
		NowConfig.DNS = cfg.DNS
	}

	// 应用配置
	if reload {
		NowConfig.General.Tun.Enable = mi.Tun
	} else {
		// 检测端口占用
		err = utils.IsPortAvailable(mi.BindAddress, mi.Port)
		if err != nil {
			log.Errorln("IsPortAvailable error: %v", err)
			mi.Port, _ = utils.GetRandomPort(mi.BindAddress)
			NowConfig.General.MixedPort = mi.Port
		}

		// 初次加载不能开启tun,不然在windows上会崩
		NowConfig.General.Tun.Enable = false
	}

	// 激活配置
	go executor.ApplyConfig(NowConfig, !reload)

	// 代理开启
	if mi.Proxy {
		_ = sysProxy.EnableProxy(mi.BindAddress, mi.Port)
	}
	// 存储配置
	_ = cache.Put(constant.Mihomo, mi)
	// 更新启动标志
	havaStartCore = true
}

// 获取统一规则分组模板
func getTemplate(profile models.Profile) (bool, string, []byte) {
	// 默认模版ID
	defaultId := fmt.Sprintf("%s%d", constant.PrefixTemplate, 0)

	// 优先启用个性模板
	var template models.Template
	if profile.Template != "" {
		if profile.Template == "m0" {
			return false, defaultId, Template_0
		}
		_ = cache.Get(profile.Template, &template)
	}
	if template.Path != "" {
		body, err := utils.ReadFile(utils.GetUserHomeDir(template.Path))
		if err == nil {
			return true, template.Id, []byte(body)
		}
	}

	// 其次启用通用模板
	var list []models.Template
	_ = cache.GetList(constant.PrefixTemplate, &list)
	for _, m := range list {
		if m.Selected {
			template = m
			break
		}
	}
	if template.Path != "" {
		body, err := utils.ReadFile(utils.GetUserHomeDir(template.Path))
		if err == nil {
			return true, template.Id, []byte(body)
		}
	}

	// 最后返回默认模板
	return false, defaultId, Template_0
}

// SwitchProfile 切换配置
func SwitchProfile(reload bool) {
	StartLock.Lock()
	defer StartLock.Unlock()

	var profiles []models.Profile
	_ = cache.GetList(constant.PrefixProfile, &profiles)

	if len(profiles) == 0 {
		return
	}

	selected := make([]models.Profile, 0, len(profiles))
	for _, p := range profiles {
		if p.Selected {
			selected = append(selected, p)
		}
	}

	if len(selected) == 0 {
		profile := profiles[0]
		profile.Selected = true
		_ = cache.Put(profile.Id, profile)
		selected = append(selected, profile)
	}

	var primaryId string
	_ = cache.Get(constant.ProfilePrimary, &primaryId)

	primary := selected[0]
	foundPrimary := false
	if primaryId != "" {
		for _, p := range selected {
			if p.Id == primaryId {
				primary = p
				foundPrimary = true
				break
			}
		}
	}
	if !foundPrimary {
		primary = selected[0]
		_ = cache.Put(constant.ProfilePrimary, primary.Id)
	}

	if !havaStartCore {
		reload = false
	}

	startCore(primary, selected, reload)
}
