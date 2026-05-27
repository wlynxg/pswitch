package pool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pswitch/internal/config"
)

func TestCandidatesSequentialSkipsUnhealthyProviders(t *testing.T) {
	p, err := New(config.Config{
		Mode:             "sequential",
		FailureThreshold: 1,
		Cooldown:         time.Second,
		Providers: []config.Provider{
			{Name: "a", BaseURL: "http://a", APIKey: "ka"},
			{Name: "b", BaseURL: "http://b", APIKey: "kb"},
			{Name: "c", BaseURL: "http://c", APIKey: "kc"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(100, 0)
	p.MarkFailure("a", now)
	got := names(p.Candidates(ModeSequential, now))
	want := []string{"b", "c"}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %v, want %v", got, want)
		}
	}
}

func TestCandidatesRoundRobinStartsAfterLastSuccess(t *testing.T) {
	p, err := New(config.Config{
		Mode:             "round_robin",
		FailureThreshold: 1,
		Cooldown:         time.Second,
		Providers: []config.Provider{
			{Name: "a", BaseURL: "http://a", APIKey: "ka"},
			{Name: "b", BaseURL: "http://b", APIKey: "kb"},
			{Name: "c", BaseURL: "http://c", APIKey: "kc"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(100, 0)
	p.MarkSuccess("b", now)
	got := names(p.Candidates(ModeRoundRobin, now))
	want := []string{"c", "a", "b"}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %v, want %v", got, want)
		}
	}
}

func TestCandidatesLeastFailuresPrefersProviderWithFewestFailures(t *testing.T) {
	p, err := New(config.Config{
		Mode:             "least_failures",
		FailureThreshold: 3,
		Cooldown:         time.Second,
		Providers: []config.Provider{
			{Name: "a", BaseURL: "http://a", APIKey: "ka"},
			{Name: "b", BaseURL: "http://b", APIKey: "kb"},
			{Name: "c", BaseURL: "http://c", APIKey: "kc"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(100, 0)
	p.MarkFailure("a", now)
	p.MarkFailure("a", now)
	p.MarkFailure("c", now)

	got := names(p.Candidates(Mode("least_failures"), now))
	want := []string{"b", "c", "a"}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %v, want %v", got, want)
		}
	}
}

func TestCandidatesForNamesFiltersToNamedProviders(t *testing.T) {
	p, err := New(config.Config{
		Mode:             "sequential",
		FailureThreshold: 1,
		Cooldown:         time.Second,
		Providers: []config.Provider{
			{Name: "a", BaseURL: "http://a", APIKey: "ka"},
			{Name: "b", BaseURL: "http://b", APIKey: "kb"},
			{Name: "c", BaseURL: "http://c", APIKey: "kc"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(100, 0)
	got := names(p.CandidatesForNames([]string{"a", "c"}, ModeSequential, now))
	want := []string{"a", "c"}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %v, want %v", got, want)
		}
	}
}

func TestCandidatesForNamesEmptyReturnsAll(t *testing.T) {
	p, err := New(config.Config{
		Mode:             "sequential",
		FailureThreshold: 1,
		Cooldown:         time.Second,
		Providers: []config.Provider{
			{Name: "a", BaseURL: "http://a", APIKey: "ka"},
			{Name: "b", BaseURL: "http://b", APIKey: "kb"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(100, 0)
	if got := len(p.CandidatesForNames(nil, ModeSequential, now)); got != 2 {
		t.Fatalf("candidates = %d, want 2", got)
	}
	if got := len(p.CandidatesForNames([]string{}, ModeSequential, now)); got != 2 {
		t.Fatalf("candidates = %d, want 2", got)
	}
}

func TestProbeDueRestoresProvider(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(config.Config{
		Mode:             "round_robin",
		FailureThreshold: 1,
		Cooldown:         0,
		Providers: []config.Provider{
			{Name: "a", BaseURL: upstream.URL, APIKey: "ka"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(100, 0)
	p.MarkFailure("a", now)
	if st, _ := p.Status("a"); st.Healthy {
		t.Fatal("provider should be unhealthy after failure")
	}

	events := p.ProbeDue(context.Background(), http.DefaultClient, now)
	if len(events) != 1 {
		t.Fatalf("events = %v, want 1 recovery event", events)
	}
	if got, want := events[0].Provider, "a"; got != want {
		t.Fatalf("event provider = %q, want %q", got, want)
	}

	st, ok := p.Status("a")
	if !ok {
		t.Fatal("provider status not found")
	}
	if !st.Healthy {
		t.Fatal("provider should become healthy after probe succeeds")
	}
}

func TestStatusTracksLastFailureAndSuccessDetails(t *testing.T) {
	p, err := New(config.Config{
		Mode:             "round_robin",
		FailureThreshold: 1,
		Cooldown:         time.Second,
		Providers: []config.Provider{
			{Name: "a", BaseURL: "http://a", APIKey: "ka"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	failAt := time.Unix(100, 0)
	successAt := failAt.Add(5 * time.Second)

	p.MarkFailureWithReason("a", failAt, "tls handshake failed")

	st, ok := p.Status("a")
	if !ok {
		t.Fatal("provider status not found")
	}
	if st.LastError != "tls handshake failed" {
		t.Fatalf("last error = %q, want %q", st.LastError, "tls handshake failed")
	}
	if !st.LastErrorAt.Equal(failAt) {
		t.Fatalf("last error at = %v, want %v", st.LastErrorAt, failAt)
	}
	if !st.LastSuccessAt.IsZero() {
		t.Fatalf("last success at = %v, want zero", st.LastSuccessAt)
	}

	p.MarkSuccess("a", successAt)

	st, ok = p.Status("a")
	if !ok {
		t.Fatal("provider status not found")
	}
	if !st.LastSuccessAt.Equal(successAt) {
		t.Fatalf("last success at = %v, want %v", st.LastSuccessAt, successAt)
	}
	if st.LastError != "tls handshake failed" {
		t.Fatalf("last error = %q, want %q", st.LastError, "tls handshake failed")
	}
}

func names(items []ProviderSnapshot) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Name)
	}
	return out
}
