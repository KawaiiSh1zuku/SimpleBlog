package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Response 定义统一的响应结构体
// 使用泛型 T 来约束 Data 字段的类型，保证 Handler 返回的数据结构清晰
type Response[T any] struct {
	Code int    `json:"code"`           // 业务状态码，与 HTTP 状态码标准相同
	Msg  string `json:"msg"`            // 提示信息
	Data T      `json:"data,omitempty"` // 业务数据，omitempty 表示 data 为空时忽略此字段
}

// JSON 输出统一响应。应用层业务结果始终使用 HTTP 200，真实状态由 code 表示。
func JSON[T any](w http.ResponseWriter, code int, msg string, data T) {
	if writer, ok := w.(interface{ SetBusinessCode(int) }); ok {
		writer.SetBusinessCode(code)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	resp := Response[T]{
		Code: code,
		Msg:  msg,
		Data: data,
	}

	// 忽略编码错误
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// Success 成功响应：HTTP 200 + Code 200
func Success[T any](w http.ResponseWriter, data T) {
	JSON(w, http.StatusOK, "success", data)
}

// SuccessWithMsg 成功响应：HTTP 200 + Code 200
func SuccessWithMsg[T any](w http.ResponseWriter, msg string) {
	JSON[any](w, http.StatusOK, msg, nil)
}

// SuccessWithData 成功响应（自定义消息）：HTTP 200 + Code 200
func SuccessWithData[T any](w http.ResponseWriter, msg string, data T) {
	JSON(w, http.StatusOK, msg, data)
}

// Error 业务错误响应：HTTP 200 但 Code 非 200（业务异常，前端走 catch 或判断 code）
func Error(w http.ResponseWriter, code int, msg string) {
	JSON[any](w, code, msg, nil)
}

// Fail 系统/参数错误响应：HTTP 200，Code 非 200
// 用于参数校验失败、系统内部错误等场景
func Fail(w http.ResponseWriter, code int, msg string) {
	JSON[any](w, code, msg, nil)
}
