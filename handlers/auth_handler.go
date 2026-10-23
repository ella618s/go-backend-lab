package handlers

import (
	"net/http"
	"time"

	"go-backend-lab/database"
	"go-backend-lab/middlewares"
	"go-backend-lab/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Register 註冊帳號
func Register(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 將明文密碼進行 bcrypt 雜湊加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密碼加密失敗"})
		return
	}
	user.Password = string(hashedPassword)

	// 2. 存入資料庫
	if result := database.DB.Create(&user); result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "註冊失敗，Email 可能已被註冊"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "註冊成功！", "email": user.Email})
}

// Login 登入並發放 JWT Token
func Login(c *gin.Context) {
	var loginBody struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 在資料庫尋找使用者
	var user models.User
	if result := database.DB.Where("email = ?", loginBody.Email).First(&user); result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "帳號或密碼錯誤"})
		return
	}

	// 2. 比對密碼是否正確
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginBody.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "帳號或密碼錯誤"})
		return
	}

	// 3. 密碼正確，簽發 JWT Token (設定 2 小時後過期)
	claims := jwt.MapClaims{
		"user_id": user.ID, // ✨ 加上這行，把使用者 ID 裝進 Token 裡面！
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 2).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middlewares.JwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token 產生失敗"})
		return
	}

	// 4. 回傳 Token 給前端
	c.JSON(http.StatusOK, gin.H{
		"message": "登入成功！",
		"token":   tokenString,
	})
}
