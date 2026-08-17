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

// ------------------- 核心输出函数 -------------------
// 所有 API 接口都返回 http.StatusOK，依靠 code 来区分业务状态

// JSON 底层通用输出函数，直接输出自定义响应体
func JSON[T any](w http.ResponseWriter, httpStatus int, code int, msg string, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	resp := Response[T]{
		Code: code,
		Msg:  msg,
		Data: data,
	}

	// 忽略编码错误
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("Failed to encode json resp", "error", err)
	}
}

// Success 成功响应：HTTP 200 + Code 200
func Success[T any](w http.ResponseWriter, data T) {
	JSON(w, http.StatusOK, 200, "success", data)
}

// SuccessWithMsg 成功响应：HTTP 200 + Code 200
func SuccessWithMsg[T any](w http.ResponseWriter, msg string) {
	JSON[any](w, http.StatusOK, 200, msg, nil)
}

// SuccessWithData 成功响应（自定义消息）：HTTP 200 + Code 200
func SuccessWithData[T any](w http.ResponseWriter, msg string, data T) {
	JSON(w, http.StatusOK, 200, msg, data)
}

// Error 业务错误响应：HTTP 200 但 Code 非 200（业务异常，前端走 catch 或判断 code）
func Error(w http.ResponseWriter, code int, msg string) {
	// 注意泛型传 interface{}，因为 Data 为 nil
	JSON[any](w, http.StatusOK, code, msg, nil)
}

// Fail 系统/参数错误响应：HTTP 200，Code 非 200
// 用于参数校验失败、系统内部错误等场景
func Fail(w http.ResponseWriter, code int, msg string) {
	JSON[any](w, http.StatusOK, code, msg, nil)
}
