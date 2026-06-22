package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const bannerOpenid = "od6ryxbFSuApeg3K3fS5FSyasUf8"

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
	}

	// Avatar uploads. Prefer UpYun's S3-compatible API when S3_ACCESS_KEY is set,
	// otherwise fall back to tencent COS inside WeChat Cloud Run. Without either,
	// the uploader stays nil and avatar uploads return 501.
	var upload avatarUploader
	if ak := os.Getenv("S3_ACCESS_KEY"); ak != "" {
		upload = upyunS3Uploader(s3Config{
			Endpoint:     envOr("S3_ENDPOINT", "https://s3.api.upyun.com"),
			Region:       envOr("S3_REGION", "us-east-1"),
			Bucket:       os.Getenv("S3_BUCKET"),
			AccessKey:    ak,
			SecretKey:    os.Getenv("S3_SECRET_KEY"),
			PublicDomain: os.Getenv("S3_PUBLIC_DOMAIN"),
		})
	} else if bucket := os.Getenv("COS_BUCKET"); bucket != "" {
		upload = wxCosUploader(cosConfig{
			Bucket:       bucket,
			Region:       envOr("COS_REGION", "ap-shanghai"),
			CloudEnv:     os.Getenv("COS_CLOUD_ENV"),
			PublicDomain: os.Getenv("COS_PUBLIC_DOMAIN"),
		})
	}

	h := &handlers{db: db, upload: upload}

	r := gin.Default()
	r.Use(cors.Default())

	api := r.Group("/api/v2")
	{
		api.GET("/health", h.health)
		api.GET("/count", h.count)

		api.GET("/composers", h.listComposers)
		api.GET("/composers/:slug", h.getComposer)

		api.GET("/works", h.listWorks)
		api.GET("/works/:slug", h.getWork)

		api.GET("/banners", h.listBannerPerformances)
		api.GET("/performances", h.listPerformances)
		api.GET("/performances/:id", h.getPerformance)

		api.GET("/articles", h.listArticles)
		api.GET("/articles/:slug", h.getArticle)

		me := api.Group("/me", wechatAuth())
		{
			me.POST("/login", h.login)
			me.GET("/profile", h.getProfile)
			me.PATCH("/profile", h.patchProfile)

			me.GET("/favorites/ids", h.favoriteIDs)
			me.GET("/tickets/ids", h.ticketIDs)
			me.GET("/favorites", h.favorites)
			me.GET("/tickets", h.tickets)

			me.POST("/favorites/:performanceId", h.addFavorite)
			me.DELETE("/favorites/:performanceId", h.removeFavorite)
			me.POST("/tickets/:performanceId", h.addTicket)
			me.DELETE("/tickets/:performanceId", h.removeTicket)
		}
	}

	port := envOr("PORT", "8080")
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
