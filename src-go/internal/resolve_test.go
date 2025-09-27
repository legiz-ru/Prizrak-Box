package internal

import (
	"net/http"
	"testing"

	"github.com/legiz-ru/prizrak-box/api/models"
)

func TestParseHeadersUnlimitedTraffic(t *testing.T) {
	header := http.Header{}
	header.Set("Subscription-Userinfo", "total=0; upload=123; download=456; expire=0")

	profile := &models.Profile{}
	ParseHeaders(header, "https://example.com/path", profile)

	if profile.Total != nil {
		t.Fatalf("expected total to be nil for unlimited subscriptions, got %v", profile.Total)
	}
	if profile.Used != nil {
		t.Fatalf("expected used to be nil for unlimited subscriptions, got %v", profile.Used)
	}
	if profile.Available != nil {
		t.Fatalf("expected available to be nil for unlimited subscriptions, got %v", profile.Available)
	}
}

func TestParseHeadersProfileTitleOverridesFilename(t *testing.T) {
	header := http.Header{}
	header.Set("Profile-Title", "Example Title")

	profile := &models.Profile{}
	ParseHeaders(header, "https://example.com/subscription.txt", profile)

	if profile.Title != "Example Title" {
		t.Fatalf("expected profile title to come from header, got %q", profile.Title)
	}
}
