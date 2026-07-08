package main

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

// notifierRepo abstracts the DB layer so the notifier can be unit-tested
// with an in-memory fake. Production wires this to gormRepo, which calls
// the plain helpers in repository.go.
type notifierRepo interface {
	listPending(limit int) ([]SaleStateTransition, error)
	findCredits(performanceID, kind string) ([]NotificationCredit, error)
	markCreditConsumed(openid, performanceID, kind string) error
	bumpCreditAttempts(openid, performanceID, kind string) error
	markCreditFailed(openid, performanceID, kind string) error
	markTransitionNotified(id int64) error
}

type notifier struct {
	repo       notifierRepo
	push       wechatPusher
	perfLookup func(id string) (*Performance, error)
	batchSize  int
	attemptCap int
	enabled    bool
	sendPause  time.Duration
}

// run is the long-lived loop. It exits immediately when NOTIFIER_ENABLED is
// off — the kill switch is the presence of the ticker itself, not an inner
// gate — so a caller only invokes run() when the feature is meant to fire.
func (n *notifier) run(ctx context.Context, every time.Duration) {
	if !n.enabled {
		log.Printf("notifier disabled by NOTIFIER_ENABLED=false")
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	if err := n.tickOnce(ctx); err != nil {
		log.Printf("notifier first tick: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := n.tickOnce(ctx); err != nil {
				log.Printf("notifier tick: %v", err)
			}
		}
	}
}

// tickOnce drains up to batchSize pending on_sale transitions. Split out
// from run() so tests can drive exactly one pass without wall-clock time.
func (n *notifier) tickOnce(ctx context.Context) error {
	trs, err := n.repo.listPending(n.batchSize)
	if err != nil {
		return err
	}
	for _, tr := range trs {
		perf, err := n.perfLookup(tr.PerformanceID)
		if err != nil || perf == nil {
			log.Printf("notifier: perf %s lookup err=%v perf=%v; skipping", tr.PerformanceID, err, perf)
			continue
		}
		creds, err := n.repo.findCredits(tr.PerformanceID, "on_sale")
		if err != nil {
			log.Printf("notifier: findCredits err=%v", err)
			continue
		}
		anyRetry := false
		for _, c := range creds {
			err := n.push.send(ctx, c.Openid, perf)
			switch {
			case err == nil:
				_ = n.repo.markCreditConsumed(c.Openid, c.PerformanceID, c.Kind)
			case errors.Is(err, errPushRefused), errors.Is(err, errPushFatal):
				// User refused or template is bust for this credit — advance.
				_ = n.repo.markCreditConsumed(c.Openid, c.PerformanceID, c.Kind)
			case errors.Is(err, errPushRateLimited):
				_ = n.repo.bumpCreditAttempts(c.Openid, c.PerformanceID, c.Kind)
				anyRetry = true
			default:
				// Generic error or token-invalid. Bump attempts; if we've hit
				// the cap, mark failed and let the transition proceed.
				_ = n.repo.bumpCreditAttempts(c.Openid, c.PerformanceID, c.Kind)
				if c.Attempts+1 >= n.attemptCap {
					_ = n.repo.markCreditFailed(c.Openid, c.PerformanceID, c.Kind)
				} else {
					anyRetry = true
				}
			}
			if n.sendPause > 0 {
				time.Sleep(n.sendPause)
			}
		}
		if !anyRetry {
			_ = n.repo.markTransitionNotified(tr.ID)
		}
	}
	return nil
}

// gormRepo is the production notifierRepo, backed by the plain helpers
// in repository.go.
type gormRepo struct{ db *gorm.DB }

func (r *gormRepo) listPending(limit int) ([]SaleStateTransition, error) {
	return listPendingOnSaleTransitions(r.db, limit)
}
func (r *gormRepo) findCredits(performanceID, kind string) ([]NotificationCredit, error) {
	return findActiveCredits(r.db, performanceID, kind)
}
func (r *gormRepo) markCreditConsumed(openid, performanceID, kind string) error {
	return markCreditConsumed(r.db, openid, performanceID, kind)
}
func (r *gormRepo) bumpCreditAttempts(openid, performanceID, kind string) error {
	return bumpCreditAttempts(r.db, openid, performanceID, kind)
}
func (r *gormRepo) markCreditFailed(openid, performanceID, kind string) error {
	return markCreditFailed(r.db, openid, performanceID, kind)
}
func (r *gormRepo) markTransitionNotified(id int64) error {
	return markTransitionNotified(r.db, id)
}
