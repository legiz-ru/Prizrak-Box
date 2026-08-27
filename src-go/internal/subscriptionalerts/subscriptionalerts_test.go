package subscriptionalerts

import "testing"

const day = dayMillis

func TestEvaluate_NoThresholds_NoAlerts(t *testing.T) {
	alerts, next := Evaluate(1000*day, 0, 0, nil, nil, nil, 0)
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %v", alerts)
	}
	if len(next) != 0 {
		t.Fatalf("expected no dedup state, got %v", next)
	}
}

func TestEvaluate_ExpiresIn_FiresOnceThenDedups(t *testing.T) {
	expireAt := int64(10 * day)
	now := int64(5 * day) // 5 days remaining, crosses the 7-day threshold

	alerts, next := Evaluate(expireAt, 0, 0, []int{1, 3, 7}, nil, nil, now)
	if len(alerts) != 1 || alerts[0].Kind != KindExpiresIn || alerts[0].Days != 7 {
		t.Fatalf("expected a single expires_in(7) alert, got %v", alerts)
	}

	// Same state again: already shown, must not refire.
	alerts2, next2 := Evaluate(expireAt, 0, 0, []int{1, 3, 7}, nil, next, now)
	if len(alerts2) != 0 {
		t.Fatalf("expected no repeat alert, got %v", alerts2)
	}
	if next2[expireKey(7)] == 0 {
		t.Fatalf("expected expire_7d to stay recorded")
	}
}

func TestEvaluate_AtMostOneAlertPerPass(t *testing.T) {
	expireAt := int64(10 * day)
	now := int64(11 * day) // already past every threshold and the deadline itself

	alerts, _ := Evaluate(expireAt, 0, 0, []int{1, 3, 7}, nil, nil, now)
	if len(alerts) != 1 || alerts[0].Kind != KindExpired {
		t.Fatalf("expected exactly one 'expired' alert when every threshold is missed at once, got %v", alerts)
	}
}

func TestEvaluate_RenewalResetsDedup(t *testing.T) {
	expireAt := int64(10 * day)
	now := int64(5 * day)

	_, next := Evaluate(expireAt, 0, 0, []int{7}, nil, nil, now)
	if next[expireKey(7)] == 0 {
		t.Fatalf("expected expire_7d recorded after first evaluate")
	}

	// Subscription renewed: new, later expiry.
	newExpireAt := int64(30 * day)
	alerts, next2 := Evaluate(newExpireAt, 0, 0, []int{7}, nil, next, now)
	if len(alerts) != 0 {
		t.Fatalf("25 days remaining should not cross a 7-day threshold, got %v", alerts)
	}
	if _, ok := next2[expireKey(7)]; ok {
		t.Fatalf("expected expire_7d to be cleared after the deadline moved, got %v", next2)
	}
	if next2[expireBaseKey] != newExpireAt {
		t.Fatalf("expected expire_base to track the new deadline")
	}
}

func TestEvaluate_TrafficUsesHighestReachedThreshold(t *testing.T) {
	alerts, next := Evaluate(0, 100, 95, nil, []int{80, 90, 100}, nil, 0)
	if len(alerts) != 1 || alerts[0].Kind != KindTrafficUsed || alerts[0].Percent != 90 {
		t.Fatalf("expected a single traffic_used(90) alert (highest reached, 100%% not yet hit), got %v", alerts)
	}
	if _, ok := next[trafficKey(100)]; ok {
		t.Fatalf("100%% threshold was not reached and must not be marked shown")
	}
}

func TestEvaluate_TrafficResetsWhenUsageDropsBack(t *testing.T) {
	_, next := Evaluate(0, 100, 85, nil, []int{80}, nil, 0)
	if _, ok := next[trafficKey(80)]; !ok {
		t.Fatalf("expected traffic_80 recorded")
	}

	// New billing period, usage reset.
	alerts, next2 := Evaluate(0, 100, 5, nil, []int{80}, next, 0)
	if len(alerts) != 0 {
		t.Fatalf("expected no alert once usage drops back under the threshold, got %v", alerts)
	}
	if _, ok := next2[trafficKey(80)]; ok {
		t.Fatalf("expected traffic_80 to be un-marked once usage dropped back under it")
	}
}

func TestEvaluate_NoExpiry_ClearsExpiryState(t *testing.T) {
	// total == 0 (quota unknown) intentionally leaves traffic_80 untouched:
	// the traffic block only runs when total > 0, matching the Android
	// reference — a transient "quota unknown" read must not silently forget
	// that the 80% threshold was already shown.
	stale := map[string]int64{expireBaseKey: 123, expireKey(7): 456, trafficKey(80): 789}
	alerts, next := Evaluate(0, 0, 0, []int{7}, []int{80}, stale, 0)
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts with no known expiry, got %v", alerts)
	}
	if _, ok := next[expireBaseKey]; ok {
		t.Fatalf("expected expire_base cleared when expireAt is unknown, got %v", next)
	}
	if _, ok := next[expireKey(7)]; ok {
		t.Fatalf("expected expire_7d cleared when expireAt is unknown, got %v", next)
	}
	if _, ok := next[trafficKey(80)]; !ok {
		t.Fatalf("expected traffic_80 to survive (total unknown, not total == 0 usage), got %v", next)
	}
}
