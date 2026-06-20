package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// wechatAuth reads the WeChat cloud-hosting gateway headers and stores the
// resolved openid/unionid in the gin context. The gateway injects X-WX-OPENID
// (signed) and X-WX-UNIONID. Set WECHAT_DEV_OPENID to bypass during local dev.
func wechatAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		openid := firstNonEmpty(
			c.GetHeader("X-WX-OPENID"),
			c.GetHeader("X-WX-FROM-OPENID"),
			os.Getenv("WECHAT_DEV_OPENID"),
		)
		if openid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing wechat identity",
			})
			return
		}
		c.Set("openid", openid)
		if unionid := c.GetHeader("X-WX-UNIONID"); unionid != "" {
			c.Set("unionid", unionid)
		}
		c.Next()
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func ctxOpenid(c *gin.Context) string {
	v, _ := c.Get("openid")
	s, _ := v.(string)
	return s
}

func ctxUnionid(c *gin.Context) *string {
	v, ok := c.Get("unionid")
	if !ok {
		return nil
	}
	s, _ := v.(string)
	if s == "" {
		return nil
	}
	return &s
}
