package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sealock/core-storage/service"
	"golang.org/x/crypto/bcrypt"
)

// ShareMiddleware handles shared link access
// It verifies the share token, checks expiration, password, and view limits
func ShareMiddleware(shareService *service.ShareService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract share token from URL
		token := c.Param("token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "无效的分享链接"})
			return
		}

		// 2. Get share record from service
		share, err := shareService.GetShareByToken(context.Background(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "分享不存在或已过期"})
			return
		}

		// 3. Check expiration
		if share.ExpiredAt != nil {
			expTime, err := time.Parse(time.RFC3339, *share.ExpiredAt)
			if err == nil && time.Now().After(expTime) {
				c.AbortWithStatusJSON(http.StatusGone, gin.H{"error": "分享已过期"})
				return
			}
		}

		// 4. Check view limits
		if share.MaxViews != nil && *share.MaxViews > 0 && share.CurrentViews >= *share.MaxViews {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "分享已达到最大访问次数"})
			return
		}

		// 5. Check password if required
		if share.PasswordHash != nil {
			password := c.Query("password")
			if password == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "需要访问密码"})
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(*share.PasswordHash), []byte(password)); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的访问密码"})
				return
			}
		}

		// 6. Update view count
		if err := shareService.IncrementViewCount(c, token); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "无法更新访问计数"})
			return
		}

		// 7. Inject resource ID into context
		c.Set("resource_id", share.ResourceID)
		c.Set("share_token", token)

		c.Next()
	}
}