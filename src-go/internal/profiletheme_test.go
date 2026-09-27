package internal

import (
	"encoding/base64"
	"testing"
)

func TestParseProfileTheme(t *testing.T) {
	th := parseProfileTheme("image=https://cdn.example.com/bg.jpg?a=1&b=2; transparency=30; blur=4px; dim=20%; mode=Dark; accent=#4F8CFF")
	if th == nil || th.Image != "https://cdn.example.com/bg.jpg?a=1&b=2" || *th.Transparency != 30 || *th.Blur != 4 ||
		*th.Dim != 20 || th.Mode != "dark" || th.Accent != "#4f8cff" {
		t.Fatalf("full header parsed wrong: %+v", th)
	}

	// Clamping, explicit zero, unknown keys, invalid values.
	th = parseProfileTheme("transparency=200; blur=0; dim=-5; mode=neon; accent=blue; foo=bar; image=ftp://x/y.jpg")
	if th == nil || *th.Transparency != 85 || th.Blur == nil || *th.Blur != 0 || *th.Dim != 0 ||
		th.Mode != "" || th.Accent != "" || th.Image != "" {
		t.Fatalf("clamping/validation wrong: %+v", th)
	}

	// Colour + mode only, no image.
	th = parseProfileTheme("mode=light; accent=#1f9e7a")
	if th == nil || th.Image != "" || th.Mode != "light" || th.Accent != "#1f9e7a" || th.Transparency != nil {
		t.Fatalf("mode/accent only wrong: %+v", th)
	}

	// base64: prefix, like announce.
	enc := base64.StdEncoding.EncodeToString([]byte("image=https://cdn.example.com/a%3Bb.jpg; accent=auto"))
	th = parseProfileTheme("base64:" + enc)
	if th == nil || th.Image != "https://cdn.example.com/a%3Bb.jpg" || th.Accent != "auto" {
		t.Fatalf("base64 value not decoded: %+v", th)
	}

	for _, empty := range []string{"", "   ", "foo=bar", "base64:!!!", "transparency=abc"} {
		if th := parseProfileTheme(empty); th != nil {
			t.Fatalf("%q: expected nil, got %+v", empty, th)
		}
	}
}
