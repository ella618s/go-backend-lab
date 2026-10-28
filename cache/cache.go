// Package cache 提供選用的 Redis 快取。
// 沒有設定 Redis 時（rdb 為 nil），所有函式都直接略過，程式照常運作。
// 快取只是加速用，Redis 出錯時一律當作沒命中，不影響 API 回應。
package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const ttl = 60 * time.Second

var rdb *redis.Client

// Init 連線到 Redis。addr 為空字串代表不使用快取。
func Init(addr string) error {
	if addr == "" {
		rdb = nil
		return nil
	}
	// 逾時設短一點：Redis 掛掉時，API 只是慢一點點，不會被快取拖住
	c := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  300 * time.Millisecond,
		WriteTimeout: 300 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		rdb = nil
		return fmt.Errorf("連線 Redis 失敗: %w", err)
	}
	rdb = c
	return nil
}

// 每位使用者一個 Redis Hash：key 是 tasks:<userID>，field 是狀態篩選條件。
// 這樣使用者的任務有變動時，只要刪掉這一個 key，所有篩選結果就一起失效。
func tasksKey(userID uint) string {
	return fmt.Sprintf("tasks:%d", userID)
}

// GetTasks 讀取快取的任務清單 JSON。回傳 false 代表沒命中（或快取不可用）。
func GetTasks(ctx context.Context, userID uint, field string) ([]byte, bool) {
	if rdb == nil {
		return nil, false
	}
	data, err := rdb.HGet(ctx, tasksKey(userID), field).Bytes()
	if err != nil {
		return nil, false
	}
	return data, true
}

// SetTasks 寫入快取，並設定 60 秒過期，避免舊資料留太久。
func SetTasks(ctx context.Context, userID uint, field string, data []byte) {
	if rdb == nil {
		return
	}
	key := tasksKey(userID)
	pipe := rdb.Pipeline()
	pipe.HSet(ctx, key, field, data)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("寫入快取失敗: %v", err)
	}
}

// InvalidateTasks 使用者的任務有新增、修改、刪除時呼叫，讓快取失效。
func InvalidateTasks(ctx context.Context, userID uint) {
	if rdb == nil {
		return
	}
	if err := rdb.Del(ctx, tasksKey(userID)).Err(); err != nil {
		log.Printf("清除快取失敗: %v", err)
	}
}
