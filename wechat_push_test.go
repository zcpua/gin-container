package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fakeResp struct {
	body   string
	status int
}

type fakeDoer struct {
	responses []fakeResp
	calls     []*http.Request
}

func (f *fakeDoer) Do(r *http.Request) (*http.Response, error) {
	f.calls = append(f.calls, r)
	if len(f.responses) == 0 {
		return nil, errors.New("fakeDoer: no responses queued")
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	status := resp.status
	if status == 0 {
		status = 200
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Header:     make(http.Header),
	}, nil
}

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestTokenCacheReusesWithinExpiry(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	d := &fakeDoer{responses: []fakeResp{
		{body: `{"access_token":"tok1","expires_in":7200}`},
		{body: `{"errcode":0}`},
		{body: `{"errcode":0}`},
	}}
	p := newWechatPusher(wechatPushConfig{AppID: "a", AppSecret: "s", OnSaleTmplID: "tpl"}, d, fixedClock(now))
	perf := &Performance{ID: "p1", Title: "T", Venue: "V", StartsAt: now}
	if err := p.send(context.Background(), "user1", perf); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := p.send(context.Background(), "user2", perf); err != nil {
		t.Fatalf("second send: %v", err)
	}
	if got := len(d.calls); got != 3 {
		t.Fatalf("expected 3 http calls (1 token + 2 send), got %d", got)
	}
}

func TestTokenCacheRefreshesAfterExpiry(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	d := &fakeDoer{responses: []fakeResp{
		{body: `{"access_token":"tok1","expires_in":7200}`},
		{body: `{"errcode":0}`},
		{body: `{"access_token":"tok2","expires_in":7200}`},
		{body: `{"errcode":0}`},
	}}
	clock := now
	p := newWechatPusher(wechatPushConfig{AppID: "a", AppSecret: "s", OnSaleTmplID: "tpl"}, d, func() time.Time { return clock })
	perf := &Performance{ID: "p1", StartsAt: now}
	if err := p.send(context.Background(), "u", perf); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(3 * time.Hour)
	if err := p.send(context.Background(), "u", perf); err != nil {
		t.Fatal(err)
	}
	if got := len(d.calls); got != 4 {
		t.Fatalf("expected 4 calls (2 token + 2 send), got %d", got)
	}
}

func TestSendMapsErrcodes(t *testing.T) {
	cases := []struct {
		body     string
		expected error
	}{
		{`{"errcode":43101}`, errPushRefused},
		{`{"errcode":45009}`, errPushRateLimited},
		{`{"errcode":47003}`, errPushFatal},
		{`{"errcode":40001}`, errPushTokenInvalid},
	}
	for _, c := range cases {
		now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
		d := &fakeDoer{responses: []fakeResp{
			{body: `{"access_token":"t","expires_in":7200}`},
			{body: c.body},
		}}
		p := newWechatPusher(wechatPushConfig{AppID: "a", AppSecret: "s", OnSaleTmplID: "tpl"}, d, fixedClock(now))
		err := p.send(context.Background(), "u", &Performance{ID: "p", StartsAt: now})
		if !errors.Is(err, c.expected) {
			t.Errorf("body=%s expected %v got %v", c.body, c.expected, err)
		}
	}
}

func TestTokenInvalidForcesRefreshOnNextCall(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	d := &fakeDoer{responses: []fakeResp{
		{body: `{"access_token":"tok1","expires_in":7200}`},
		{body: `{"errcode":40001}`},
		// After the token-invalid response the next send() should trigger
		// a fresh token fetch, then the actual push.
		{body: `{"access_token":"tok2","expires_in":7200}`},
		{body: `{"errcode":0}`},
	}}
	p := newWechatPusher(wechatPushConfig{AppID: "a", AppSecret: "s", OnSaleTmplID: "tpl"}, d, fixedClock(now))
	perf := &Performance{ID: "p", StartsAt: now}
	if err := p.send(context.Background(), "u", perf); !errors.Is(err, errPushTokenInvalid) {
		t.Fatalf("first send should return token-invalid; got %v", err)
	}
	if err := p.send(context.Background(), "u", perf); err != nil {
		t.Fatalf("second send should succeed after token refresh; got %v", err)
	}
	if got := len(d.calls); got != 4 {
		t.Fatalf("expected 4 calls, got %d", got)
	}
}

func TestTruncRunes(t *testing.T) {
	if got := truncRunes("演出名称测试超出长度", 5); got != "演出名称测" {
		t.Fatalf("truncRunes rune count wrong: %q", got)
	}
	if got := truncRunes("short", 20); got != "short" {
		t.Fatalf("truncRunes should not truncate short input: %q", got)
	}
}
