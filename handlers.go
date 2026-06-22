package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type handlers struct {
	db     *gorm.DB
	upload avatarUploader
}

func (h *handlers) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "dbType": "postgres"})
}

func (h *handlers) count(c *gin.Context) {
	n, err := pingDb(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "dbType": "postgres", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "dbType": "postgres", "value": n})
}

func (h *handlers) listComposers(c *gin.Context) {
	rows, err := listComposers(h.db)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *handlers) getComposer(c *gin.Context) {
	row, err := findComposerBySlug(h.db, c.Param("slug"))
	if abortOnErr(c, err) {
		return
	}
	if row == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *handlers) listWorks(c *gin.Context) {
	rows, err := listWorks(h.db)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *handlers) getWork(c *gin.Context) {
	row, err := findWorkBySlug(h.db, c.Param("slug"))
	if abortOnErr(c, err) {
		return
	}
	if row == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *handlers) listPerformances(c *gin.Context) {
	rows, err := listPerformances(h.db)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *handlers) listBannerPerformances(c *gin.Context) {
	rows, err := listBannerPerformances(h.db, bannerOpenid)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *handlers) getPerformance(c *gin.Context) {
	row, err := findPerformanceByID(h.db, c.Param("id"))
	if abortOnErr(c, err) {
		return
	}
	if row == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *handlers) listArticles(c *gin.Context) {
	rows, err := listArticles(h.db)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *handlers) getArticle(c *gin.Context) {
	row, err := findArticleBySlug(h.db, c.Param("slug"))
	if abortOnErr(c, err) {
		return
	}
	if row == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *handlers) login(c *gin.Context) {
	openid := ctxOpenid(c)
	unionid := ctxUnionid(c)
	if err := ensureUser(h.db, openid, unionid); abortOnErr(c, err) {
		return
	}
	user, err := findUser(h.db, openid)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"openid": openid, "unionid": unionid, "user": user})
}

func (h *handlers) getProfile(c *gin.Context) {
	openid := ctxOpenid(c)
	user, err := findUser(h.db, openid)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"openid": openid, "user": user})
}

type patchProfileBody struct {
	Nickname     *string `json:"nickname"`
	AvatarBase64 *string `json:"avatarBase64"`
}

func (h *handlers) patchProfile(c *gin.Context) {
	openid := ctxOpenid(c)
	unionid := ctxUnionid(c)

	var body patchProfileBody
	_ = c.ShouldBindJSON(&body)

	existing, err := findUser(h.db, openid)
	if abortOnErr(c, err) {
		return
	}

	next := User{Openid: openid, Unionid: unionid}
	if existing != nil {
		next.Nickname = existing.Nickname
		next.AvatarUrl = existing.AvatarUrl
		next.AvatarFileID = existing.AvatarFileID
	}

	if body.Nickname != nil {
		nickname := truncate(*body.Nickname, 32)
		next.Nickname = &nickname
	}

	if body.AvatarBase64 != nil && *body.AvatarBase64 != "" {
		decoded := decodeDataURL(*body.AvatarBase64)
		if decoded == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar"})
			return
		}
		if h.upload == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "avatar upload not configured"})
			return
		}
		key := "avatars/" + openid + "." + extForContentType(decoded.ContentType)
		res, err := h.upload(c.Request.Context(), decoded.Bytes, decoded.ContentType, key)
		if abortOnErr(c, err) {
			return
		}
		next.AvatarUrl = &res.URL
		next.AvatarFileID = res.FileID
	}

	if err := upsertUser(h.db, next); abortOnErr(c, err) {
		return
	}
	user, err := findUser(h.db, openid)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"openid": openid, "user": user})
}

func (h *handlers) favoriteIDs(c *gin.Context) { h.collectionIDs(c, kindFavorites) }
func (h *handlers) ticketIDs(c *gin.Context)   { h.collectionIDs(c, kindTickets) }

func (h *handlers) collectionIDs(c *gin.Context, kind collectionKind) {
	ids, err := listCollectionIDs(h.db, ctxOpenid(c), kind)
	if abortOnErr(c, err) {
		return
	}
	if ids == nil {
		ids = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"ids": ids})
}

func (h *handlers) favorites(c *gin.Context) { h.collection(c, kindFavorites) }
func (h *handlers) tickets(c *gin.Context)   { h.collection(c, kindTickets) }

func (h *handlers) collection(c *gin.Context, kind collectionKind) {
	rows, err := listCollectionPerformances(h.db, ctxOpenid(c), kind)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *handlers) addFavorite(c *gin.Context) { h.addToCollection(c, kindFavorites) }
func (h *handlers) addTicket(c *gin.Context)   { h.addToCollection(c, kindTickets) }

func (h *handlers) addToCollection(c *gin.Context, kind collectionKind) {
	performanceID := c.Param("performanceId")
	perf, err := findPerformanceByID(h.db, performanceID)
	if abortOnErr(c, err) {
		return
	}
	if perf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "performance not found"})
		return
	}
	openid := ctxOpenid(c)
	if err := ensureUser(h.db, openid, ctxUnionid(c)); abortOnErr(c, err) {
		return
	}
	if err := addCollection(h.db, openid, performanceID, kind); abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handlers) removeFavorite(c *gin.Context) { h.removeFromCollection(c, kindFavorites) }
func (h *handlers) removeTicket(c *gin.Context)   { h.removeFromCollection(c, kindTickets) }

func (h *handlers) removeFromCollection(c *gin.Context, kind collectionKind) {
	err := removeCollection(h.db, ctxOpenid(c), c.Param("performanceId"), kind)
	if abortOnErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func abortOnErr(c *gin.Context, err error) bool {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}
	return false
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		r = r[:max]
	}
	return string(r)
}
