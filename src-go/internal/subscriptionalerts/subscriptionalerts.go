// Package subscriptionalerts decides which subscription-state alerts —
// "expires in N days", "expired", "N% of traffic used" — are newly due for a
// profile, from a plain snapshot of its subscription state and of what has
// already been shown. It knows nothing about HTTP headers, notifications, or
// storage; those are the caller's job (see internal.ParseHeaders for the
// header parsing and api/job/refresh.go for wiring Evaluate into the refresh
// cycle), which keeps this package trivial to reason about and to test.
//
// Ported from the reference Android implementation
// (service/subscription/SubscriptionAlerts.kt in legiz-ru/Prizrak-Box-android,
// branch claude/prizrak-moshen-muxcool) — see docs/.../supported-headers.mdx
// for the user-facing behavior this implements.
package subscriptionalerts

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultExpireDays is used in two different places, opting in differently:
//   - internal.ParseHeaders applies it only when the panel sent the bare
//     `notification-subs-expire: true` toggle without an explicit
//     notify-expire-days list — never as a substitute for the header being
//     absent entirely.
//   - the desktop UI's "renew subscription" nudge button (the frontend's
//     showRenewButton) uses it even with no opt-in header at all: the one
//     place in the whole feature where an absent header does not disable the
//     behavior, because the button is a UI nudge, not a push notification.
var DefaultExpireDays = []int{1, 3, 7}

const (
	// Alert kinds.
	KindExpired     = "expired"
	KindExpiresIn   = "expires_in"
	KindTrafficUsed = "traffic_used"

	expiredKey    = "expired"
	expireBaseKey = "expire_base"

	dayMillis = int64(24 * 60 * 60 * 1000)
)

func expireKey(days int) string { return fmt.Sprintf("expire_%dd", days) }

func trafficKey(percent int) string { return fmt.Sprintf("traffic_%d", percent) }

// Alert is a single subscription-state alert due to be shown.
type Alert struct {
	Kind    string `json:"kind"`
	Days    int    `json:"days,omitempty"`    // set when Kind == KindExpiresIn
	Percent int    `json:"percent,omitempty"` // set when Kind == KindTrafficUsed
}

// Evaluate decides which alerts are newly due, given the subscription's
// current state and the dedup state from the previous evaluation.
//
// expireAtMillis == 0 means no expiry date is known; total == 0 means no
// traffic quota is known. nowMillis should already carry the panel
// clock-skew correction (see internal.ParseHeaders) — this function just
// compares numbers, it doesn't know or care where they came from.
//
// Returns the alerts to show (empty if none) and the updated dedup state to
// persist on the profile (models.Profile.NotifiedAlerts) regardless of
// whether any alert fired — a threshold crossed while the "Subscription
// reminders" setting is off must still be marked "shown", or turning the
// setting back on would flood the user with every threshold missed while it
// was off.
func Evaluate(
	expireAtMillis int64,
	total, used int64,
	expireDays, trafficPercent []int,
	notified map[string]int64,
	nowMillis int64,
) (alerts []Alert, next map[string]int64) {
	next = make(map[string]int64, len(notified))
	for k, v := range notified {
		next[k] = v
	}

	// Drop bookkeeping for thresholds that no longer apply — the panel
	// changed its mind about which days/percentages matter.
	valid := func(key string) bool {
		if key == expireBaseKey {
			return true
		}
		if key == expiredKey {
			return len(expireDays) > 0
		}
		for _, d := range expireDays {
			if key == expireKey(d) {
				return true
			}
		}
		for _, p := range trafficPercent {
			if key == trafficKey(p) {
				return true
			}
		}
		return false
	}
	for k := range next {
		if !valid(k) {
			delete(next, k)
		}
	}

	if expireAtMillis != 0 && len(expireDays) > 0 {
		// The expiry moment itself changed (renewal, panel clock fixed) —
		// every prior "shown" mark for this category is about a different
		// deadline and means nothing now.
		if next[expireBaseKey] != expireAtMillis {
			for k := range next {
				if k == expiredKey || strings.HasPrefix(k, "expire_") {
					delete(next, k)
				}
			}
			next[expireBaseKey] = expireAtMillis
		}

		remaining := expireAtMillis - nowMillis

		// Time moved back across a threshold (renewal) — rearm it so the
		// next real crossing notifies again instead of staying "shown".
		if remaining > 0 {
			delete(next, expiredKey)
		}
		for _, days := range expireDays {
			if remaining > int64(days)*dayMillis {
				delete(next, expireKey(days))
			}
		}

		type passedEntry struct {
			key   string
			alert *Alert
		}
		var passed []passedEntry
		if remaining <= 0 {
			passed = append(passed, passedEntry{expiredKey, &Alert{Kind: KindExpired}})
		}
		sortedDays := append([]int(nil), expireDays...)
		sort.Ints(sortedDays)
		for _, days := range sortedDays {
			if remaining > 0 && remaining <= int64(days)*dayMillis {
				passed = append(passed, passedEntry{expireKey(days), &Alert{Kind: KindExpiresIn, Days: days}})
			} else if remaining <= 0 {
				passed = append(passed, passedEntry{expireKey(days), nil})
			}
		}

		// At most one alert per category per pass: a subscription that sat
		// untouched past every threshold at once (first run after a long
		// gap) should say "expired" once, not "7 days / 3 days / 1 day /
		// expired" back to back.
		shown := false
		for _, p := range passed {
			if _, ok := next[p.key]; !ok {
				if !shown && p.alert != nil {
					alerts = append(alerts, *p.alert)
					shown = true
				}
				next[p.key] = nowMillis
			} else if p.alert != nil {
				shown = true
			}
		}
	} else if expireAtMillis == 0 {
		for k := range next {
			if k == expiredKey || k == expireBaseKey || strings.HasPrefix(k, "expire_") {
				delete(next, k)
			}
		}
	}

	if total > 0 && len(trafficPercent) > 0 {
		if used < 0 {
			used = 0
		}
		reached := func(threshold int) bool { return percentReached(used, total, threshold) }

		for _, threshold := range trafficPercent {
			if !reached(threshold) {
				delete(next, trafficKey(threshold))
			}
		}

		sortedDesc := append([]int(nil), trafficPercent...)
		sort.Sort(sort.Reverse(sort.IntSlice(sortedDesc)))
		shown := false
		for _, threshold := range sortedDesc {
			if !reached(threshold) {
				continue
			}
			key := trafficKey(threshold)
			if _, ok := next[key]; !ok {
				if !shown {
					alerts = append(alerts, Alert{Kind: KindTrafficUsed, Percent: threshold})
					shown = true
				}
				next[key] = nowMillis
			} else {
				shown = true
			}
		}
	}

	return alerts, next
}

// percentReached mirrors the Android reference's integer-only arithmetic
// (avoids float precision surprises right at a threshold boundary).
func percentReached(used, total int64, threshold int) bool {
	whole := used / total * 100
	rest := used % total * 100 / total
	return whole+rest >= int64(threshold)
}
