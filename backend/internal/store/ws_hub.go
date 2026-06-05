package store

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
	Name string
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
	Mu      sync.RWMutex

	Snakes       map[*Client]*Snake
	Foods        []Point
	MaxFood      int       
	NextFoodTime time.Time 
	Cols         int
	Rows         int
}

func NewHub() *Hub {
	h := &Hub{
		Clients:      make(map[*Client]bool),
		Snakes:       make(map[*Client]*Snake),
		Foods:        make([]Point, 0),
		MaxFood:      40, 
		Cols:         40,
		Rows:         25,
		NextFoodTime: time.Now().Add(2 * time.Second), 
	}

	// 一開始先放 5 顆食物作為開局基礎
	for i := 0; i < 5; i++ {
		h.spawnSingleFood()
	}

	go h.RunGameEngine()
	return h
}

func (h *Hub) spawnSingleFood() {
	if len(h.Foods) >= h.MaxFood {
		return
	}

	for attempts := 0; attempts < 50; attempts++ { 
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

		for _, f := range h.Foods {
			if f.X == fx && f.Y == fy {
				overlap = true
				break
			}
		}

		if !overlap {
			h.Foods = append(h.Foods, Point{X: fx, Y: fy})
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
		delete(h.Snakes, c)
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

func (h *Hub) SpawnSnake(c *Client) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	
	startX := h.Cols/2 + rand.Intn(10) - 5
	startY := h.Rows/2 + rand.Intn(10) - 5
	
	h.Snakes[c] = &Snake{
		Body:    []Point{{X: startX, Y: startY}, {X: startX - 1, Y: startY}, {X: startX - 2, Y: startY}},
		Dir:     Point{X: 1, Y: 0},
		NextDir: Point{X: 1, Y: 0},
		Score:   0,
	}
}

func (h *Hub) ChangeDirection(c *Client, x, y int) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	if snake, ok := h.Snakes[c]; ok {
		if snake.Dir.X != 0 && x == -snake.Dir.X { return }
		if snake.Dir.Y != 0 && y == -snake.Dir.Y { return }
		snake.NextDir = Point{X: x, Y: y}
	}
}

func (h *Hub) RunGameEngine() {
	ticker := time.NewTicker(120 * time.Millisecond)
	for range ticker.C {
		h.Mu.Lock()

		for c, snake := range h.Snakes {
			snake.Dir = snake.NextDir
			newHead := Point{X: snake.Body[0].X + snake.Dir.X, Y: snake.Body[0].Y + snake.Dir.Y}

			if newHead.X < 0 || newHead.X >= h.Cols || newHead.Y < 0 || newHead.Y >= h.Rows {
				c.SendJSON("game_over", map[string]int{"score": snake.Score})
				delete(h.Snakes, c)
				continue
			}

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
				delete(h.Snakes, c)
				continue
			}

			snake.Body = append([]Point{newHead}, snake.Body...)

			eatenIdx := -1
			for i, f := range h.Foods {
				if newHead.X == f.X && newHead.Y == f.Y {
					eatenIdx = i
					break
				}
			}

			if eatenIdx != -1 {
				snake.Score++
				h.Foods = append(h.Foods[:eatenIdx], h.Foods[eatenIdx+1:]...)
			} else {
				snake.Body = snake.Body[:len(snake.Body)-1]
			}
		}

		// ✨ 修正：只有在場上有玩家（蛇）的時候，才進行大自然生態運作
		if len(h.Snakes) > 0 {
			if len(h.Foods) < h.MaxFood && time.Now().After(h.NextFoodTime) {
				h.spawnSingleFood() 

				// ✨ 調整為更長的間隔
				ratio := float64(len(h.Foods)) / float64(h.MaxFood)
				// 基礎時間拉長到 2秒 ~ 10秒 (baseDelay + ratio * 8000)
				baseDelay := 2000.0 + (ratio * 8000.0) 
				randomOffset := rand.Intn(4000)       

				totalDelayMs := time.Duration(baseDelay+float64(randomOffset)) * time.Millisecond
				h.NextFoodTime = time.Now().Add(totalDelayMs)
			}
		} else {
			// ✨ 如果沒人玩，把下次生成時間往後推，確保第一個玩家加進來時不會瞬間爆出一堆食物
			h.NextFoodTime = time.Now().Add(2 * time.Second)
		}

		snakesData := make(map[string]interface{})
		for c, snake := range h.Snakes {
			snakesData[c.Name] = snake
		}

		payload := map[string]interface{}{
			"snakes": snakesData,
			"foods":  h.Foods,
			"cols":   h.Cols,
			"rows":   h.Rows,
		}
		h.Mu.Unlock()

		h.Broadcast("game_update", payload)
	}
}