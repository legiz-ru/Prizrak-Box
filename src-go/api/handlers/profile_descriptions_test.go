package handlers

import (
	"strings"
	"testing"
)

func TestParseProxyDescriptions(t *testing.T) {
	content := `
proxies:
  - name: node-primary
    type: vless
    serverDescription: "  Primary text  "
    server_description: ignored
    description: ignored too
  - name: node-snake
    type: vless
    server_description: Snake text
    description: ignored
  - name: node-kebab
    type: vless
    server-description: Kebab text
  - name: node-fallback
    type: vless
    description: Fallback text
  - name: node-blank-primary
    type: vless
    serverDescription: "   "
    description: Used after blank
  - name: node-none
    type: vless
proxy-groups:
  - name: 👻 Prizrak
    type: select
    description: Auto-pick the best location
    serverDescription: not read for groups
    proxies: [node-primary]
  - name: group-blank
    type: select
    description: "  "
  - name: group-none
    type: select
`
	got := parseProxyDescriptions(content)

	want := map[string]string{
		"node-primary":       "Primary text",
		"node-snake":         "Snake text",
		"node-kebab":         "Kebab text",
		"node-fallback":      "Fallback text",
		"node-blank-primary": "Used after blank",
		"👻 Prizrak":          "Auto-pick the best location",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d descriptions %v, want %d %v", len(got), got, len(want), want)
	}
	for name, desc := range want {
		if got[name] != desc {
			t.Errorf("description of %q = %q, want %q", name, got[name], desc)
		}
	}
}

func TestParseProxyDescriptionsNoLengthLimit(t *testing.T) {
	long := strings.Repeat("я", 100)
	got := parseProxyDescriptions("proxy-groups:\n  - name: g\n    type: select\n    description: " + long + "\n")
	if got["g"] != long {
		t.Errorf("long group description was altered: %q", got["g"])
	}
}

func TestParseProxyDescriptionsInvalid(t *testing.T) {
	for _, content := range []string{"", "not: [valid", "proxies: nope\nproxy-groups: 5", "proxies:\n  - just-a-string\n  - name: 5"} {
		if got := parseProxyDescriptions(content); len(got) != 0 {
			t.Errorf("parseProxyDescriptions(%q) = %v, want empty", content, got)
		}
	}
}
