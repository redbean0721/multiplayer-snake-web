package response

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Response_Model struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Time    int64       `json:"time"`
}

// JSONResponse 為通用 API 回應函式，接受以下參數:
// - ctx: gin.Context, 用於處理 HTTP 請求和回應
// - status_code: int, IETF 標準 HTTP 狀態碼 (例如 200, 400, 500)
// - message: string, 回應訊息 (例如 "success", "error")
// - args: 可選參數, 可以是 biz_code (int) 或 data (interface{})
// biz_code 為自定義狀態碼, 用於區分不同的業務錯誤類型 (例如 1001: 用戶不存在, 1002: 密碼錯誤)
// 調用範例:
// JSONResponse(ctx, 200, "success", 1000, map[string]interface{}{"user_id": 123})
// JSONResponse(ctx, 400, "invalid request", 1001)
// JSONResponse(ctx, 500, "internal server error")
func JSONResponse(ctx *gin.Context, status_code int, message string, args ...interface{}) {
	biz_code := status_code
	var data interface{} = nil

	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			// 如果傳入 int, 則視為 biz_code
			biz_code = v
		default:
			// 如果是其他類型, 則視為 data
			data = v
		}
	}

	ctx.JSON(status_code, Response_Model{
		Code:    biz_code,
		Message: message,
		Data:    data,
		Time:    time.Now().UnixMilli(), // 13位時間戳
	})
}