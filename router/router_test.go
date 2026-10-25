package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-backend-lab/database"
	"go-backend-lab/middlewares"

	"github.com/gin-gonic/gin"
)

const testSecret = "test-secret-key-that-is-at-least-32-chars"

// setup 為每個測試建立獨立的記憶體資料庫與路由。
func setup(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	if err := middlewares.SetJWTSecret(testSecret); err != nil {
		t.Fatal(err)
	}
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	if err := database.Init(dsn); err != nil {
		t.Fatal(err)
	}
	return New()
}

func call(t *testing.T, r http.Handler, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("回應不是合法 JSON: %v\n%s", err, w.Body.String())
	}
	return m
}

// registerAndLogin 註冊並登入，回傳 Token。
func registerAndLogin(t *testing.T, r http.Handler, name, email string) string {
	t.Helper()
	w := call(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"name": name, "email": email, "password": "password123",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("註冊失敗，狀態碼 %d: %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"email": email, "password": "password123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("登入失敗，狀態碼 %d: %s", w.Code, w.Body.String())
	}
	token, _ := decode(t, w)["token"].(string)
	if token == "" {
		t.Fatal("登入後沒有拿到 token")
	}
	return token
}

func TestProtectedRoutesRequireToken(t *testing.T) {
	r := setup(t)

	for _, path := range []string{"/api/me", "/api/tasks"} {
		w := call(t, r, http.MethodGet, path, "", nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s 沒帶 Token 應回 401，實際 %d", path, w.Code)
		}
	}

	w := call(t, r, http.MethodGet, "/api/tasks", "not-a-real-token", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("假 Token 應回 401，實際 %d", w.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	r := setup(t)

	w := call(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"name": "A", "email": "not-an-email", "password": "password123",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("Email 格式錯誤應回 400，實際 %d", w.Code)
	}

	w = call(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"name": "A", "email": "a@example.com", "password": "short",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("密碼太短應回 400，實際 %d", w.Code)
	}

	registerAndLogin(t, r, "A", "dup@example.com")
	w = call(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"name": "A2", "email": "dup@example.com", "password": "password123",
	})
	if w.Code != http.StatusConflict {
		t.Errorf("重複 Email 應回 409，實際 %d", w.Code)
	}
}

func TestResponsesNeverLeakPassword(t *testing.T) {
	r := setup(t)
	token := registerAndLogin(t, r, "Alice", "alice@example.com")

	call(t, r, http.MethodPost, "/api/tasks", token, map[string]string{"title": "t1"})

	for _, path := range []string{"/api/me", "/api/tasks"} {
		body := strings.ToLower(call(t, r, http.MethodGet, path, token, nil).Body.String())
		if strings.Contains(body, "password") || strings.Contains(body, "$2a$") {
			t.Errorf("GET %s 的回應含有密碼資訊: %s", path, body)
		}
	}
}

func TestRegisterCannotChooseRole(t *testing.T) {
	r := setup(t)

	call(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"name": "Eve", "email": "eve@example.com", "password": "password123", "role": "admin",
	})
	w := call(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"email": "eve@example.com", "password": "password123",
	})
	token, _ := decode(t, w)["token"].(string)

	me := decode(t, call(t, r, http.MethodGet, "/api/me", token, nil))
	user := me["user"].(map[string]interface{})
	if user["role"] != "user" {
		t.Errorf("註冊時帶 role=admin，實際角色應為 user，得到 %v", user["role"])
	}
}

func TestUsersOnlySeeTheirOwnTasks(t *testing.T) {
	r := setup(t)
	alice := registerAndLogin(t, r, "Alice", "alice@example.com")
	bob := registerAndLogin(t, r, "Bob", "bob@example.com")

	w := call(t, r, http.MethodPost, "/api/tasks", alice, map[string]string{"title": "Alice 的任務"})
	if w.Code != http.StatusCreated {
		t.Fatalf("Alice 新增任務失敗: %d %s", w.Code, w.Body.String())
	}
	task := decode(t, w)["task"].(map[string]interface{})
	id := int(task["ID"].(float64))

	// Bob 看不到 Alice 的任務
	bobList := decode(t, call(t, r, http.MethodGet, "/api/tasks", bob, nil))
	if bobList["count"].(float64) != 0 {
		t.Errorf("Bob 不應看到任何任務，count=%v", bobList["count"])
	}

	// Bob 不能修改或刪除 Alice 的任務，且不能得知它存在（回 404）
	path := fmt.Sprintf("/api/tasks/%d", id)
	w = call(t, r, http.MethodPut, path, bob, map[string]string{"title": "被竄改"})
	if w.Code != http.StatusNotFound {
		t.Errorf("Bob 修改 Alice 的任務應回 404，實際 %d", w.Code)
	}
	w = call(t, r, http.MethodDelete, path, bob, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("Bob 刪除 Alice 的任務應回 404，實際 %d", w.Code)
	}

	// Alice 的任務沒被動到
	aliceList := decode(t, call(t, r, http.MethodGet, "/api/tasks", alice, nil))
	if aliceList["count"].(float64) != 1 {
		t.Errorf("Alice 應該仍有 1 個任務，count=%v", aliceList["count"])
	}
}

func TestTaskCRUDAndStatusFilter(t *testing.T) {
	r := setup(t)
	token := registerAndLogin(t, r, "Alice", "alice@example.com")

	w := call(t, r, http.MethodPost, "/api/tasks", token, map[string]string{"title": "寫測試"})
	id := int(decode(t, w)["task"].(map[string]interface{})["ID"].(float64))
	path := fmt.Sprintf("/api/tasks/%d", id)

	w = call(t, r, http.MethodPut, path, token, map[string]string{"title": "寫測試", "status": "done"})
	if w.Code != http.StatusOK {
		t.Fatalf("更新失敗: %d %s", w.Code, w.Body.String())
	}

	done := decode(t, call(t, r, http.MethodGet, "/api/tasks?status=done", token, nil))
	pending := decode(t, call(t, r, http.MethodGet, "/api/tasks?status=pending", token, nil))
	if done["count"].(float64) != 1 || pending["count"].(float64) != 0 {
		t.Errorf("狀態篩選錯誤: done=%v pending=%v", done["count"], pending["count"])
	}

	w = call(t, r, http.MethodPost, "/api/tasks", token, map[string]string{"title": "x", "status": "weird"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("不合法的 status 應回 400，實際 %d", w.Code)
	}

	if w = call(t, r, http.MethodDelete, path, token, nil); w.Code != http.StatusOK {
		t.Errorf("刪除自己的任務應回 200，實際 %d", w.Code)
	}
}
