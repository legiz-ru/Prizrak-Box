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

func getProfile(w http.ResponseWriter, r *http.Request) {
	var res []models.Profile
	_ = cache.GetList(constant.PrefixProfile, &res)

	var order []models.Profile
	_ = cache.Get(constant.ProfileOrder, &order)

	// 创建一个 map 用于快速查找 order 中的元素
	orderMap := make(map[string]int)
	for index, item := range order {
		orderMap[item.Id] = index
	}

	// 对 res 进行排序
	sort.SliceStable(res, func(i, j int) bool {
		// 如果 res[i] 和 res[j] 都在 order 中，按 order 中的顺序排序
		indexI, existsI := orderMap[res[i].Id]
		indexJ, existsJ := orderMap[res[j].Id]
		if existsI && existsJ {
			return indexI < indexJ
		}
		// 如果只有一个在 order 中，优先排序在 order 中的
		if existsI {
			return true
		}
		if existsJ {
			return false
		}
		// 如果都不在 order 中，按 Order 字段排序
		return res[i].Order < res[j].Order
	})

	var primaryId string
	_ = cache.Get(constant.ProfilePrimary, &primaryId)

	primarySet := false
	if primaryId != "" {
		for i := range res {
			if res[i].Id == primaryId && res[i].Selected {
				primarySet = true
				break
			}
		}
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

	for i := range res {
		res[i].Primary = res[i].Id == primaryId
	}

	render.JSON(w, r, res)
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

		// 解析存盘
		subProfile := &models.Profile{
			Content: sub,
		}
		err = internal.Resolve(res.Body, subProfile, false)
		if err == nil {
			// 进行请求头解析
			mergedHeaders := internal.MergeHeaders(res.Headers, internal.ParseInlineHeaders(res.Body))
			internal.ParseHeaders(mergedHeaders, sub, subProfile)
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
	err = internal.Resolve(res.Body, profile, true)
	if err == nil {
		// 进行请求头解析
		mergedHeaders := internal.MergeHeaders(res.Headers, internal.ParseInlineHeaders(res.Body))
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

	render.JSON(w, r, profile)
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

		if desired {
			var primaryId string
			_ = cache.Get(constant.ProfilePrimary, &primaryId)
			primarySelected := false
			if primaryId != "" {
				for _, p := range profiles {
					if p.Id == primaryId && p.Selected {
						primarySelected = true
						break
					}
				}
			}
			if !primarySelected {
				_ = cache.Put(constant.ProfilePrimary, target.Id)
			}
		} else {
			var primaryId string
			_ = cache.Get(constant.ProfilePrimary, &primaryId)
			if primaryId == target.Id {
				for _, p := range profiles {
					if p.Id != target.Id && p.Selected {
						_ = cache.Put(constant.ProfilePrimary, p.Id)
						break
					}
				}
			}
		}
	}

	internal.SwitchProfile(true)

	render.NoContent(w, r)
}
