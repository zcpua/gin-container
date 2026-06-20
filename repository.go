package main

import (
	"errors"

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
