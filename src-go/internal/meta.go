package internal

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/legiz-ru/prizrak-box/pkg/constant"
	sysProxy "github.com/legiz-ru/prizrak-box/pkg/sys/proxy"
	"github.com/metacubex/mihomo/hub/executor"
	RC "github.com/metacubex/mihomo/rules/common"
	"github.com/metacubex/mihomo/tunnel"

	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/pkg/cache"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/mihomo/config"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
	plog "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
	releaseGeoData()

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

func renameProfileProxies(proxies []map[string]any, tag string, applyTag bool, used map[string]int) map[string]string {
	mapping := map[string]string{}
	if len(proxies) == 0 {
		return mapping
	}

	for _, proxy := range proxies {
		name, ok := proxy["name"].(string)
		if !ok || name == "" {
			continue
		}
		newName := formatProxyName(name, tag, applyTag, used)
		if newName != name {
			proxy["name"] = newName
			if _, exists := mapping[name]; !exists {
				mapping[name] = newName
			}
		}
	}

	if len(mapping) == 0 {
		return mapping
	}

	for _, proxy := range proxies {
		dialer, ok := proxy["dialer-proxy"].(string)
		if !ok || dialer == "" {
			continue
		}
		if newName, ok := mapping[dialer]; ok {
			proxy["dialer-proxy"] = newName
		}
	}

	return mapping
}

func addProxyOrigins(proxies []map[string]any, tag string, origins map[string]string) {
	if len(proxies) == 0 || tag == "" || origins == nil {
		return
	}

	for _, proxy := range proxies {
		name, ok := proxy["name"].(string)
		if !ok || name == "" {
			continue
		}
		origins[name] = tag
	}
}

func readProviderProxies(path string) []map[string]any {
	if path == "" {
		return nil
	}

	body, err := os.ReadFile(utils.GetUserHomeDir(path))
	if err != nil {
		return nil
	}

	var yml models.Yml
	if err := yaml.Unmarshal(body, &yml); err == nil && len(yml.Proxies) > 0 {
		return yml.Proxies
	}

	var payload struct {
		Payload []map[string]any `yaml:"payload"`
	}
	if err := yaml.Unmarshal(body, &payload); err == nil && len(payload.Payload) > 0 {
		return payload.Payload
	}

	var proxies []map[string]any
	if err := yaml.Unmarshal(body, &proxies); err == nil && len(proxies) > 0 {
		return proxies
	}

	return nil
}

func providerAdditionalSuffix(provider map[string]any) string {
	if len(provider) == 0 {
		return ""
	}

	if override, ok := provider["override"].(map[string]any); ok {
		if suffix, ok := override["additional-suffix"].(string); ok {
			return suffix
		}
	} else if legacy, ok := provider["override"].(map[string]string); ok {
		if suffix, ok := legacy["additional-suffix"]; ok {
			return suffix
		}
	}

	return ""
}

func parseOriginSuffix(suffix string) string {
	suffix = strings.TrimSpace(suffix)
	if !strings.HasSuffix(suffix, "]") {
		return ""
	}
	start := strings.LastIndex(suffix, "[")
	if start == -1 || start+1 >= len(suffix)-1 {
		return ""
	}
	origin := strings.TrimSpace(suffix[start+1 : len(suffix)-1])
	return origin
}

func addProviderProxyOrigins(provider map[string]any, tag string, origins map[string]string) {
	if len(provider) == 0 || tag == "" || origins == nil {
		return
	}

	path, ok := provider["path"].(string)
	if !ok || path == "" {
		return
	}

	proxies := readProviderProxies(path)
	if len(proxies) == 0 {
		return
	}

	suffix := providerAdditionalSuffix(provider)
	origin := parseOriginSuffix(suffix)
	if origin == "" {
		origin = tag
	}

	for _, proxy := range proxies {
		name, ok := proxy["name"].(string)
		if !ok || name == "" {
			continue
		}
		mappedName := name
		if suffix != "" {
			mappedName = name + suffix
		}
		origins[mappedName] = origin
	}
}

func applyProviderProxyMapping(provider map[string]any, mapping map[string]string) {
	if len(mapping) == 0 || len(provider) == 0 {
		return
	}

	if dialer, ok := provider["dialer-proxy"].(string); ok {
		if newName, ok := mapping[dialer]; ok {
			provider["dialer-proxy"] = newName
		}
	}

	if name, ok := provider["proxy"].(string); ok {
		if newName, ok := mapping[name]; ok {
			provider["proxy"] = newName
		}
	}
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

func isTruthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func isGroupIncludeAll(group map[string]any) bool {
	if group == nil {
		return false
	}
	if isTruthy(group["include-all"]) {
		return true
	}
	if isTruthy(group["include-all-proxies"]) {
		return true
	}
	return false
}

func extractProxyNames(proxies []map[string]any) []string {
	names := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		name, ok := proxy["name"].(string)
		if !ok || name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func mergeGroupProxies(groups []map[string]any, proxyNames []string) {
	if len(groups) == 0 || len(proxyNames) == 0 {
		return
	}

	for _, group := range groups {
		if isGroupIncludeAll(group) {
			continue
		}
		if _, ok := group["use"]; ok {
			continue
		}

		value, ok := group["proxies"]
		if !ok {
			continue
		}

		switch proxies := value.(type) {
		case []string:
			seen := make(map[string]bool, len(proxies))
			merged := make([]string, 0, len(proxies)+len(proxyNames))
			for _, name := range proxies {
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				merged = append(merged, name)
			}
			for _, name := range proxyNames {
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				merged = append(merged, name)
			}
			group["proxies"] = merged
		case []any:
			seen := make(map[string]bool, len(proxies))
			merged := make([]any, 0, len(proxies)+len(proxyNames))
			for _, item := range proxies {
				name, ok := item.(string)
				if !ok {
					merged = append(merged, item)
					continue
				}
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				merged = append(merged, name)
			}
			for _, name := range proxyNames {
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				merged = append(merged, name)
			}
			group["proxies"] = merged
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
	proxyOrigins := map[string]string{}

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

		if multi && len(primaryRaw.Proxy) > 0 {
			proxyNameMap = renameProfileProxies(primaryRaw.Proxy, primaryTag, true, usedProxyNames)
		}

		if len(proxyNameMap) > 0 {
			applyProxyNameMapping(rawCfg, proxyNameMap)
		}

		if len(primaryRaw.ProxyProvider) > 0 {
			providerCount := len(primaryRaw.ProxyProvider)
			for key, value := range primaryRaw.ProxyProvider {
				if multi {
					applyProviderSuffix(value, providerSuffix(primaryTag, key, providerCount))
				}
				applyProviderProxyMapping(value, proxyNameMap)
				if multi {
					addProviderProxyOrigins(value, primaryTag, proxyOrigins)
				}
				providers[key] = value
				if multi {
					usedProviderKeys[key] = 1
				}
			}
		}

		if len(primaryRaw.Proxy) > 0 {
			if multi {
				addProxyOrigins(primaryRaw.Proxy, primaryTag, proxyOrigins)
			}
			proxies = append(proxies, primaryRaw.Proxy...)
		}

		if multi {
			for _, extra := range extras {
				extraRaw, err := loadProfileRawConfig(extra)
				if err != nil {
					log.Warnln("Read config error for %s: %s", extra.Id, err.Error())
					continue
				}

				tag := profileTags[extra.Id]

				extraProxyMap := map[string]string{}
				if len(extraRaw.Proxy) > 0 {
					extraProxyMap = renameProfileProxies(extraRaw.Proxy, tag, multi, usedProxyNames)
					if multi {
						addProxyOrigins(extraRaw.Proxy, tag, proxyOrigins)
					}
					proxies = append(proxies, extraRaw.Proxy...)
				}

				if len(extraRaw.ProxyProvider) > 0 {
					providerCount := len(extraRaw.ProxyProvider)
					for key, value := range extraRaw.ProxyProvider {
						if multi {
							applyProviderSuffix(value, providerSuffix(tag, key, providerCount))
						}
						applyProviderProxyMapping(value, extraProxyMap)
						if multi {
							addProviderProxyOrigins(value, tag, proxyOrigins)
						}
						newKey := fmt.Sprintf("%s-%s", tag, key)
						if providerCount == 1 {
							newKey = tag
						}
						newKey = ensureUniqueKey(newKey, usedProviderKeys)
						providers[newKey] = value
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

		if multi {
			mergeGroupProxies(rawCfg.ProxyGroup, extractProxyNames(rawCfg.Proxy))
		}

		_ = cache.Put(constant.ProfileProxyOrigin, proxyOrigins)
		return rawCfg, nil
	}

	rawCfg := primaryRaw
	if !multi || len(extras) == 0 {
		_ = cache.Put(constant.ProfileProxyOrigin, proxyOrigins)
		return rawCfg, nil
	}

	providers := map[string]map[string]any{}
	proxies := make([]map[string]any, 0)
	usedProviderKeys := map[string]int{}
	usedProxyNames := map[string]int{}

	proxyNameMap = renameProfileProxies(rawCfg.Proxy, primaryTag, true, usedProxyNames)
	if multi {
		addProxyOrigins(rawCfg.Proxy, primaryTag, proxyOrigins)
	}

	if len(rawCfg.ProxyProvider) > 0 {
		providerCount := len(rawCfg.ProxyProvider)
		for key, value := range rawCfg.ProxyProvider {
			usedProviderKeys[key] = 1
			applyProviderSuffix(value, providerSuffix(primaryTag, key, providerCount))
			applyProviderProxyMapping(value, proxyNameMap)
			addProviderProxyOrigins(value, primaryTag, proxyOrigins)
		}
	}

	if len(proxyNameMap) > 0 {
		applyProxyNameMapping(rawCfg, proxyNameMap)
	}

	for _, extra := range extras {
		extraRaw, err := loadProfileRawConfig(extra)
		if err != nil {
			log.Warnln("Read config error for %s: %s", extra.Id, err.Error())
			continue
		}

		tag := profileTags[extra.Id]

		extraProxyMap := map[string]string{}
		if len(extraRaw.Proxy) > 0 {
			extraProxyMap = renameProfileProxies(extraRaw.Proxy, tag, multi, usedProxyNames)
			if multi {
				addProxyOrigins(extraRaw.Proxy, tag, proxyOrigins)
			}
		}

		if len(extraRaw.ProxyProvider) > 0 {
			providerCount := len(extraRaw.ProxyProvider)
			for key, value := range extraRaw.ProxyProvider {
				if multi {
					applyProviderSuffix(value, providerSuffix(tag, key, providerCount))
				}
				applyProviderProxyMapping(value, extraProxyMap)
				if multi {
					addProviderProxyOrigins(value, tag, proxyOrigins)
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
			proxies = append(proxies, extraRaw.Proxy...)
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

	if multi {
		mergeGroupProxies(rawCfg.ProxyGroup, extractProxyNames(rawCfg.Proxy))
	}

	_ = cache.Put(constant.ProfileProxyOrigin, proxyOrigins)
	return rawCfg, nil
}

func sortProfilesByOrder(profiles []models.Profile) []models.Profile {
	ordered := make([]models.Profile, len(profiles))
	copy(ordered, profiles)

	var order []models.Profile
	_ = cache.Get(constant.ProfileOrder, &order)

	orderMap := make(map[string]int)
	for index, item := range order {
		orderMap[item.Id] = index
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		indexI, existsI := orderMap[ordered[i].Id]
		indexJ, existsJ := orderMap[ordered[j].Id]
		if existsI && existsJ {
			return indexI < indexJ
		}
		if existsI {
			return true
		}
		if existsJ {
			return false
		}
		return ordered[i].Order < ordered[j].Order
	})

	return ordered
}

func orderSelectedProfiles(profiles []models.Profile) []models.Profile {
	selected := make([]models.Profile, 0, len(profiles))
	selectedMap := make(map[string]models.Profile, len(profiles))
	for _, p := range profiles {
		if p.Selected {
			selected = append(selected, p)
			selectedMap[p.Id] = p
		}
	}
	if len(selected) == 0 {
		return selected
	}

	var selectionOrder []string
	_ = cache.Get(constant.ProfileSelectionOrder, &selectionOrder)

	if len(selectionOrder) == 0 {
		ordered := sortProfilesByOrder(profiles)
		result := make([]models.Profile, 0, len(selected))
		for _, p := range ordered {
			if p.Selected {
				result = append(result, p)
			}
		}
		return result
	}

	result := make([]models.Profile, 0, len(selected))
	seen := make(map[string]bool, len(selected))
	for _, id := range selectionOrder {
		if p, ok := selectedMap[id]; ok && !seen[id] {
			result = append(result, p)
			seen[id] = true
		}
	}

	if len(result) < len(selected) {
		ordered := sortProfilesByOrder(profiles)
		for _, p := range ordered {
			if p.Selected && !seen[p.Id] {
				result = append(result, p)
				seen[p.Id] = true
			}
		}
	}

	return result
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

	// 代理开启 - включаем системный прокси только если включен режим системного прокси
	if mi.Proxy && mi.SystemProxyMode {
		if mi.Username != "" {
			_ = sysProxy.EnableProxyForUser(mi.BindAddress, mi.Port, mi.Username)
		} else {
			_ = sysProxy.EnableProxy(mi.BindAddress, mi.Port)
		}
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

	selected := orderSelectedProfiles(profiles)

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

// 释放GEO数据
func releaseGeoData() {
	GeoIpPath := utils.GetUserHomeDir("geoip.metadb")
	if !utils.FileExists(GeoIpPath) {
		_, _ = utils.SaveFile(GeoIpPath, GeoIp)
	}

	GeoSitePath := utils.GetUserHomeDir("GeoSite.dat")
	if !utils.FileExists(GeoSitePath) {
		_, _ = utils.SaveFile(GeoSitePath, GeoSite)
	}

	ASNPath := utils.GetUserHomeDir("ASN.mmdb")
	if !utils.FileExists(ASNPath) {
		_, _ = utils.SaveFile(ASNPath, ASN)
	}
}
