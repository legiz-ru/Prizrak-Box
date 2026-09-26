package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func TestRemoteImageProxy(t *testing.T) {
	var hits atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/random": // random-image hosts answer with a redirect
			http.Redirect(w, r, "/img", http.StatusFound)
		case "/img":
			n := hits.Add(1)
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte{0xff, 0xd8, byte(n)}) // a different "picture" per request
		case "/html":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html></html>"))
		}
	}))
	defer origin.Close()

	mw := CustomBackgroundMiddleware(http.NotFoundHandler())
	get := func(target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, remoteImagePath+"?url="+url.QueryEscape(target), nil))
		return rec
	}

	first := get(origin.URL + "/random?a=1&date=1")
	if first.Code != http.StatusOK || first.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("first fetch: %d %q", first.Code, first.Header().Get("Content-Type"))
	}
	// The CSS background reloads the same URL: it must get the same picture.
	second := get(origin.URL + "/random?a=1&date=1")
	if second.Body.String() != first.Body.String() || hits.Load() != 1 {
		t.Fatalf("repeat fetch not served from cache (hits=%d)", hits.Load())
	}
	if got := get(origin.URL + "/html"); got.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("non-image: got %d", got.Code)
	}
	for _, bad := range []string{"file:///etc/passwd", "javascript:alert(1)", ""} {
		if got := get(bad); got.Code != http.StatusBadRequest {
			t.Fatalf("%q: got %d", bad, got.Code)
		}
	}
}
