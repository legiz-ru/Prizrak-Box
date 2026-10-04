package utils

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGenerateHWID(t *testing.T) {
	resetCachedDeviceDetailsForTest()

	hwid1 := generateHWID()
	hwid2 := generateHWID()

	if hwid1 == "" {
		t.Fatal("HWID should not be empty")
	}

	if hwid1 != hwid2 {
		t.Errorf("HWID should be consistent across calls, got '%s' and '%s'", hwid1, hwid2)
	}
}

func TestBuildDeviceHeaders(t *testing.T) {
	resetCachedDeviceDetailsForTest()

	// Test with HWID disabled — no device headers expected
	config := &HTTPClientConfig{
		EnableHWID: false,
	}
	UpdateHTTPClientConfig(config)

	headers := buildDeviceHeaders()
	if headers != nil {
		t.Errorf("Expected nil headers when HWID is disabled, got %v", headers)
	}

	// Test with HWID enabled — all device headers must be present
	config = &HTTPClientConfig{
		EnableHWID:  true,
		DeviceOS:    "Linux",
		DeviceOSVer: "5.4.0",
		DeviceModel: "TestDevice",
	}
	UpdateHTTPClientConfig(config)

	headers = buildDeviceHeaders()
	if headers == nil {
		t.Fatal("Expected headers when HWID is enabled, got nil")
	}

	if _, exists := headers["x-hwid"]; !exists {
		t.Error("Expected x-hwid header")
	}

	if headers["x-device-os"] != "Linux" {
		t.Errorf("Expected x-device-os to be 'Linux', got '%s'", headers["x-device-os"])
	}

	if headers["x-ver-os"] != "5.4.0" {
		t.Errorf("Expected x-ver-os to be '5.4.0', got '%s'", headers["x-ver-os"])
	}

	if headers["x-device-model"] != "TestDevice" {
		t.Errorf("Expected x-device-model to be 'TestDevice', got '%s'", headers["x-device-model"])
	}

	config = &HTTPClientConfig{
		EnableHWID: true,
		DeviceOS:   "Windows x64",
	}
	UpdateHTTPClientConfig(config)

	headers = buildDeviceHeaders()
	if headers["x-device-os"] != "Windows" {
		t.Errorf("Expected x-device-os to be 'Windows', got '%s'", headers["x-device-os"])
	}
}

func TestUpdateHTTPClientConfig(t *testing.T) {
	// Определяем ожидаемое имя ОС для текущей платформы.
	expectedOS := defaultOSName()

	// Test: UA должен иметь единый формат с версией, HWID enabled
	config := &HTTPClientConfig{
		EnableHWID: true,
		Version:    "1.0.1",
	}
	UpdateHTTPClientConfig(config)

	ua := globalConfig.UserAgent
	expectedPrefix := "prizrak-box/1.0.1 (Desktop Build; "
	if !strings.HasPrefix(ua, expectedPrefix) {
		t.Errorf("Expected UA to start with %q, got %q", expectedPrefix, ua)
	}
	if !strings.Contains(ua, expectedOS) {
		t.Errorf("Expected UA to contain OS %q, got %q", expectedOS, ua)
	}

	// Test: UA должен иметь единый формат с версией, HWID disabled
	config = &HTTPClientConfig{
		EnableHWID: false,
		Version:    "1.0.1",
	}
	UpdateHTTPClientConfig(config)

	ua = globalConfig.UserAgent
	if !strings.HasPrefix(ua, expectedPrefix) {
		t.Errorf("Expected UA to start with %q even when HWID disabled, got %q", expectedPrefix, ua)
	}

	// Test: UA без версии — версия "unknown", слэш после имени сохраняется
	config = &HTTPClientConfig{
		EnableHWID: false,
	}
	UpdateHTTPClientConfig(config)

	ua = globalConfig.UserAgent
	if !strings.HasPrefix(ua, "prizrak-box/unknown (Desktop Build; ") {
		t.Errorf("Expected UA to start with 'prizrak-box/unknown (Desktop Build; ', got %q", ua)
	}
	if !strings.Contains(ua, expectedOS) {
		t.Errorf("Expected UA to contain OS %q, got %q", expectedOS, ua)
	}
}

func TestBuildUserAgent(t *testing.T) {
	cases := []struct {
		version  string
		deviceOS string
		wantContains []string
	}{
		{"1.2.3", "Windows", []string{"prizrak-box/1.2.3 (", "Desktop Build", "Windows OS"}},
		{"2.0.0", "Linux", []string{"prizrak-box/2.0.0 (", "Desktop Build", "Linux OS"}},
		{"", "macOS", []string{"prizrak-box/unknown (", "Desktop Build", "macOS OS"}},
		{"1.0.0", "", []string{"prizrak-box/1.0.0 (", "Desktop Build", defaultOSName() + " OS"}},
	}

	for _, c := range cases {
		ua := buildUserAgent(c.version, c.deviceOS)
		for _, want := range c.wantContains {
			if !strings.Contains(ua, want) {
				t.Errorf("buildUserAgent(%q, %q) = %q, want it to contain %q", c.version, c.deviceOS, ua, want)
			}
		}
	}

	_ = runtime.GOOS // убеждаемся что пакет runtime используется
}

func TestBuildUserAgentCoreVersion(t *testing.T) {
	saved := coreVersion
	defer func() { coreVersion = saved }()

	coreVersion = "v1.19.32-r1"
	if got, want := buildUserAgent("1.2.3", "Windows"), "prizrak-box/1.2.3 (Desktop Build; Windows OS; Prizrak-Core v1.19.32-r1)"; got != want {
		t.Errorf("buildUserAgent = %q, want %q", got, want)
	}

	coreVersion = ""
	if got, want := buildUserAgent("1.2.3", "Windows"), "prizrak-box/1.2.3 (Desktop Build; Windows OS)"; got != want {
		t.Errorf("buildUserAgent = %q, want %q", got, want)
	}
}

// Remnawave sends serverDescription only to "extended" clients, recognised by
// the regex /^prizrak-box\// on the User-Agent (remnawave/backend,
// extended-clients.ts). The UA must keep matching it.
func TestBuildUserAgentMatchesRemnawaveExtendedClient(t *testing.T) {
	extended := regexp.MustCompile(`^prizrak-box/`)
	for _, version := range []string{"1.0.21-beta06", ""} {
		if ua := buildUserAgent(version, "Windows"); !extended.MatchString(ua) {
			t.Errorf("buildUserAgent(%q) = %q does not match %v", version, ua, extended)
		}
	}
}

// fakeRoute is a server standing in for one route to the panel: the panel
// itself (direct) or the local proxy (which here simply answers).
type fakeRoute struct {
	*httptest.Server
	hits atomic.Int32
}

func newFakeRoute(t *testing.T, status int, body string, delay time.Duration) *fakeRoute {
	t.Helper()
	r := &fakeRoute{}
	r.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.hits.Add(1)
		select {
		case <-time.After(delay):
		case <-req.Context().Done():
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(r.Close)
	return r
}

func shortRouteDelay(t *testing.T) time.Duration {
	t.Helper()
	saved := subscriptionRouteDelay
	subscriptionRouteDelay = 200 * time.Millisecond
	t.Cleanup(func() { subscriptionRouteDelay = saved })
	return subscriptionRouteDelay
}

func TestFetchSubscriptionCandidateNoProxyIsOneRequest(t *testing.T) {
	panel := newFakeRoute(t, 200, "config", 0)

	res, route, err := FetchSubscriptionCandidate(panel.URL, "", "")
	if err != nil || res == nil || res.Body != "config" {
		t.Fatalf("got %v %v", res, err)
	}
	if route != "" {
		t.Errorf("route = %q, want empty: there was no choice to remember", route)
	}
	if got := panel.hits.Load(); got != 1 {
		t.Errorf("panel hits = %d, want 1", got)
	}
}

func TestFetchSubscriptionCandidatePreferredRouteAloneIsOneRequest(t *testing.T) {
	shortRouteDelay(t)

	for _, preferred := range []string{"", SubscriptionRouteDirect, SubscriptionRouteProxy} {
		panel := newFakeRoute(t, 200, "direct", 0)
		proxy := newFakeRoute(t, 200, "proxy", 0)

		res, route, err := FetchSubscriptionCandidate(panel.URL, proxy.URL, preferred)
		if err != nil || res == nil {
			t.Fatalf("preferred %q: %v", preferred, err)
		}

		want, wantBody, otherHits := SubscriptionRouteDirect, "direct", proxy.hits.Load
		if preferred == SubscriptionRouteProxy {
			want, wantBody, otherHits = SubscriptionRouteProxy, "proxy", panel.hits.Load
		}
		if route != want || res.Body != wantBody {
			t.Errorf("preferred %q: route %q body %q, want %q %q", preferred, route, res.Body, want, wantBody)
		}
		// Give a wrongly started second route time to show up.
		time.Sleep(300 * time.Millisecond)
		if otherHits() != 0 {
			t.Errorf("preferred %q: the other route was used (%d hits)", preferred, otherHits())
		}
	}
}

func TestFetchSubscriptionCandidateSlowRouteHandsOverAfterDelay(t *testing.T) {
	delay := shortRouteDelay(t)
	panel := newFakeRoute(t, 200, "direct", 3*time.Second)
	proxy := newFakeRoute(t, 200, "proxy", 0)

	start := time.Now()
	res, route, err := FetchSubscriptionCandidate(panel.URL, proxy.URL, SubscriptionRouteDirect)
	elapsed := time.Since(start)

	if err != nil || res == nil || res.Body != "proxy" || route != SubscriptionRouteProxy {
		t.Fatalf("got %v %q %v", res, route, err)
	}
	if elapsed < delay || elapsed > 2*time.Second {
		t.Errorf("handover took %v, want about %v", elapsed, delay)
	}
}

func TestFetchSubscriptionCandidateFailedRouteHandsOverAtOnce(t *testing.T) {
	delay := shortRouteDelay(t)
	panel := newFakeRoute(t, 500, "boom", 0)
	proxy := newFakeRoute(t, 200, "proxy", 0)

	start := time.Now()
	res, route, err := FetchSubscriptionCandidate(panel.URL, proxy.URL, SubscriptionRouteDirect)
	elapsed := time.Since(start)

	if err != nil || res == nil || route != SubscriptionRouteProxy {
		t.Fatalf("got %v %q %v", res, route, err)
	}
	if elapsed >= delay {
		t.Errorf("a failed route should not wait out the delay, took %v", elapsed)
	}
}

func TestFetchSubscriptionCandidateAllRoutesFail(t *testing.T) {
	shortRouteDelay(t)
	panel := newFakeRoute(t, 500, "boom", 0)
	proxy := newFakeRoute(t, 503, "boom", 0)

	if _, _, err := FetchSubscriptionCandidate(panel.URL, proxy.URL, ""); err == nil {
		t.Fatal("expected an error")
	}
}
