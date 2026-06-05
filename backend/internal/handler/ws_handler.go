package handler

import (
	"context"
	"encoding/json"
	"math/rand"
	"multiplayer-snake-web-backend/internal/store"
	"multiplayer-snake-web-backend/pkg/request"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type ChatPayload struct {
	ID      int64  `json:"id"`
	User    string `json:"user"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func HandleWs(hub *store.Hub, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }

	// 使用剛剛寫好的 Client 封裝
	client := &store.Client{Conn: conn}
	hub.Register(client)

	// 用來控制與關閉目前玩家的遊戲迴圈
	var cancelGame context.CancelFunc
	var moveMu sync.Mutex // 保護方向的鎖
	var currentDir Point
	var nextDir Point

	// 玩家斷線時清理資源
	defer func() {
		if cancelGame != nil {
			cancelGame()
		}
		hub.Unregister(client)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { break }

		var wsMsg request.WsMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil { continue }

		switch wsMsg.Type {
		case "ping":
			client.SendRaw(msg)

		case "chat":
			var chatData ChatPayload
			if err := json.Unmarshal(wsMsg.Payload, &chatData); err == nil {
				chatData.ID = time.Now().UnixNano()
				chatData.Time = time.Now().Format("15:04")
				hub.Broadcast("chat", chatData)
			}

		case "start_game":
			// 如果已經有在玩的遊戲，先把它關掉
			if cancelGame != nil { cancelGame() }
			
			// 建立一個可以被取消的 Context
			ctx, cancel := context.WithCancel(context.Background())
			cancelGame = cancel

			// 解析前端傳來的網格大小
			var startData struct { Cols int `json:"cols"`; Rows int `json:"rows"` }
			json.Unmarshal(wsMsg.Payload, &startData)
			cols, rows := startData.Cols, startData.Rows

			// 初始化遊戲狀態
			moveMu.Lock()
			currentDir = Point{X: 1, Y: 0}
			nextDir = Point{X: 1, Y: 0}
			moveMu.Unlock()

			score := 0
			startX, startY := cols/2, rows/2
			snake := []Point{{X: startX, Y: startY}, {X: startX - 1, Y: startY}, {X: startX - 2, Y: startY}}
			var food Point

			// 生成食物的閉包函數
			spawnFood := func() {
				for {
					fx, fy := rand.Intn(cols), rand.Intn(rows)
					overlap := false
					for _, s := range snake {
						if s.X == fx && s.Y == fy { overlap = true; break }
					}
					if !overlap { food = Point{X: fx, Y: fy}; break }
				}
			}
			spawnFood()

			// ✨ 啟動遊戲引擎 (背景獨立運作)
			go func(ctx context.Context) {
				// 120 毫秒走一步
				ticker := time.NewTicker(120 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done(): // 收到取消訊號就結束
						return
					case <-ticker.C: // 每 120ms 觸發一次
						moveMu.Lock()
						currentDir = nextDir
						moveMu.Unlock()

						newHead := Point{X: snake[0].X + currentDir.X, Y: snake[0].Y + currentDir.Y}

						// 1. 撞牆判定
						if newHead.X < 0 || newHead.X >= cols || newHead.Y < 0 || newHead.Y >= rows {
							client.SendJSON("game_over", map[string]int{"score": score})
							return
						}

						// 2. 撞到自己判定
						collision := false
						for _, s := range snake {
							if s.X == newHead.X && s.Y == newHead.Y { collision = true; break }
						}
						if collision {
							client.SendJSON("game_over", map[string]int{"score": score})
							return
						}

						// 蛇頭往前伸
						snake = append([]Point{newHead}, snake...)

						// 3. 吃食物判定
						if newHead.X == food.X && newHead.Y == food.Y {
							score++
							spawnFood()
						} else {
							snake = snake[:len(snake)-1] // 沒吃到就縮尾巴
						}

						// 4. 將最新畫面傳給前端
						client.SendJSON("game_update", map[string]interface{}{
							"snake": snake,
							"food":  food,
							"score": score,
						})
					}
				}
			}(ctx)

		case "move":
			var m Point
			if err := json.Unmarshal(wsMsg.Payload, &m); err == nil {
				moveMu.Lock()
				// 防止 180 度大迴轉自殺
				if currentDir.X != 0 && m.X == -currentDir.X {
				} else if currentDir.Y != 0 && m.Y == -currentDir.Y {
				} else {
					nextDir = m
				}
				moveMu.Unlock()
			}
		}
	}
}