// Same-origin proxy for remote theme backgrounds.
//
// Random-image hosts used by the theme picker (anime, "random", …) mostly send
// no CORS headers. WKWebView (macOS) and WebKitGTK (Linux) then taint the
// canvas, so the frontend can neither pick the accent colour from the image nor
// cache it. WebView2 avoids this with --disable-web-security (see main.go);
// there is no such switch for WebKit, so the image is fetched here and served
// from the app's own origin instead: GET /remote-image?url=<http(s) URL>.
//
// The endpoint lives on the Wails asset server, which only the app's webview
// can reach. A fetched image is kept briefly in memory because random-image
// URLs return a different picture per request, while the frontend loads the
// same URL twice (once to analyse it, once as the CSS background).
package services

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	remoteImagePath     = "/remote-image"
	remoteImageMaxBytes = 15 << 20 // 15MB
	remoteImageTimeout  = 10 * time.Second
	remoteImageCacheLen = 4
	remoteImageCacheTTL = 10 * time.Minute
)

type remoteImage struct {
	url         string
	contentType string
	body        []byte
	fetchedAt   time.Time
}

var (
	remoteImageMu    sync.Mutex
	remoteImageCache []remoteImage // most recent last

	remoteImageClient = &http.Client{
		Timeout: remoteImageTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("unsupported redirect scheme")
			}
			return nil
		},
	}
)

func cachedRemoteImage(u string) (remoteImage, bool) {
	remoteImageMu.Lock()
	defer remoteImageMu.Unlock()
	for i := len(remoteImageCache) - 1; i >= 0; i-- {
		img := remoteImageCache[i]
		if img.url == u && time.Since(img.fetchedAt) < remoteImageCacheTTL {
			return img, true
		}
	}
	return remoteImage{}, false
}

func storeRemoteImage(img remoteImage) {
	remoteImageMu.Lock()
	defer remoteImageMu.Unlock()
	kept := remoteImageCache[:0]
	for _, c := range remoteImageCache {
		if c.url != img.url && time.Since(c.fetchedAt) < remoteImageCacheTTL {
			kept = append(kept, c)
		}
	}
	kept = append(kept, img)
	if len(kept) > remoteImageCacheLen {
		kept = kept[len(kept)-remoteImageCacheLen:]
	}
	remoteImageCache = kept
}

func fetchRemoteImage(r *http.Request, target string) (remoteImage, int) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		return remoteImage{}, http.StatusBadRequest
	}
	req.Header.Set("Accept", "image/*")
	resp, err := remoteImageClient.Do(req)
	if err != nil {
		return remoteImage{}, http.StatusBadGateway
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return remoteImage{}, http.StatusBadGateway
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return remoteImage{}, http.StatusUnsupportedMediaType
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, remoteImageMaxBytes+1))
	if err != nil {
		return remoteImage{}, http.StatusBadGateway
	}
	if len(body) > remoteImageMaxBytes {
		return remoteImage{}, http.StatusRequestEntityTooLarge
	}
	return remoteImage{url: target, contentType: contentType, body: body, fetchedAt: time.Now()}, http.StatusOK
}

func handleRemoteImage(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	img, ok := cachedRemoteImage(target)
	if !ok {
		var status int
		img, status = fetchRemoteImage(r, target)
		if status != http.StatusOK {
			http.Error(w, http.StatusText(status), status)
			return
		}
		storeRemoteImage(img)
	}

	w.Header().Set("Content-Type", img.contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(img.body)))
	w.Header().Set("Cache-Control", "private, max-age=600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(img.body)
}
