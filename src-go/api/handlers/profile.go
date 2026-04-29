package handlers

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/legiz-ru/prizrak-box/api/job"
	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/internal"
	"github.com/legiz-ru/prizrak-box/pkg/cache"
	"github.com/legiz-ru/prizrak-box/pkg/constant"
	"github.com/legiz-ru/prizrak-box/pkg/proxy"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/chi"
	"github.com/metacubex/chi/render"
	"github.com/metacubex/http"
	"github.com/metacubex/mihomo/hub/route"
	"github.com/metacubex/mihomo/log"
	"gopkg.in/yaml.v3"
)

func Profile(r chi.Router) {
	r.Mount("/profile", profileRouter())
}

func profileRouter() http.Handler {
	r := chi.NewRouter()
	// 增加
	r.Post("/", addFromWeb)
	r.Post("/file", addFromFile)
	// 删除
	r.Post("/delete", deleteProfile)
	// 修改
	r.Put("/", putProfile)
	// 查找
	r.Get("/", getProfile)

	r.Get("/proxy-origins", getProxyOrigins)

	r.Get("/serverDescriptions", getProxyServerDescriptions)

	// 更新订阅
	r.Put("/refresh", refreshProfile)
	// 切换订阅
	r.Patch("/", switchProfile)
	// 存储排序
	r.Get("/order", saveProfileOrder)

	return r
}

// ErrorResponse 是一个共通的方法，用于返回错误信息到客户端
func ErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	render.Status(r, http.StatusBadRequest)
	render.JSON(w, r, route.HTTPError{Message: err.Error()})
}

type hwidErrorResponse struct {
	Message              string `json:"message"`
	HwidNotSupported     bool   `json:"hwidNotSupported,omitempty"`
	HwidMaxDevicesReached bool   `json:"hwidMaxDevicesReached,omitempty"`
	SupportUrl           string `json:"supportUrl,omitempty"`
}

// checkHwidHeaders читает HWID-заголовки из ответа сервера подписки.
// Возвращает (notSupported, maxDevicesReached).
func checkHwidHeaders(headers http.Header) (notSupported, maxDevicesReached bool) {
	notSupported = strings.EqualFold(strings.TrimSpace(headers.Get("X-Hwid-Not-Supported")), "true")
	maxDevicesReached = strings.EqualFold(strings.TrimSpace(headers.Get("X-Hwid-Max-Devices-Reached")), "true")
	return
}

// profileRefreshResponse расширяет Profile транзитными HWID-полями для ответа /profile/refresh.
// Поля HwidNotSupported и HwidMaxDevicesReached не кешируются — они задаются после UpdateDb.
type profileRefreshResponse struct {
	*models.Profile
	HwidNotSupported     bool `json:"hwidNotSupported,omitempty"`
	HwidMaxDevicesReached bool `json:"hwidMaxDevicesReached,omitempty"`
}

func getProxyOrigins(w http.ResponseWriter, r *http.Request) {
	var origins map[string]string
	if err := cache.Get(constant.ProfileProxyOrigin, &origins); err != nil || origins == nil {
		origins = map[string]string{}
	}

	render.JSON(w, r, origins)
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

func updateSelectionOrder(order []string, profiles []models.Profile, targetId string, desired bool, primaryId string) []string {
	selected := make(map[string]bool, len(profiles))
	for _, p := range profiles {
		if p.Selected {
			selected[p.Id] = true
		}
	}
	if len(selected) == 0 {
		return nil
	}

	next := make([]string, 0, len(selected))
	seen := make(map[string]bool, len(selected))
	for _, id := range order {
		if id == targetId && !desired {
			continue
		}
		if selected[id] && !seen[id] {
			next = append(next, id)
			seen[id] = true
		}
	}

	if desired && selected[targetId] && !seen[targetId] {
		next = append(next, targetId)
		seen[targetId] = true
	}

	if len(next) < len(selected) {
		orderedProfiles := sortProfilesByOrder(profiles)
		for _, p := range orderedProfiles {
			if selected[p.Id] && !seen[p.Id] {
				next = append(next, p.Id)
				seen[p.Id] = true
			}
		}
	}

	if primaryId != "" && selected[primaryId] {
		for i, id := range next {
			if id == primaryId {
				if i > 0 {
					next = append([]string{primaryId}, append(next[:i], next[i+1:]...)...)
				}
				break
			}
		}
	}

	return next
}

func getProfile(w http.ResponseWriter, r *http.Request) {
	var res []models.Profile
	_ = cache.GetList(constant.PrefixProfile, &res)

	res = sortProfilesByOrder(res)

	var primaryId string
	_ = cache.Get(constant.ProfilePrimary, &primaryId)
	var selectionOrder []string
	_ = cache.Get(constant.ProfileSelectionOrder, &selectionOrder)

	selectionOrder = updateSelectionOrder(selectionOrder, res, "", true, primaryId)
	if len(selectionOrder) > 0 {
		_ = cache.Put(constant.ProfileSelectionOrder, selectionOrder)
	}

	primarySet := false
	if primaryId != "" {
		for i := range res {
			if res[i].Id == primaryId && res[i].Selected {
				primarySet = true
				break
			}
		}
	}

	if !primarySet && len(selectionOrder) > 0 {
		primaryId = selectionOrder[0]
		primarySet = true
		_ = cache.Put(constant.ProfilePrimary, primaryId)
	}

	if !primarySet {
		for i := range res {
			if res[i].Selected {
				primaryId = res[i].Id
				primarySet = true
				_ = cache.Put(constant.ProfilePrimary, primaryId)
				break
			}
		}
	}

	if !primarySet && len(res) > 0 {
		primaryId = res[0].Id
		_ = cache.Put(constant.ProfilePrimary, primaryId)
	}

	selectionOrder = updateSelectionOrder(selectionOrder, res, "", true, primaryId)
	if len(selectionOrder) > 0 {
		_ = cache.Put(constant.ProfileSelectionOrder, selectionOrder)
	}
	selectionIndex := make(map[string]int, len(selectionOrder))
	for index, id := range selectionOrder {
		selectionIndex[id] = index + 1
	}

	for i := range res {
		res[i].Primary = res[i].Id == primaryId
		if order, ok := selectionIndex[res[i].Id]; ok && res[i].Selected {
			res[i].SelectionOrder = order
		} else {
			res[i].SelectionOrder = 0
		}
	}

	render.JSON(w, r, res)
}

func getProxyServerDescriptions(w http.ResponseWriter, r *http.Request) {
	var profiles []models.Profile
	_ = cache.GetList(constant.PrefixProfile, &profiles)

	var selectedProfile *models.Profile
	for _, profile := range profiles {
		if profile.Selected {
			selectedProfile = &profile
			break
		}
	}
	if selectedProfile == nil && len(profiles) > 0 {
		selectedProfile = &profiles[0]
	}

	if selectedProfile == nil || selectedProfile.Path == "" {
		render.JSON(w, r, map[string]string{})
		return
	}

	content, err := utils.ReadFile(utils.GetUserHomeDir(selectedProfile.Path))
	if err != nil {
		render.JSON(w, r, map[string]string{})
		return
	}

	rawConfig := map[string]any{}
	if err := yaml.Unmarshal([]byte(content), &rawConfig); err != nil {
		render.JSON(w, r, map[string]string{})
		return
	}

	proxiesRaw, ok := rawConfig["proxies"]
	if !ok {
		render.JSON(w, r, map[string]string{})
		return
	}

	proxiesSlice, ok := proxiesRaw.([]any)
	if !ok {
		render.JSON(w, r, map[string]string{})
		return
	}

	descriptions := map[string]string{}
	for _, proxy := range proxiesSlice {
		proxyMap, ok := proxy.(map[string]any)
		if !ok {
			continue
		}
		name, ok := proxyMap["name"].(string)
		if !ok || name == "" {
			continue
		}
		desc := ""
		if value, ok := proxyMap["serverDescription"].(string); ok {
			desc = value
		} else if value, ok := proxyMap["server_description"].(string); ok {
			desc = value
		} else if value, ok := proxyMap["server-description"].(string); ok {
			desc = value
		}
		desc = strings.TrimSpace(desc)
		if desc == "" {
			continue
		}
		descriptions[name] = desc
	}

	render.JSON(w, r, descriptions)
}

func addFromFile(w http.ResponseWriter, r *http.Request) {
	// 获取数据
	profile := &models.Profile{}
	if err := render.DecodeJSON(r.Body, profile); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	// 解析存盘
	err := internal.Resolve(profile.Content, profile, false)
	if err != nil {
		log.Errorln("[addFromFile] Resolve Error:%v", err)
		ErrorResponse(w, r, err)
		return
	}

	// 更新数据库
	job.UpdateDb(profile, 2)

	render.NoContent(w, r)
}

func addFromWeb(w http.ResponseWriter, r *http.Request) {
	// 获取数据
	profile := &models.Profile{}
	if err := render.DecodeJSON(r.Body, profile); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	// 返回页面list
	ps := make([]*models.Profile, 0)

	// 返回页面错误
	var tempErr error

	// 解析存盘
	err := internal.Resolve(profile.Content, profile, false)
	if err == nil {
		inlineHeaders := internal.ParseInlineHeaders(profile.Content)
		if len(inlineHeaders) > 0 {
			notSupported, maxDevicesReached := checkHwidHeaders(http.Header(map[string][]string(inlineHeaders)))
			if notSupported || maxDevicesReached {
				supportUrl := strings.TrimSpace(inlineHeaders.Get("Support-Url"))
				resp := hwidErrorResponse{
					HwidNotSupported:      notSupported,
					HwidMaxDevicesReached: maxDevicesReached,
					SupportUrl:            supportUrl,
				}
				if notSupported {
					resp.Message = "HWID not supported by client"
				} else {
					resp.Message = "HWID max devices reached"
				}
				render.Status(r, http.StatusUnprocessableEntity)
				render.JSON(w, r, resp)
				return
			}
			internal.ParseHeaders(inlineHeaders, "", profile)
		}
		job.UpdateDb(profile, 2)
		ps = append(ps, profile)
		render.JSON(w, r, ps)
		return
	} else {
		tempErr = err
		log.Errorln("[addFromWeb] Resolve Error:%v", err)
	}

	// 扫描订阅
	subs := internal.ScanSubs(profile.Content)
	ok := false
	for _, sub := range subs {
		headers := map[string]string{}
		res, err := utils.FastGet(sub, headers, proxy.GetProxyUrl())
		if err != nil {
			tempErr = err
			log.Errorln("[addFromWeb] URL = %s, Request Error:%v", sub, err)
			continue
		}

		// Проверяем HWID-заголовки до сохранения профиля (HTTP + инлайн)
		mergedSubHeaders := internal.MergeHeaders(res.Headers, internal.ParseInlineHeaders(res.Body))
		notSupported, maxDevicesReached := checkHwidHeaders(http.Header(map[string][]string(mergedSubHeaders)))
		if notSupported || maxDevicesReached {
			supportUrl := strings.TrimSpace(mergedSubHeaders.Get("Support-Url"))
			resp := hwidErrorResponse{
				HwidNotSupported:     notSupported,
				HwidMaxDevicesReached: maxDevicesReached,
				SupportUrl:           supportUrl,
			}
			if notSupported {
				resp.Message = "HWID not supported by client"
			} else {
				resp.Message = "HWID max devices reached"
			}
			log.Warnln("[addFromWeb] URL = %s, HWID error: notSupported=%v maxDevicesReached=%v", sub, notSupported, maxDevicesReached)
			render.Status(r, http.StatusUnprocessableEntity)
			render.JSON(w, r, resp)
			return
		}

		// 解析存盘
		subProfile := &models.Profile{
			Content: sub,
		}
		err = internal.Resolve(res.Body, subProfile, false)
		if err == nil {
			// 进行请求头解析
			internal.ParseHeaders(mergedSubHeaders, sub, subProfile)
			job.UpdateDb(subProfile, 1)
			ps = append(ps, subProfile)
			ok = true
		} else {
			tempErr = err
			log.Errorln("[addFromWeb] URL = %s, Resolve Error:%v", sub, err)
		}
	}
	if !ok {
		ErrorResponse(w, r, tempErr)
		return
	}

	render.JSON(w, r, ps)
}

func refreshProfile(w http.ResponseWriter, r *http.Request) {
	// 获取数据
	profile := &models.Profile{}
	if err := render.DecodeJSON(r.Body, profile); err != nil {
		ErrorResponse(w, r, err)
		return
	}
	title := profile.Title

	// 发送请求
	sub := profile.Content
	headers := map[string]string{}
	res, err := utils.FastGet(sub, headers, proxy.GetProxyUrl())
	if err != nil {
		ErrorResponse(w, r, err)
		log.Errorln("[refreshProfile] URL = %s, Request Error:%v", sub, err)
		return
	}

	// 解析存盘
	mergedHeaders := internal.MergeHeaders(res.Headers, internal.ParseInlineHeaders(res.Body))
	err = internal.Resolve(res.Body, profile, true)
	if err == nil {
		// 进行请求头解析
		internal.ParseHeaders(mergedHeaders, sub, profile)
		if title != "" {
			profile.Title = title
		}
		job.UpdateDb(profile, 1)

		// 如果配置正在使用中  进行配置更新
		if profile.Selected {
			internal.SwitchProfile(true)
		}
	} else {
		ErrorResponse(w, r, err)
		log.Errorln("[refreshProfile] URL = %s, Resolve Error:%v", sub, err)
		return
	}

	// Читаем HWID-статус ПОСЛЕ UpdateDb — эти поля транзитные, в кеш не попадают
	// Проверяем как HTTP-заголовки, так и инлайн-заголовки из тела ответа
	notSupported, maxDevicesReached := checkHwidHeaders(http.Header(map[string][]string(mergedHeaders)))
	resp := profileRefreshResponse{
		Profile:              profile,
		HwidNotSupported:     notSupported,
		HwidMaxDevicesReached: maxDevicesReached,
	}
	render.JSON(w, r, resp)
}

func putProfile(w http.ResponseWriter, r *http.Request) {
	var profile models.Profile
	if err := render.DecodeJSON(r.Body, &profile); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	// 从数据库获取原始配置
	var dbProfile models.Profile
	_ = cache.Get(profile.Id, &dbProfile)

	if profile.Logo == "" {
		profile.Logo = dbProfile.Logo
	}

	// 存储更新后的数据
	_ = cache.Put(profile.Id, profile)

	// 如果配置正在使用中  进行配置更新
	if profile.Selected && dbProfile.Template != profile.Template {
		internal.SwitchProfile(true)
	}

	render.NoContent(w, r)
}

// 删除配置
func deleteProfile(w http.ResponseWriter, r *http.Request) {
	profile := &models.Profile{}
	if err := render.DecodeJSON(r.Body, profile); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	path := utils.GetUserHomeDir(profile.Path)
	dir := filepath.Dir(path)
	if strings.HasSuffix(dir, "profiles") {
		_ = utils.DeletePath(path)
	} else {
		_ = utils.DeletePath(dir)
	}
	_ = cache.Delete(profile.Id)

	render.NoContent(w, r)
}

type switchProfileRequest struct {
	Id        string `json:"id"`
	Selected  *bool  `json:"selected,omitempty"`
	Exclusive *bool  `json:"exclusive,omitempty"`
}

// 切换配置
func switchProfile(w http.ResponseWriter, r *http.Request) {
	var req switchProfileRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		ErrorResponse(w, r, err)
		return
	}

	if req.Id == "" {
		ErrorResponse(w, r, errors.New("profile id is required"))
		return
	}

	var profiles []models.Profile
	_ = cache.GetList(constant.PrefixProfile, &profiles)
	if len(profiles) == 0 {
		render.NoContent(w, r)
		return
	}

	exclusive := true
	if req.Exclusive != nil {
		exclusive = *req.Exclusive
	}

	if exclusive {
		found := false
		for _, p := range profiles {
			if p.Id == req.Id {
				found = true
				break
			}
		}
		if !found {
			ErrorResponse(w, r, errors.New("profile not found"))
			return
		}

		for _, p := range profiles {
			updated := p
			if p.Id == req.Id {
				updated.Selected = true
			} else if p.Selected {
				updated.Selected = false
			}
			if updated.Selected != p.Selected {
				_ = cache.Put(updated.Id, updated)
			}
		}
		_ = cache.Put(constant.ProfilePrimary, req.Id)
		_ = cache.Put(constant.ProfileSelectionOrder, []string{req.Id})
	} else {
		var target *models.Profile
		for i := range profiles {
			if profiles[i].Id == req.Id {
				target = &profiles[i]
				break
			}
		}
		if target == nil {
			ErrorResponse(w, r, errors.New("profile not found"))
			return
		}

		desired := target.Selected
		if req.Selected != nil {
			desired = *req.Selected
		} else {
			desired = !target.Selected
		}

		if target.Selected != desired {
			target.Selected = desired
			_ = cache.Put(target.Id, *target)
		}

		selectedCount := 0
		for _, p := range profiles {
			if p.Id == target.Id {
				if desired {
					selectedCount++
				}
				continue
			}
			if p.Selected {
				selectedCount++
			}
		}
		if selectedCount == 0 {
			target.Selected = true
			desired = true
			_ = cache.Put(target.Id, *target)
		}

		var primaryId string
		_ = cache.Get(constant.ProfilePrimary, &primaryId)

		var selectionOrder []string
		_ = cache.Get(constant.ProfileSelectionOrder, &selectionOrder)
		selectionOrder = updateSelectionOrder(selectionOrder, profiles, target.Id, desired, primaryId)
		if len(selectionOrder) > 0 {
			_ = cache.Put(constant.ProfileSelectionOrder, selectionOrder)
		}

		primarySelected := false
		if primaryId != "" {
			for _, p := range profiles {
				if p.Id == primaryId && p.Selected {
					primarySelected = true
					break
				}
			}
		}

		if desired {
			if !primarySelected {
				if len(selectionOrder) > 0 {
					_ = cache.Put(constant.ProfilePrimary, selectionOrder[0])
				} else {
					_ = cache.Put(constant.ProfilePrimary, target.Id)
				}
			}
		} else {
			if primaryId == target.Id {
				if len(selectionOrder) > 0 {
					_ = cache.Put(constant.ProfilePrimary, selectionOrder[0])
				} else {
					for _, p := range profiles {
						if p.Id != target.Id && p.Selected {
							_ = cache.Put(constant.ProfilePrimary, p.Id)
							break
						}
					}
				}
			}
		}
	}

	internal.SwitchProfile(true)

	render.NoContent(w, r)
}
