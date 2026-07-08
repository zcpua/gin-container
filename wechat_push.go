package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Typed sentinels the notifier switches on. Errors from the raw HTTP layer
// (network failures, JSON decode failures) come back as generic errors and
// count against the per-credit attempt cap.
var (
	errPushRefused      = errors.New("wechat: user refused or not subscribed")
	errPushRateLimited  = errors.New("wechat: rate limited")
	errPushFatal        = errors.New("wechat: fatal (template/data)")
	errPushTokenInvalid = errors.New("wechat: token invalid")
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type wechatPushConfig struct {
	AppID            string
	AppSecret        string
	OnSaleTmplID     string
	MiniprogramState string // "developer" | "trial" | "formal"
}

type wechatPusher interface {
	send(ctx context.Context, openid string, perf *Performance) error
}

type pusher struct {
	cfg   wechatPushConfig
	http  httpDoer
	now   func() time.Time
	mu    sync.Mutex
	token string
	exp   time.Time
}

func newWechatPusher(cfg wechatPushConfig, h httpDoer, now func() time.Time) wechatPusher {
	if now == nil {
		now = time.Now
	}
	if h == nil {
		h = &http.Client{Timeout: 10 * time.Second}
	}
	return &pusher{cfg: cfg, http: h, now: now}
}

// Refreshes 5 minutes before WeChat's stated expiry to stay clear of the
// grace window; 40001 responses on send() invalidate the cached token so
// the next send() forces a fresh fetch (see below).
func (p *pusher) accessToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exp.Sub(p.now()) > time.Minute {
		return p.token, nil
	}
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		p.cfg.AppID, p.cfg.AppSecret,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("token fetch failed: errcode=%d errmsg=%s", body.Errcode, body.Errmsg)
	}
	p.token = body.AccessToken
	p.exp = p.now().Add(time.Duration(body.ExpiresIn-300) * time.Second)
	return p.token, nil
}

// send posts a single 订阅消息. Returns nil on success, a typed sentinel for
// the errcodes the notifier knows how to handle (refused/rate-limited/fatal/
// token-invalid), or a generic error for everything else.
func (p *pusher) send(ctx context.Context, openid string, perf *Performance) error {
	token, err := p.accessToken(ctx)
	if err != nil {
		return err
	}
	body := map[string]any{
		"touser":      openid,
		"template_id": p.cfg.OnSaleTmplID,
		"page":        fmt.Sprintf("pages/detail/index?id=%s", perf.ID),
		"data": map[string]any{
			// The exact field IDs (thing1/time2/thing3) depend on the approved
			// 订阅消息 template — swap them out if 微信公众平台 审批 assigns
			// different keys.
			"thing1": map[string]string{"value": truncRunes(perf.Title, 20)},
			"time2":  map[string]string{"value": perf.StartsAt.In(cstZone).Format("2006年01月02日 15:04")},
			"thing3": map[string]string{"value": truncRunes(perf.Venue, 20)},
		},
	}
	if p.cfg.MiniprogramState != "" {
		body["miniprogram_state"] = p.cfg.MiniprogramState
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	u := "https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=" + token
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var r struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("decode push response: %w; body=%s", err, string(raw))
	}
	switch r.Errcode {
	case 0:
		return nil
	case 40001, 40014, 42001:
		// Force a token refresh on the next call.
		p.mu.Lock()
		p.exp = time.Time{}
		p.mu.Unlock()
		return errPushTokenInvalid
	case 43101:
		return errPushRefused
	case 45009, 45040:
		return errPushRateLimited
	case 47003:
		return errPushFatal
	default:
		return fmt.Errorf("wechat: errcode=%d errmsg=%s", r.Errcode, r.Errmsg)
	}
}

var cstZone = time.FixedZone("CST", 8*3600)

func truncRunes(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		r = r[:max]
	}
	return string(r)
}
