package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// 定义资料库角色常量
const (
	OwnerRole        = "owner"
	CollaboratorRole = "collaborator"
	GuestRole        = "guest"
)

// AuthMiddleware JWT鉴权中间件
// 实现资料库（Repo）级别的权限校验，支持Owner/Collaborator/Guest三种角色
// 针对敏感操作（删除库、修改成员）增加二级验证逻辑
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从请求头获取JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			return
		}

		// 2. 验证token格式
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌格式"})
			return
		}

		tokenString := tokenParts[1]

		// 3. 解析并验证JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("无效的签名算法")
			}
			// 返回密钥（应从配置获取）
			return []byte("your-secret-key"), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效或过期的令牌"})
			return
		}

		// 4. 提取用户信息
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌声明"})
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "令牌中缺少用户ID"})
			return
		}

		// 5. 获取当前请求的RepoID
		repoID := c.Param("repo_id")
		if repoID == "" {
			// 从路径中尝试提取（适用于非RESTful路径）
			repoID = extractRepoIDFromPath(c.Request.URL.Path)
		}

		if repoID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "未指定资料库ID"})
			return
		}

		// 6. 模拟检查用户在该Repo的权限（简化实现）
		role := OwnerRole // 简化实现，实际应查询数据库

		// 7. 根据角色和请求方法验证权限
		if !checkPermission(role, c.Request.Method, c.Request.URL.Path) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			return
		}

		// 8. 敏感操作的二级验证
		if isSensitiveOperation(c.Request.Method, c.Request.URL.Path) {
			if !validateSecondaryAuth(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":       "需要二级验证",
					"require_2fa": true,
				})
				return
			}
		}

		// 9. 将用户ID和角色存入上下文
		c.Set("user_id", uint(userID))
		c.Set("repo_role", role)
		c.Set("repo_id", repoID)

		c.Next()
	}
}

// checkPermission 检查角色是否有权限执行特定操作
func checkPermission(role string, method, path string) bool {
	switch role {
	case OwnerRole:
		return true // Owner拥有全部权限
	case CollaboratorRole:
		// Collaborator拥有读写权限，但不能删除库或修改成员
		return method != "DELETE" || !isRepoDeletionPath(path)
	case GuestRole:
		// Guest只有只读权限
		return method == "GET"
	default:
		return false
	}
}

// isSensitiveOperation 判断是否为敏感操作
func isSensitiveOperation(method, path string) bool {
	return (method == "DELETE" && isRepoDeletionPath(path)) ||
		(method == "PUT" && strings.Contains(path, "/members"))
}

// validateSecondaryAuth 验证二级认证（简化实现）
func validateSecondaryAuth(c *gin.Context) bool {
	// 实际应用中应验证额外的token或确认码
	// 这里简化为检查特定header
	return c.GetHeader("X-Secondary-Auth") == "verified"
}

// extractRepoIDFromPath 从路径中提取RepoID
func extractRepoIDFromPath(_ string) string {
	// 实现路径解析逻辑，例如：/api/repos/{repo_id}/files -> 提取repo_id
	// 这里简化实现
	return ""
}

// isRepoDeletionPath 判断是否为资料库删除路径
func isRepoDeletionPath(path string) bool {
	return strings.Contains(path, "/repos/") && strings.HasSuffix(path, "/delete")
}
