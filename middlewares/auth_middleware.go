package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ContextUserID 是放在 gin.Context 裡的使用者 ID 的 key。
const ContextUserID = "userID"

const minSecretLength = 32

var jwtSecret []byte

// SetJWTSecret 設定簽發與驗證 Token 用的密鑰。
// 密鑰必須從環境變數取得，不可寫死在程式碼裡，長度至少 32 字元。
func SetJWTSecret(secret string) error {
	if len(secret) < minSecretLength {
		return errors.New("JWT_SECRET 未設定或長度不足 32 字元")
	}
	jwtSecret = []byte(secret)
	return nil
}

// SignToken 簽發一組帶有效期限的 JWT，payload 只放 user_id 與 email，不放敏感資料。
func SignToken(userID uint, email string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

func unauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
}

// AuthMiddleware 驗證 Authorization: Bearer <token>，通過後把 user_id 放進 Context。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(c, "請提供 Authorization: Bearer <token>")
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(
			parts[1],
			claims,
			func(t *jwt.Token) (interface{}, error) { return jwtSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}), // 只接受 HS256，避免演算法被竄改
			jwt.WithExpirationRequired(),            // 沒有過期時間的 Token 一律拒絕
		)
		if err != nil || !token.Valid {
			unauthorized(c, "Token 無效或已過期")
			return
		}

		// JSON 數字解析出來是 float64，轉回 uint
		userID, ok := claims["user_id"].(float64)
		if !ok || userID <= 0 {
			unauthorized(c, "Token 內容不完整")
			return
		}

		c.Set(ContextUserID, uint(userID))
		c.Next()
	}
}
