package handlers

import (
	"net/http"

	"go-backend-lab/chat"
	"go-backend-lab/database"
	"go-backend-lab/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 不覆寫 CheckOrigin：預設只允許同源（Origin 與 Host 相同）的瀏覽器連線，
// 避免別的網站用你的登入狀態偷偷連進來。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ChatWS 把已通過驗證的 HTTP 請求升級成 WebSocket，並加入聊天室。
func ChatWS(hub *chat.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}

		var user models.User
		if err := database.DB.Select("id", "name").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到使用者"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return // Upgrade 失敗時已自行回應錯誤
		}
		chat.Serve(hub, conn, user.Name)
	}
}

// ChatPage 回傳簡單的測試頁面，用瀏覽器就能試聊天室。
func ChatPage(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(chatHTML))
}

const chatHTML = `<!doctype html>
<html lang="zh-Hant">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>聊天室測試</title>
<style>
  body { font-family: sans-serif; max-width: 640px; margin: 24px auto; padding: 0 16px; }
  #log { border: 1px solid #ccc; height: 320px; overflow-y: auto; padding: 8px; margin: 12px 0; }
  .row { display: flex; gap: 8px; margin-bottom: 8px; }
  input { flex: 1; padding: 6px; }
</style>
</head>
<body>
<h2>聊天室測試</h2>
<div class="row">
  <input id="token" placeholder="貼上登入取得的 token">
  <button id="connect">連線</button>
</div>
<div id="log"></div>
<div class="row">
  <input id="msg" placeholder="輸入訊息後按 Enter">
  <button id="send">送出</button>
</div>
<script>
var ws = null;
var log = document.getElementById("log");

function addLine(text) {
  var div = document.createElement("div");
  div.textContent = text; // 用 textContent，不用 innerHTML，避免別人送來的內容被當成 HTML 執行（XSS）
  log.appendChild(div);
  log.scrollTop = log.scrollHeight;
}

document.getElementById("connect").onclick = function () {
  var token = document.getElementById("token").value.trim();
  if (!token) { addLine("請先貼上 token"); return; }
  var proto = location.protocol === "https:" ? "wss://" : "ws://";
  ws = new WebSocket(proto + location.host + "/api/ws?token=" + encodeURIComponent(token));
  ws.onopen = function () { addLine("[已連線]"); };
  ws.onclose = function () { addLine("[連線已關閉]"); };
  ws.onmessage = function (e) {
    var m = JSON.parse(e.data);
    addLine(m.from + "：" + m.text);
  };
};

function sendMsg() {
  var input = document.getElementById("msg");
  if (ws && ws.readyState === 1 && input.value.trim()) {
    ws.send(input.value);
    input.value = "";
  }
}
document.getElementById("send").onclick = sendMsg;
document.getElementById("msg").onkeydown = function (e) { if (e.key === "Enter") sendMsg(); };
</script>
</body>
</html>`
