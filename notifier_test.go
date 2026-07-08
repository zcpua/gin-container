package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// --- Test doubles ---

type mockPusher struct {
	mu       sync.Mutex
	calls    []string
	results  []error
	defaultE error
}

func (m *mockPusher) send(_ context.Context, openid string, _ *Performance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, openid)
	if len(m.results) > 0 {
		r := m.results[0]
		m.results = m.results[1:]
		return r
	}
	return m.defaultE
}

type fakeRepo struct {
	pending  []SaleStateTransition
	creds    map[string][]NotificationCredit
	consumed []string
	failed   []string
	notified []int64
	attempts map[string]int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{creds: map[string][]NotificationCredit{}, attempts: map[string]int{}}
}

func (f *fakeRepo) listPending(int) ([]SaleStateTransition, error) { return f.pending, nil }
func (f *fakeRepo) findCredits(perfID, kind string) ([]NotificationCredit, error) {
	return f.creds[perfID], nil
}
func (f *fakeRepo) markCreditConsumed(openid, _, _ string) error {
	f.consumed = append(f.consumed, openid)
	return nil
}
func (f *fakeRepo) bumpCreditAttempts(openid, perfID, kind string) error {
	f.attempts[openid+"|"+perfID+"|"+kind]++
	return nil
}
func (f *fakeRepo) markCreditFailed(openid, _, _ string) error {
	f.failed = append(f.failed, openid)
	return nil
}
func (f *fakeRepo) markTransitionNotified(id int64) error {
	f.notified = append(f.notified, id)
	return nil
}

func stubPerf(_ string) (*Performance, error) {
	return &Performance{ID: "p1", Title: "T", Venue: "V", StartsAt: time.Now()}, nil
}

// --- Tests ---

func TestNotifierHappyPath(t *testing.T) {
	repo := newFakeRepo()
	repo.pending = []SaleStateTransition{{ID: 1, PerformanceID: "p1", FromState: "pre_sale", ToState: "on_sale"}}
	repo.creds["p1"] = []NotificationCredit{
		{Openid: "userA", PerformanceID: "p1", Kind: "on_sale"},
		{Openid: "userB", PerformanceID: "p1", Kind: "on_sale"},
	}
	push := &mockPusher{}
	n := &notifier{push: push, repo: repo, perfLookup: stubPerf, attemptCap: 3, batchSize: 10}
	if err := n.tickOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := len(push.calls); got != 2 {
		t.Fatalf("want 2 pushes, got %d", got)
	}
	if got := len(repo.consumed); got != 2 {
		t.Fatalf("want 2 credits consumed, got %d", got)
	}
	if len(repo.notified) != 1 || repo.notified[0] != 1 {
		t.Fatalf("want transition 1 marked notified, got %+v", repo.notified)
	}
}

func TestNotifierRateLimitedLeavesPending(t *testing.T) {
	repo := newFakeRepo()
	repo.pending = []SaleStateTransition{{ID: 1, PerformanceID: "p1", FromState: "pre_sale", ToState: "on_sale"}}
	repo.creds["p1"] = []NotificationCredit{{Openid: "u", PerformanceID: "p1", Kind: "on_sale"}}
	push := &mockPusher{defaultE: errPushRateLimited}
	n := &notifier{push: push, repo: repo, perfLookup: stubPerf, attemptCap: 3, batchSize: 10}
	if err := n.tickOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.consumed) != 0 {
		t.Fatalf("rate-limited credit should not be consumed")
	}
	if len(repo.notified) != 0 {
		t.Fatalf("rate-limited transition should not be marked notified")
	}
	if repo.attempts["u|p1|on_sale"] != 1 {
		t.Fatalf("expected attempts=1, got %d", repo.attempts["u|p1|on_sale"])
	}
}

func TestNotifierRefusedConsumesCredit(t *testing.T) {
	repo := newFakeRepo()
	repo.pending = []SaleStateTransition{{ID: 1, PerformanceID: "p1", FromState: "pre_sale", ToState: "on_sale"}}
	repo.creds["p1"] = []NotificationCredit{{Openid: "u", PerformanceID: "p1", Kind: "on_sale"}}
	push := &mockPusher{defaultE: errPushRefused}
	n := &notifier{push: push, repo: repo, perfLookup: stubPerf, attemptCap: 3, batchSize: 10}
	if err := n.tickOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.consumed) != 1 {
		t.Fatalf("refused credit should be consumed (no retry); got %d", len(repo.consumed))
	}
	if len(repo.notified) != 1 {
		t.Fatalf("transition should advance even when user refused")
	}
}

func TestNotifierFatalConsumesCredit(t *testing.T) {
	repo := newFakeRepo()
	repo.pending = []SaleStateTransition{{ID: 1, PerformanceID: "p1", FromState: "pre_sale", ToState: "on_sale"}}
	repo.creds["p1"] = []NotificationCredit{{Openid: "u", PerformanceID: "p1", Kind: "on_sale"}}
	push := &mockPusher{defaultE: errPushFatal}
	n := &notifier{push: push, repo: repo, perfLookup: stubPerf, attemptCap: 3, batchSize: 10}
	if err := n.tickOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.consumed) != 1 || len(repo.notified) != 1 {
		t.Fatalf("fatal errcode should consume credit + advance transition; consumed=%d notified=%d", len(repo.consumed), len(repo.notified))
	}
}

func TestNotifierAttemptCapMarksFailed(t *testing.T) {
	repo := newFakeRepo()
	repo.pending = []SaleStateTransition{{ID: 1, PerformanceID: "p1", FromState: "pre_sale", ToState: "on_sale"}}
	// Attempts=2 already, so the third failure this tick hits the cap.
	repo.creds["p1"] = []NotificationCredit{{Openid: "u", PerformanceID: "p1", Kind: "on_sale", Attempts: 2}}
	push := &mockPusher{defaultE: errors.New("network fail")}
	n := &notifier{push: push, repo: repo, perfLookup: stubPerf, attemptCap: 3, batchSize: 10}
	if err := n.tickOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.failed) != 1 {
		t.Fatalf("expected 1 credit failed, got %d", len(repo.failed))
	}
	if len(repo.notified) != 1 {
		t.Fatalf("transition should advance once all credits either consumed or failed; got %d", len(repo.notified))
	}
}

func TestNotifierSkipsMissingPerformance(t *testing.T) {
	repo := newFakeRepo()
	repo.pending = []SaleStateTransition{{ID: 1, PerformanceID: "gone", ToState: "on_sale"}}
	push := &mockPusher{}
	n := &notifier{
		push:       push,
		repo:       repo,
		perfLookup: func(string) (*Performance, error) { return nil, nil },
		attemptCap: 3,
		batchSize:  10,
	}
	if err := n.tickOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(push.calls) != 0 {
		t.Fatalf("no push when perf missing")
	}
	if len(repo.notified) != 0 {
		t.Fatalf("transition should stay pending when perf missing; got %d", len(repo.notified))
	}
}
