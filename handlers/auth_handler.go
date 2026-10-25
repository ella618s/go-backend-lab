package handlers

import (
	"net/http"
	"time"

	"go-backend-lab/database"
	"go-backend-lab/middlewares"
	"go-backend-lab/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 2 * time.Hour

// registerInput 只列出「允許用戶端填寫」的欄位。
// 不直接綁定 models.User，這樣用戶端就沒辦法自己指定 Role 之類的欄位（mass assignment）。
// bcrypt 只處理前 72 bytes，所以密碼上限設 72。
type registerInput struct {
	Name     string `json:"name" binding:"required,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type loginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register 註冊帳號
func Register(c *gin.Context) {
	var in registerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int64
	database.DB.Model(&models.User{}).Where("email = ?", in.Email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "此 Email 已被註冊"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密碼加密失敗"})
		return
	}

	user := models.User{
		Name:     in.Name,
		Email:    in.Email,
		Password: string(hashed),
		Role:     "user", // 角色由伺服器決定，不接受用戶端指定
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立帳號失敗"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "註冊成功", "email": user.Email})
}

// Login 登入並發放 JWT Token
func Login(c *gin.Context) {
	var in loginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 帳號不存在與密碼錯誤回傳同一則訊息，避免讓人探測哪些 Email 已註冊
	const invalid = "帳號或密碼錯誤"

	var user models.User
	if err := database.DB.Where("email = ?", in.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": invalid})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": invalid})
		return
	}

	token, err := middlewares.SignToken(user.ID, user.Email, tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token 產生失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "登入成功", "token": token})
}
