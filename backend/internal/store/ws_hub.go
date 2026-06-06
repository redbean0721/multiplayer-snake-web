package store

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"multiplayer-snake-web-backend/internal/models"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type Client struct { Conn *websocket.Conn; Mu sync.Mutex; Name string }

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

type Point struct { X int `json:"x"`; Y int `json:"y"` }
type Food struct { X int `json:"x"`; Y int `json:"y"`; Type string `json:"type"` }

type Snake struct {
	Body       []Point `json:"body"`
	Dir        Point   `json:"-"`
	NextDir    Point   `json:"-"`
	Score      int     `json:"score"`
	BoostSteps int     `json:"-"`
	TickCount  int     `json:"-"`
	
	// ✨ 新增局內任務進度快取
	SessionApples int `json:"-"`
	SessionKills  int `json:"-"`
}

type Hub struct {
	DB           *gorm.DB
	Clients      map[*Client]bool
	Mu           sync.RWMutex
	Snakes       map[*Client]*Snake
	Foods        []Food
	MaxFood      int
	NextFoodTime time.Time
	Cols         int
	Rows         int
}

func NewHub(db *gorm.DB) *Hub {
	h := &Hub{
		DB:           db,
		Clients:      make(map[*Client]bool),
		Snakes:       make(map[*Client]*Snake),
		Foods:        make([]Food, 0),
		MaxFood:      40,
		Cols:         40,
		Rows:         25,
		NextFoodTime: time.Now().Add(2 * time.Second),
	}
	for i := 0; i < 5; i++ { h.spawnSingleFood() }
	go h.RunGameEngine()
	return h
}

func (h *Hub) SyncResources(c *Client) {
	var user models.User
	if err := h.DB.Where("username = ?", c.Name).First(&user).Error; err == nil {
		c.SendJSON("resource_update", map[string]int{"coins": user.Coins, "stars": user.Stars, "diamonds": user.Diamonds})
	}
}

func (h *Hub) spawnSingleFood() {
	if len(h.Foods) >= h.MaxFood { return }
	for attempts := 0; attempts < 50; attempts++ {
		fx, fy := rand.Intn(h.Cols), rand.Intn(h.Rows)
		overlap := false
		for _, snake := range h.Snakes {
			for _, s := range snake.Body { if s.X == fx && s.Y == fy { overlap = true; break } }
		}
		for _, f := range h.Foods {
			if f.X == fx && f.Y == fy { overlap = true; break }
		}
		if !overlap {
			fType := "apple"
			if rand.Intn(100) < 10 { fType = "star" }
			h.Foods = append(h.Foods, Food{X: fx, Y: fy, Type: fType})
			break
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.Mu.Lock()
	h.Clients[c] = true
	snakesData := make(map[string]interface{})
	for client, snake := range h.Snakes { snakesData[client.Name] = snake }
	
	// ✨ 修正：連線時也一併把食物傳給觀戰者
	payload := map[string]interface{}{"snakes": snakesData, "foods": h.Foods, "cols": h.Cols, "rows": h.Rows}
	h.Mu.Unlock()
	c.SendJSON("game_update", payload)
}

func (h *Hub) Unregister(c *Client) {
	h.Mu.Lock()
	// ✨ 玩家若直接關閉網頁斷線，也要結算他的任務進度
	if snake, ok := h.Snakes[c]; ok {
		h.DB.Model(&models.User{}).Where("username = ?", c.Name).Updates(map[string]interface{}{
			"daily_apples": gorm.Expr("daily_apples + ?", snake.SessionApples),
			"daily_kills":  gorm.Expr("daily_kills + ?", snake.SessionKills),
		})
		delete(h.Snakes, c)
	}
	if _, ok := h.Clients[c]; ok { delete(h.Clients, c); c.Conn.Close() }
	h.Mu.Unlock()
}

func (h *Hub) Broadcast(msgType string, payload interface{}) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	for client := range h.Clients { client.SendJSON(msgType, payload) }
}

func (h *Hub) SpawnSnake(c *Client) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	startX := h.Cols/2 + rand.Intn(10) - 5
	startY := h.Rows/2 + rand.Intn(10) - 5
	h.Snakes[c] = &Snake{
		Body:       []Point{{X: startX, Y: startY}, {X: startX - 1, Y: startY}, {X: startX - 2, Y: startY}},
		Dir:        Point{X: 1, Y: 0}, NextDir: Point{X: 1, Y: 0},
		Score:      0, BoostSteps: 0, TickCount: 0,
		SessionApples: 0, SessionKills: 0, // ✨ 初始化
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
	ticker := time.NewTicker(60 * time.Millisecond)
	for range ticker.C {
		h.Mu.Lock()

		if len(h.Snakes) == 0 {
			if len(h.Foods) != 5 {
				h.Foods = make([]Food, 0)
				for i := 0; i < 5; i++ { h.spawnSingleFood() }
			}
			h.NextFoodTime = time.Now().Add(2 * time.Second)
			h.Mu.Unlock()
			continue 
		}

		for c, snake := range h.Snakes {
			snake.TickCount++
			speedFactor := 2
			if snake.BoostSteps > 0 { speedFactor = 1 }
			if snake.TickCount%speedFactor != 0 { continue }
			if snake.BoostSteps > 0 { snake.BoostSteps-- }

			snake.Dir = snake.NextDir
			newHead := Point{X: snake.Body[0].X + snake.Dir.X, Y: snake.Body[0].Y + snake.Dir.Y}

			if newHead.X < 0 || newHead.X >= h.Cols || newHead.Y < 0 || newHead.Y >= h.Rows {
				h.handleDeath(c, snake, nil)
				continue
			}

			collision := false
			var killer *Client
			for otherClient, otherSnake := range h.Snakes {
				for _, s := range otherSnake.Body {
					if s.X == newHead.X && s.Y == newHead.Y {
						collision = true
						if otherClient != c { 
							killer = otherClient 
							// ✨ 給凶手增加擊殺數
							otherSnake.SessionKills++
						}
						break
					}
				}
				if collision { break }
			}

			if collision {
				h.handleDeath(c, snake, killer)
				continue
			}

			snake.Body = append([]Point{newHead}, snake.Body...)

			eatenIdx := -1
			for i, f := range h.Foods {
				if newHead.X == f.X && newHead.Y == f.Y { eatenIdx = i; break }
			}

			if eatenIdx != -1 {
				eatenFood := h.Foods[eatenIdx]
				prevScore := snake.Score
				h.Foods = append(h.Foods[:eatenIdx], h.Foods[eatenIdx+1:]...)

				if eatenFood.Type == "star" {
					snake.Score += 5; snake.BoostSteps += 30
				} else {
					snake.Score += 1
					snake.SessionApples++ // ✨ 增加蘋果食用數
				}

				if prevScore/5 < snake.Score/5 {
					earnedStars := (snake.Score / 5) - (prevScore / 5)
					h.DB.Model(&models.User{}).Where("username = ?", c.Name).UpdateColumn("stars", gorm.Expr("stars + ?", earnedStars))
					h.SyncResources(c)
				}
			} else {
				snake.Body = snake.Body[:len(snake.Body)-1]
			}
		}

		if len(h.Snakes) > 0 {
			if len(h.Foods) < h.MaxFood && time.Now().After(h.NextFoodTime) {
				h.spawnSingleFood()
				ratio := float64(len(h.Foods)) / float64(h.MaxFood)
				baseDelay := 2000.0 + (ratio * 8000.0)
				h.NextFoodTime = time.Now().Add(time.Duration(baseDelay+float64(rand.Intn(4000))) * time.Millisecond)
			}
		}

		snakesData := make(map[string]interface{})
		for c, snake := range h.Snakes { snakesData[c.Name] = snake }
		
		// ✨ 修正：取消過濾，直接把食物發給全部連線的人（包含大廳觀戰者）
		payload := map[string]interface{}{"snakes": snakesData, "foods": h.Foods, "cols": h.Cols, "rows": h.Rows}
		h.Mu.Unlock()
		h.Broadcast("game_update", payload)
	}
}

func (h *Hub) handleDeath(deadClient *Client, deadSnake *Snake, killer *Client) {
	coinsEarned := deadSnake.Score * 10
	
	var user models.User
	if err := h.DB.Where("username = ?", deadClient.Name).First(&user).Error; err == nil {
		user.Coins += coinsEarned
		if deadSnake.Score > user.HighestScore { user.HighestScore = deadSnake.Score }
		h.DB.Save(&user) 
		
		// ✨ 死亡時儲存任務進度
		h.DB.Model(&user).Updates(map[string]interface{}{
			"daily_apples": gorm.Expr("daily_apples + ?", deadSnake.SessionApples),
			"daily_kills":  gorm.Expr("daily_kills + ?", deadSnake.SessionKills),
		})
	}

	if killer != nil {
		h.DB.Model(&models.User{}).Where("username = ?", killer.Name).UpdateColumn("diamonds", gorm.Expr("diamonds + ?", 1))
		h.SyncResources(killer)
	}

	deadClient.SendJSON("game_over", map[string]interface{}{"score": deadSnake.Score, "coins": coinsEarned})
	h.SyncResources(deadClient)
	delete(h.Snakes, deadClient)
}