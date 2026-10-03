package handlers

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// proxyDescriptionKeys — ключи описания прокси в порядке приоритета.
// serverDescription — основной (его отдаёт панель), остальные — запасные.
var proxyDescriptionKeys = []string{
	"serverDescription",
	"server_description",
	"server-description",
	"description",
}

// groupDescriptionKey — единственный ключ описания у proxy-groups.
const groupDescriptionKey = "description"

// parseProxyDescriptions собирает карту «имя → описание» из YAML профиля:
// описания прокси из блока proxies и описания групп из блока proxy-groups.
// Имена прокси и групп в mihomo уникальны, поэтому одна карта на оба блока.
// Пустые и состоящие из пробелов значения пропускаются; длина не ограничивается.
func parseProxyDescriptions(content string) map[string]string {
	descriptions := map[string]string{}

	rawConfig := map[string]any{}
	if err := yaml.Unmarshal([]byte(content), &rawConfig); err != nil {
		return descriptions
	}

	collectDescriptions(rawConfig["proxies"], proxyDescriptionKeys, descriptions)
	collectDescriptions(rawConfig["proxy-groups"], []string{groupDescriptionKey}, descriptions)

	return descriptions
}

func collectDescriptions(raw any, keys []string, out map[string]string) {
	items, ok := raw.([]any)
	if !ok {
		return
	}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, ok := entry["name"].(string)
		if !ok || name == "" {
			continue
		}
		for _, key := range keys {
			value, ok := entry[key].(string)
			if !ok {
				continue
			}
			if value = strings.TrimSpace(value); value != "" {
				out[name] = value
				break
			}
		}
	}
}
