package models

import (
	"testing"
)

// 測試 User 結構體與欄位是否正確賦值
func TestUserStruct(t *testing.T) {
	// 1. 準備測試資料
	user := User{
		Name:  "淑華",
		Email: "shuhua@example.com",
		Role:  "Android & Go Engineer",
	}

	// 2. 驗證 Name 欄位
	if user.Name != "淑華" {
		t.Errorf("預期 Name 是 淑華，但實際得到的是: %s", user.Name)
	}

	// 3. 驗證 Email 欄位
	if user.Email != "shuhua@example.com" {
		t.Errorf("預期 Email 是 shuhua@example.com，但實際得到的是: %s", user.Email)
	}
}
