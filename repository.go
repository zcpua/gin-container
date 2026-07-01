package main

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func listComposers(db *gorm.DB) ([]Composer, error) {
	var rows []Composer
	err := db.Order("birth_year asc").Find(&rows).Error
	return rows, err
}

func findComposerBySlug(db *gorm.DB, slug string) (*Composer, error) {
	var row Composer
	err := db.Where("slug = ?", slug).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func listWorks(db *gorm.DB) ([]Work, error) {
	var rows []Work
	err := db.Order("year is null asc").Order("year asc").Find(&rows).Error
	return rows, err
}

func findWorkBySlug(db *gorm.DB, slug string) (*Work, error) {
	var row Work
	err := db.Where("slug = ?", slug).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func listPerformances(db *gorm.DB) ([]Performance, error) {
	var rows []Performance
	err := db.Order("starts_at asc").Find(&rows).Error
	return rows, err
}

func listBannerPerformances(db *gorm.DB, openid string) ([]Performance, error) {
	var rows []Performance
	err := db.Table("performances").
		Joins("INNER JOIN favorites f ON f.performance_id = performances.id").
		Where("f.openid = ?", openid).
		Order("f.created_at desc").
		Find(&rows).Error
	return rows, err
}

func findPerformanceByID(db *gorm.DB, id string) (*Performance, error) {
	var row Performance
	err := db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func listArticles(db *gorm.DB) ([]Article, error) {
	var rows []Article
	err := db.Order("published_at desc").Find(&rows).Error
	return rows, err
}

func findArticleBySlug(db *gorm.DB, slug string) (*Article, error) {
	var row Article
	err := db.Where("slug = ?", slug).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func findUser(db *gorm.DB, openid string) (*User, error) {
	var row User
	err := db.Where("openid = ?", openid).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Insert the user row if absent, leaving an existing row untouched.
func ensureUser(db *gorm.DB, openid string, unionid *string) error {
	return db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&User{Openid: openid, Unionid: unionid}).Error
}

// Idempotent upsert keeping the latest nickname/avatar.
func upsertUser(db *gorm.DB, u User) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "openid"}},
		DoUpdates: clause.AssignmentColumns([]string{"unionid", "nickname", "avatar_url", "avatar_file_id", "updated_at"}),
	}).Create(&u).Error
}

type collectionKind string

const (
	kindFavorites collectionKind = "favorites"
	kindTickets   collectionKind = "tickets"
)

func (k collectionKind) table() string { return string(k) }

func listCollectionIDs(db *gorm.DB, openid string, kind collectionKind) ([]string, error) {
	var ids []string
	err := db.Table(kind.table()).
		Where("openid = ?", openid).
		Order("created_at desc").
		Pluck("performance_id", &ids).Error
	return ids, err
}

func listCollectionPerformances(db *gorm.DB, openid string, kind collectionKind) ([]Performance, error) {
	var rows []Performance
	err := db.Table("performances").
		Joins("INNER JOIN "+kind.table()+" t ON t.performance_id = performances.id").
		Where("t.openid = ?", openid).
		Order("t.created_at desc").
		Find(&rows).Error
	return rows, err
}

func addCollection(db *gorm.DB, openid, performanceID string, kind collectionKind) error {
	row := map[string]any{"openid": openid, "performance_id": performanceID}
	return db.Table(kind.table()).Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error
}

func removeCollection(db *gorm.DB, openid, performanceID string, kind collectionKind) error {
	return db.Table(kind.table()).
		Where("openid = ? AND performance_id = ?", openid, performanceID).
		Delete(nil).Error
}

func pingDb(db *gorm.DB) (int64, error) {
	var n int64
	err := db.Table("composers").Count(&n).Error
	return n, err
}

// --- Notification credits + sale-state transitions ---

func listPendingOnSaleTransitions(db *gorm.DB, limit int) ([]SaleStateTransition, error) {
	var rows []SaleStateTransition
	err := db.
		Where("to_state = ? AND notified_at IS NULL", "on_sale").
		Order("detected_at asc").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// Active credits for a given performance + kind — unconsumed, un-failed.
func findActiveCredits(db *gorm.DB, performanceID, kind string) ([]NotificationCredit, error) {
	var rows []NotificationCredit
	err := db.
		Where("performance_id = ? AND kind = ? AND consumed_at IS NULL AND failed_at IS NULL", performanceID, kind).
		Find(&rows).Error
	return rows, err
}

func markCreditConsumed(db *gorm.DB, openid, performanceID, kind string) error {
	now := time.Now()
	return db.Model(&NotificationCredit{}).
		Where("openid = ? AND performance_id = ? AND kind = ?", openid, performanceID, kind).
		Update("consumed_at", now).Error
}

// Non-atomic-relative-to-select but the notifier holds an in-memory Attempts
// count from the row it just read, so it does not race with itself.
func bumpCreditAttempts(db *gorm.DB, openid, performanceID, kind string) error {
	return db.Exec(
		`UPDATE notification_credits SET attempts = attempts + 1
		 WHERE openid = ? AND performance_id = ? AND kind = ?`,
		openid, performanceID, kind,
	).Error
}

func markCreditFailed(db *gorm.DB, openid, performanceID, kind string) error {
	now := time.Now()
	return db.Model(&NotificationCredit{}).
		Where("openid = ? AND performance_id = ? AND kind = ?", openid, performanceID, kind).
		Update("failed_at", now).Error
}

func markTransitionNotified(db *gorm.DB, id int64) error {
	now := time.Now()
	return db.Model(&SaleStateTransition{}).
		Where("id = ?", id).
		Update("notified_at", now).Error
}

// Read the list of active credit performance IDs for a given user + kind.
// Used by /me/notification-credits/ids to hydrate the mini-program's cache.
func listNotificationCreditIDs(db *gorm.DB, openid, kind string) ([]string, error) {
	var ids []string
	err := db.Table("notification_credits").
		Where("openid = ? AND kind = ? AND consumed_at IS NULL AND failed_at IS NULL", openid, kind).
		Pluck("performance_id", &ids).Error
	return ids, err
}

// Upserts a credit so a re-tap after a prior consumption re-arms the row.
func upsertNotificationCredit(db *gorm.DB, openid, performanceID, kind string) error {
	return db.Exec(
		`INSERT INTO notification_credits (openid, performance_id, kind)
		 VALUES (?, ?, ?)
		 ON CONFLICT (openid, performance_id, kind) DO UPDATE
		   SET granted_at = now(), consumed_at = NULL, attempts = 0, failed_at = NULL`,
		openid, performanceID, kind,
	).Error
}

func removeNotificationCredit(db *gorm.DB, openid, performanceID, kind string) error {
	return db.
		Where("openid = ? AND performance_id = ? AND kind = ?", openid, performanceID, kind).
		Delete(&NotificationCredit{}).Error
}
