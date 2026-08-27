package models

import (
	"math/big"
	"time"

	"github.com/legiz-ru/prizrak-box/internal/subscriptionalerts"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
)

type Profile struct {
	Id                 string   `json:"id"`
	Type               int      `json:"type"` // 1: 远程订阅 2：本地配置 3：爬取合并
	Title              string   `json:"title,omitempty"`
	HeaderTitle        string   `json:"headerTitle,omitempty"`
	Order              string   `json:"order"`
	Selected           bool     `json:"selected,omitempty"`
	Primary            bool     `json:"primary,omitempty"`
	SelectionOrder     int      `json:"selectionOrder,omitempty"`
	Path               string   `json:"path"`
	Content            string   `json:"content,omitempty"`
	Used               *big.Int `json:"used,omitempty"`
	Available          *big.Int `json:"available,omitempty"`
	Total              *big.Int `json:"total,omitempty"`
	Expire             string   `json:"expire,omitempty"`
	Interval           string   `json:"interval,omitempty"`
	Home               string   `json:"home,omitempty"`
	Support            string   `json:"support,omitempty"`
	Logo               string   `json:"logo,omitempty"`
	Announce           string   `json:"announce,omitempty"`
	AnnounceUrl        string   `json:"announceUrl,omitempty"`
	Update             string   `json:"update,omitempty"`
	Template           string   `json:"template,omitempty"`
	PxdTemplateUrl     string   `json:"pxdTemplateUrl,omitempty"`
	PxdTemplateScheme  string   `json:"pxdTemplateScheme,omitempty"`
	HwidActive         bool     `json:"hwidActive,omitempty"`
	AgeSecretKey       string   `json:"ageSecretKey,omitempty"`
	FallbackUrl        string   `json:"fallbackUrl,omitempty"`
	FallbackDomain     string   `json:"fallbackDomain,omitempty"`
	GlobalModeDisabled bool     `json:"globalModeDisabled,omitempty"`

	// RenewUrl — subscription-renew-url. Absent header means every renew-related
	// UI element (profile editor, toolbar/list icons, the renew button under the
	// announce) is simply not shown.
	RenewUrl string `json:"renewUrl,omitempty"`

	// NotifyExpireDays / NotifyTrafficPercent — notify-expire-days /
	// notify-traffic-percent thresholds (see internal.ParseHeaders). nil means
	// the panel opted out of that reminder kind entirely; the renew button (not
	// push notifications) falls back to subscriptionalerts.DefaultExpireDays
	// when NotifyExpireDays is nil — see ActiveProfile.vue's showRenewButton.
	NotifyExpireDays     []int `json:"notifyExpireDays,omitempty"`
	NotifyTrafficPercent []int `json:"notifyTrafficPercent,omitempty"`

	// NotifiedAlerts — subscriptionalerts.Evaluate's "already shown" bookkeeping
	// for this profile (expire_base/expire_Nd/expired/traffic_N -> unix millis).
	// Persisted via the same cache.Put(profile) as everything else on this
	// struct (pkg/cache round-trips through encoding/json too), so it rides
	// along in API responses as harmless dedup bookkeeping — nothing in the
	// frontend reads it.
	NotifiedAlerts map[string]int64 `json:"notifiedAlerts,omitempty"`

	// ClockSkewSeconds/ClockSkewAtSeconds — how far the panel's clock (from the
	// standard Date response header) was ahead of the device's, and when that
	// was measured (device time). See internal.ParseHeaders and
	// subscriptionalerts for the correction and its 30-day staleness cutoff.
	ClockSkewSeconds   int64 `json:"clockSkewSeconds,omitempty"`
	ClockSkewAtSeconds int64 `json:"clockSkewAtSeconds,omitempty"`

	// PendingAlerts — alerts newly due as of the last subscriptionalerts.Evaluate
	// pass (see internal.EvaluateSubscriptionAlerts, called after every
	// ParseHeaders), not yet delivered to the frontend as a native OS
	// notification. Cleared by the frontend via PUT /prizrak/ackSubscriptionAlert
	// once it has acted on them (or decided, per the local "Subscription
	// reminders" setting, not to) — "read and ack" rather than "read clears it",
	// since NotifiedAlerts above already guarantees each threshold fires at most
	// once no matter when the frontend happens to pick it up.
	PendingAlerts []subscriptionalerts.Alert `json:"pendingAlerts,omitempty"`
}

func (p *Profile) GetUpdateTime() time.Time {
	dateTime, _ := utils.ParseDateTime(p.Update)
	return dateTime
}

func (p *Profile) SetUpdateTime() {
	p.Update = utils.GetDateTime()
}

const maxClockSkewAgeSeconds = int64(30 * 24 * 60 * 60)

// ClockSkewMillis returns the panel-clock correction in milliseconds, derived
// from the standard Date response header (see internal.ParseHeaders), or 0
// when there is none to apply. A measurement older than 30 days is discarded
// rather than used: what's dangerous is not crystal drift (seconds a month)
// but the device clock having been corrected since — by the user, or by time
// sync after a reboot — which turns a stale correction into an error of its
// own size. A negative age (measured "in the future") means the same thing
// and is discarded the same way.
func (p *Profile) ClockSkewMillis() int64 {
	if p.ClockSkewSeconds == 0 || p.ClockSkewAtSeconds == 0 {
		return 0
	}

	ageSeconds := time.Now().Unix() - p.ClockSkewAtSeconds
	if ageSeconds < 0 || ageSeconds > maxClockSkewAgeSeconds {
		return 0
	}

	return p.ClockSkewSeconds * 1000
}

// ExpireMillis parses Expire (the same "2006-01-02 15:04" local-time layout
// as Update, see utils.ParseDateTime) into unix milliseconds, or 0 when
// Expire is empty or unparseable — the same "no expiry known" sentinel
// subscriptionalerts.Evaluate expects.
func (p *Profile) ExpireMillis() int64 {
	if p.Expire == "" {
		return 0
	}

	t, err := utils.ParseDateTime(p.Expire)
	if err != nil {
		return 0
	}

	return t.UnixMilli()
}
