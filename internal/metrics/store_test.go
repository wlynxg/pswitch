package metrics

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsAndAggregatesProviderUsage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	store.RecordSuccess("alpha", Usage{
		RequestCount: 1,
		InputTokens:  120,
		OutputTokens: 30,
		TotalTokens:  150,
	}, "gpt-5.4", now)
	store.RecordSuccess("alpha", Usage{
		RequestCount: 1,
		InputTokens:  80,
		OutputTokens: 20,
		TotalTokens:  100,
	}, "gpt-5.4", now.Add(-2*time.Hour))
	store.RecordSuccess("beta", Usage{
		RequestCount: 1,
		InputTokens:  50,
		OutputTokens: 10,
		TotalTokens:  60,
	}, "gpt-4.1", now.Add(-26*time.Hour))

	reloaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := reloaded.Snapshot(now)
	if got, want := snapshot.Overview.TotalRequests, int64(3); got != want {
		t.Fatalf("total requests = %d, want %d", got, want)
	}
	if got, want := snapshot.Overview.TotalTokens, int64(310); got != want {
		t.Fatalf("total tokens = %d, want %d", got, want)
	}
	if got, want := snapshot.Windows.Last24h.TotalTokens, int64(250); got != want {
		t.Fatalf("24h total tokens = %d, want %d", got, want)
	}
	if got, want := snapshot.Windows.Last7d.TotalTokens, int64(310); got != want {
		t.Fatalf("7d total tokens = %d, want %d", got, want)
	}

	alpha := snapshot.Provider("alpha")
	if got, want := alpha.TotalTokens, int64(250); got != want {
		t.Fatalf("alpha total tokens = %d, want %d", got, want)
	}
	if got, want := alpha.Last24h.TotalTokens, int64(250); got != want {
		t.Fatalf("alpha 24h total tokens = %d, want %d", got, want)
	}
	if got, want := alpha.Last7d.TotalTokens, int64(250); got != want {
		t.Fatalf("alpha 7d total tokens = %d, want %d", got, want)
	}

	beta := snapshot.Provider("beta")
	if got, want := beta.Last24h.TotalTokens, int64(0); got != want {
		t.Fatalf("beta 24h total tokens = %d, want %d", got, want)
	}
	if got, want := beta.Last7d.TotalTokens, int64(60); got != want {
		t.Fatalf("beta 7d total tokens = %d, want %d", got, want)
	}

	gpt54 := snapshot.Model("gpt-5.4")
	if got, want := gpt54.TotalTokens, int64(250); got != want {
		t.Fatalf("gpt-5.4 total tokens = %d, want %d", got, want)
	}
	gpt41 := snapshot.Model("gpt-4.1")
	if got, want := gpt41.Last7d.TotalTokens, int64(60); got != want {
		t.Fatalf("gpt-4.1 7d total tokens = %d, want %d", got, want)
	}
}

func TestStoreRecordsFailuresAndPrunesExpiredBuckets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	store.RecordFailure("alpha", "gpt-5.4", now)
	store.RecordFailure("alpha", "gpt-5.4", now.Add(-8*24*time.Hour))
	store.RecordSuccess("alpha", Usage{
		RequestCount: 1,
		InputTokens:  10,
		OutputTokens: 5,
		TotalTokens:  15,
	}, "gpt-5.4", now.Add(-8*24*time.Hour))

	snapshot := store.Snapshot(now)
	if got, want := snapshot.Overview.TotalFailures, int64(2); got != want {
		t.Fatalf("total failures = %d, want %d", got, want)
	}
	if got, want := snapshot.Windows.Last7d.FailureCount, int64(1); got != want {
		t.Fatalf("7d failures = %d, want %d", got, want)
	}
	if got, want := snapshot.Windows.Last7d.TotalTokens, int64(0); got != want {
		t.Fatalf("7d total tokens = %d, want %d", got, want)
	}

	reloaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot = reloaded.Snapshot(now)
	if got, want := len(snapshot.Series.Hourly24h), 24; got != want {
		t.Fatalf("hourly_24h points = %d, want %d", got, want)
	}
	if got, want := len(snapshot.Series.Daily7d), 7; got != want {
		t.Fatalf("daily_7d points = %d, want %d", got, want)
	}
	if got, want := snapshot.Model("gpt-5.4").FailureCount, int64(2); got != want {
		t.Fatalf("model failures = %d, want %d", got, want)
	}
}

func TestStoreRecordsStreamUsageIssueCounters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	store.RecordStreamUsageIssue("alpha", "gpt-5.4", StreamUsageIssueOmitted, now)
	store.RecordStreamUsageIssue("alpha", "gpt-5.4", StreamUsageIssueParseError, now)
	store.RecordStreamUsageIssue("beta", "gpt-4.1", StreamUsageIssueInterrupted, now)

	snapshot := store.Snapshot(now)
	if got, want := snapshot.Overview.StreamUsageMissingCount, int64(3); got != want {
		t.Fatalf("overview missing stream usage = %d, want %d", got, want)
	}
	if got, want := snapshot.Overview.StreamUsageOmittedCount, int64(1); got != want {
		t.Fatalf("overview omitted stream usage = %d, want %d", got, want)
	}
	if got, want := snapshot.Overview.StreamUsageParseErrorCount, int64(1); got != want {
		t.Fatalf("overview parse error stream usage = %d, want %d", got, want)
	}
	if got, want := snapshot.Overview.StreamUsageInterruptedCount, int64(1); got != want {
		t.Fatalf("overview interrupted stream usage = %d, want %d", got, want)
	}

	alpha := snapshot.Provider("alpha")
	if got, want := alpha.StreamUsageMissingCount, int64(2); got != want {
		t.Fatalf("alpha missing stream usage = %d, want %d", got, want)
	}
	if got, want := alpha.StreamUsageOmittedCount, int64(1); got != want {
		t.Fatalf("alpha omitted stream usage = %d, want %d", got, want)
	}
	if got, want := alpha.StreamUsageParseErrorCount, int64(1); got != want {
		t.Fatalf("alpha parse error stream usage = %d, want %d", got, want)
	}

	model := snapshot.Model("gpt-5.4")
	if got, want := model.StreamUsageMissingCount, int64(2); got != want {
		t.Fatalf("model missing stream usage = %d, want %d", got, want)
	}
	if got, want := model.StreamUsageOmittedCount, int64(1); got != want {
		t.Fatalf("model omitted stream usage = %d, want %d", got, want)
	}
	if got, want := model.StreamUsageParseErrorCount, int64(1); got != want {
		t.Fatalf("model parse error stream usage = %d, want %d", got, want)
	}
}
