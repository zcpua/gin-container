package main

import (
	"time"

	"gorm.io/datatypes"
)

type Composer struct {
	ID                 string         `gorm:"primaryKey" json:"id"`
	Slug               string         `json:"slug"`
	Name               string         `json:"name"`
	NameCn             string         `json:"nameCn"`
	BirthYear          int            `json:"birthYear"`
	DeathYear          *int           `json:"deathYear"`
	Country            string         `json:"country"`
	Period             string         `json:"period"`
	PortraitUrl        string         `json:"portraitUrl"`
	ShortBio           string         `json:"shortBio"`
	Bio                string         `json:"bio"`
	StyleTags          datatypes.JSON `json:"styleTags"`
	Timeline           datatypes.JSON `json:"timeline"`
	StarterWorkIds     datatypes.JSON `json:"starterWorkIds"`
	RelatedComposerIds datatypes.JSON `json:"relatedComposerIds"`
	Featured           bool           `json:"featured"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

func (Composer) TableName() string { return "composers" }

type Work struct {
	ID             string         `gorm:"primaryKey" json:"id"`
	Slug           string         `json:"slug"`
	ComposerID     string         `gorm:"column:composer_id" json:"composerId"`
	Title          string         `json:"title"`
	TitleCn        string         `json:"titleCn"`
	Year           *int           `json:"year"`
	Genre          string         `json:"genre"`
	Period         string         `json:"period"`
	Description    string         `json:"description"`
	Movements      datatypes.JSON `json:"movements"`
	ListeningLinks datatypes.JSON `json:"listeningLinks"`
	Featured       bool           `json:"featured"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (Work) TableName() string { return "works" }

type Performance struct {
	ID             string          `gorm:"primaryKey" json:"id"`
	Title          string          `json:"title"`
	City           string          `json:"city"`
	Venue          string          `json:"venue"`
	StartsAt       time.Time       `json:"startsAt"`
	Artists        datatypes.JSON  `json:"artists"`
	Program        datatypes.JSON  `json:"program"`
	TicketUrl      *string         `json:"ticketUrl"`
	SourceUrl      string          `json:"sourceUrl"`
	SourceName     string          `json:"sourceName"`
	ImageUrl       *string         `json:"imageUrl"`
	PriceLabel     *string         `json:"priceLabel"`
	SaleStatus     *string         `json:"saleStatus"`
	Address        *string         `json:"address"`
	Intro          *string         `json:"intro"`
	IsClassical    *bool           `json:"isClassical"`
	SourceID       *string         `gorm:"column:source_id" json:"sourceId"`
	SourceMetadata *datatypes.JSON `json:"sourceMetadata"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

func (Performance) TableName() string { return "performances" }

type Article struct {
	ID                 string         `gorm:"primaryKey" json:"id"`
	Slug               string         `json:"slug"`
	Title              string         `json:"title"`
	Excerpt            string         `json:"excerpt"`
	CoverUrl           string         `json:"coverUrl"`
	Category           string         `json:"category"`
	PublishedAt        time.Time      `json:"publishedAt"`
	Content            string         `json:"content"`
	RelatedComposerIds datatypes.JSON `json:"relatedComposerIds"`
	RelatedWorkIds     datatypes.JSON `json:"relatedWorkIds"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

func (Article) TableName() string { return "articles" }

type User struct {
	Openid       string    `gorm:"primaryKey" json:"openid"`
	Unionid      *string   `json:"unionid"`
	Nickname     *string   `json:"nickname"`
	AvatarUrl    *string   `json:"avatarUrl"`
	AvatarFileID *string   `gorm:"column:avatar_file_id" json:"avatarFileId"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (User) TableName() string { return "users" }

type Favorite struct {
	Openid        string    `gorm:"primaryKey" json:"openid"`
	PerformanceID string    `gorm:"primaryKey;column:performance_id" json:"performanceId"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (Favorite) TableName() string { return "favorites" }

type Ticket struct {
	Openid        string    `gorm:"primaryKey" json:"openid"`
	PerformanceID string    `gorm:"primaryKey;column:performance_id" json:"performanceId"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (Ticket) TableName() string { return "tickets" }
