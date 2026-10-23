package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var JwtSecret = []byte("my_super_secret_key_for_japan_market")

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 從 HTTP Header 的 Authorization 欄位拿 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供授權憑證 (Authorization Header missing)"})
			c.Abort()
			return
		}

		// 2. 格式切割
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 格式錯誤，應為 Bearer <token>"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. 解析並驗證 Token (使用 MapClaims 來讀取裡面的資料)
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 無效或已過期"})
			c.Abort()
			return
		}

		// 4. ✨ 關鍵：從 Token 的 Claims 中把 user_id 取出來，並塞進 Context
		// (因為 JSON 解析出來的數字預設會是 float64，我們把它轉成 uint)
		if userIdFloat, ok := claims["user_id"].(float64); ok {
			c.Set("userID", uint(userIdFloat))
		} else if userIdUint, ok := claims["userID"].(uint); ok {
			c.Set("userID", userIdUint)
		}

		// 5. 驗證通過，放行！
		c.Next()
	}
}
