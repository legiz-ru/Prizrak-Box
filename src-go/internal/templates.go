package internal

import (
	"fmt"

	"github.com/legiz-ru/prizrak-box/api"
	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/pkg/cache"
	"github.com/legiz-ru/prizrak-box/pkg/constant"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
)

// EnsureBuiltinTemplates synchronizes the bundled rule templates with the
// cached metadata and files on disk. It recreates missing records and files and
// forces an overwrite whenever the bundled version changes.
func EnsureBuiltinTemplates() {
	builtinBodies := [3][]byte{Template_0, Template_1, Template_2}
	builtinTitles := [3]string{"m1", "m2", "m3"}

	var storedVersion string
	_ = cache.Get(constant.TemplateBuiltinVersion, &storedVersion)

	needOverwrite := api.Version != "" && storedVersion != api.Version

	for i, body := range builtinBodies {
		id := fmt.Sprintf("%s%d", constant.PrefixTemplate, i)
		path := fmt.Sprintf("./%s/%s.yaml", constant.DefaultTemplateDir, id)
		fullPath := utils.GetUserHomeDir(path)

		var template models.Template
		if err := cache.Get(id, &template); err != nil || template.Id == "" {
			template = models.Template{
				Id:       id,
				Order:    int64(i),
				Title:    builtinTitles[i],
				Path:     path,
				Selected: false,
			}
			_ = cache.Put(id, template)
		} else {
			needUpdate := false
			if template.Path != path {
				template.Path = path
				needUpdate = true
			}
			if template.Title == "" {
				template.Title = builtinTitles[i]
				needUpdate = true
			}
			if needUpdate {
				_ = cache.Put(id, template)
			}
		}

		if needOverwrite {
			_, _ = utils.SaveFile(fullPath, body)
			continue
		}

		if !utils.FileExists(fullPath) {
			_, _ = utils.SaveFile(fullPath, body)
		}
	}

	if api.Version != "" && (needOverwrite || storedVersion == "") {
		_ = cache.Put(constant.TemplateBuiltinVersion, api.Version)
	}
}
