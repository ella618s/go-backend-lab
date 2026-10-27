package chat

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const (
	writeWait   = 10 * time.Second
	pongWait    = 60 * time.Second
	pingPeriod  = 50 * time.Second // 必須小於 pongWait
	maxReadSize = 2048             // 單則訊息最大位元組
	maxTextLen  = 500              // 單則訊息最大字數
	sendBuffer  = 16
)

// Message 是廣播給所有人的訊息格式。
type Message struct {
	From string    `json:"from"`
	Text string    `json:"text"`
	Time time.Time `json:"time"`
}

// Client 代表一條 WebSocket 連線（一位使用者的一個分頁）。
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte // 要寫給這位使用者的訊息，由 writePump 讀取
	name string
}

// Hub 管理所有連線。所有對 clients 的修改都只在 Run 這一個 goroutine 裡進行，
// 所以不需要 mutex：靠 channel 傳遞訊息來避免資料競爭。
type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 64),
	}
}

// Run 持續處理加入、離開與廣播，要用 go hub.Run() 啟動。
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = struct{}{}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// 這位使用者的緩衝已滿（太慢），直接移除，避免拖慢所有人
					delete(h.clients, c)
					close(c.send)
				}
			}
		}
	}
}

// Serve 把一條已升級的連線加入聊天室，並啟動讀、寫兩個 goroutine。
func Serve(hub *Hub, conn *websocket.Conn, name string) {
	c := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendBuffer),
		name: name,
	}
	hub.register <- c
	go c.writePump()
	go c.readPump()
}

// readPump 讀取使用者送來的訊息並交給 Hub 廣播。
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxReadSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return // 連線中斷或逾時
		}
		text := strings.TrimSpace(string(data))
		if text == "" || utf8.RuneCountInString(text) > maxTextLen {
			continue
		}
		out, err := json.Marshal(Message{From: c.name, Text: text, Time: time.Now().UTC()})
		if err != nil {
			continue
		}
		c.hub.broadcast <- out
	}
}

// writePump 把 send 裡的訊息寫給使用者，並定期送 ping 確認對方還在。
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok { // Hub 關閉了這個 channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
