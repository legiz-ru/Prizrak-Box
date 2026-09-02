package internal

import (
	"time"

	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/internal/subscriptionalerts"
)

// EvaluateSubscriptionAlerts runs subscriptionalerts.Evaluate for one profile
// and folds the result back onto it — NotifiedAlerts (the dedup state used on
// the next call) is always updated, and PendingAlerts gains any alerts newly
// due this pass, for the frontend to pick up and turn into a native
// notification (see api/handlers/profile.go's ackSubscriptionAlert).
//
// Called right after ParseHeaders, wherever that already runs (api/job/refresh.go's
// DoRefresh, and the refresh/add handlers in api/handlers/profile.go) — no
// separate schedule of its own, so it inherits exactly the same periodicity
// as subscription refreshing itself, including a profile's own
// Profile-Update-Interval.
func EvaluateSubscriptionAlerts(profile *models.Profile) {
	if len(profile.NotifyExpireDays) == 0 && len(profile.NotifyTrafficPercent) == 0 {
		// The panel never opted this profile into either reminder kind (or
		// stopped sending the headers) — nothing to track, and leftover
		// dedup state would just be dead weight.
		profile.NotifiedAlerts = nil
		return
	}

	var total, used int64
	if profile.Total != nil {
		total = profile.Total.Int64()
	}
	if profile.Used != nil {
		used = profile.Used.Int64()
	}

	now := time.Now().UnixMilli() + profile.ClockSkewMillis()

	alerts, next := subscriptionalerts.Evaluate(
		profile.ExpireMillis(),
		total, used,
		profile.NotifyExpireDays, profile.NotifyTrafficPercent,
		profile.NotifiedAlerts,
		now,
	)
	profile.NotifiedAlerts = next
	if len(alerts) > 0 {
		profile.PendingAlerts = append(profile.PendingAlerts, alerts...)
	}
}
