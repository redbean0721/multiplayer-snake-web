package store

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client 封裝了 WebSocket 連線與互斥鎖，保證併發寫入安全
type Client struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
	Name string // 記錄玩家名稱
}

func (c *Client) SendJSON(msgType string, payload interface{}) {
	data, _ := json.Marshal(map[string]interface{}{"type": msgType, "payload": payload})
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) SendRaw(data []byte) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Conn.WriteMessage(websocket.TextMessage, data)
}

// 遊戲實體定義
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Snake struct {
	Body    []Point `json:"body"`
	Dir     Point   `json:"-"`
	NextDir Point   `json:"-"`
	Score   int     `json:"score"`
}

type Hub struct {
	Clients map[*Client]bool
	Mu      sync.RWMutex // 保護 Map 的鎖

	// 多人遊戲狀態
	Snakes map[*Client]*Snake
	Food   Point
	Cols   int
	Rows   int
}

func NewHub() *Hub {
	h := &Hub{
		Clients: make(map[*Client]bool),
		Snakes:  make(map[*Client]*Snake),
		Cols:    40, // ✨ 伺服器統一決定地圖大小：寬 40 格
		Rows:    25, // ✨ 伺服器統一決定地圖大小：高 25 格
	}
	h.spawnFood()
	go h.RunGameEngine() // 伺服器一啟動，世界時鐘就開始運轉
	return h
}

func (h *Hub) spawnFood() {
	for {
		fx, fy := rand.Intn(h.Cols), rand.Intn(h.Rows)
		overlap := false
		for _, snake := range h.Snakes {
			for _, s := range snake.Body {
				if s.X == fx && s.Y == fy {
					overlap = true
					break
				}
			}
		}
		if !overlap {
			h.Food = Point{X: fx, Y: fy}
			break
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.Mu.Lock()
	h.Clients[c] = true
	h.Mu.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.Mu.Lock()
	if _, ok := h.Clients[c]; ok {
		delete(h.Clients, c)
		delete(h.Snakes, c) // 玩家斷線時，把他的蛇移除
		c.Conn.Close()
	}
	h.Mu.Unlock()
}

func (h *Hub) Broadcast(msgType string, payload interface{}) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	for client := range h.Clients {
		client.SendJSON(msgType, payload)
	}
}

// 玩家加入遊戲
func (h *Hub) SpawnSnake(c *Client) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	
	// 在地圖中央附近隨機出生
	startX := h.Cols/2 + rand.Intn(10) - 5
	startY := h.Rows/2 + rand.Intn(10) - 5
	
	h.Snakes[c] = &Snake{
		Body:    []Point{{X: startX, Y: startY}, {X: startX - 1, Y: startY}, {X: startX - 2, Y: startY}},
		Dir:     Point{X: 1, Y: 0},
		NextDir: Point{X: 1, Y: 0},
		Score:   0,
	}
}

// 接收玩家方向改變
func (h *Hub) ChangeDirection(c *Client, x, y int) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	if snake, ok := h.Snakes[c]; ok {
		// 防止 180 度大迴轉自殺
		if snake.Dir.X != 0 && x == -snake.Dir.X { return }
		if snake.Dir.Y != 0 && y == -snake.Dir.Y { return }
		snake.NextDir = Point{X: x, Y: y}
	}
}

// ✨ 多人遊戲世界時鐘
func (h *Hub) RunGameEngine() {
	ticker := time.NewTicker(120 * time.Millisecond)
	for range ticker.C {
		h.Mu.Lock()

		// 1. 移動所有蛇
		for c, snake := range h.Snakes {
			snake.Dir = snake.NextDir
			newHead := Point{X: snake.Body[0].X + snake.Dir.X, Y: snake.Body[0].Y + snake.Dir.Y}

			// 撞牆判定
			if newHead.X < 0 || newHead.X >= h.Cols || newHead.Y < 0 || newHead.Y >= h.Rows {
				c.SendJSON("game_over", map[string]int{"score": snake.Score})
				delete(h.Snakes, c) // 撞牆死掉
				continue
			}

			// 撞到自己或其他蛇的判定
			collision := false
			for _, otherSnake := range h.Snakes {
				for _, s := range otherSnake.Body {
					if s.X == newHead.X && s.Y == newHead.Y {
						collision = true
						break
					}
				}
			}
			if collision {
				c.SendJSON("game_over", map[string]int{"score": snake.Score})
				delete(h.Snakes, c) // 撞蛇死掉
				continue
			}

			// 推進身體
			snake.Body = append([]Point{newHead}, snake.Body...)

			// 吃食物判定
			if newHead.X == h.Food.X && newHead.Y == h.Food.Y {
				snake.Score++
				h.spawnFood()
			} else {
				snake.Body = snake.Body[:len(snake.Body)-1] // 沒吃到就縮尾巴
			}
		}

		// 2. 打包所有存活的蛇與狀態給前端
		snakesData := make(map[string]interface{})
		for c, snake := range h.Snakes {
			snakesData[c.Name] = snake // 用玩家名字作為 key 傳給前端
		}

		payload := map[string]interface{}{
			"snakes": snakesData,
			"food":   h.Food,
			"cols":   h.Cols,
			"rows":   h.Rows,
		}
		h.Mu.Unlock() // 廣播前先解鎖，避免死鎖

		// 3. 廣播世界狀態給所有連線者 (不管有沒有在玩都能觀戰)
		h.Broadcast("game_update", payload)
	}
}