package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 표준 에러 응답 구조
type ErrorResponse struct {
	Error   string `json:"error"`   // 에러 코드 (프론트엔드에서 매핑용)
	Message string `json:"message"` // 사용자 친화적 메시지 (한글)
}

// RespondWithError 에러 응답 헬퍼
// statusCode: HTTP 상태 코드
// errorCode: 에러 코드 상수 (codes.go 참조)
// message: 사용자에게 보여질 한글 메시지
func RespondWithError(c *gin.Context, statusCode int, errorCode string, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// 자주 사용하는 에러 응답 단축 함수들

func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "로그인이 필요합니다"
	}
	RespondWithError(c, http.StatusUnauthorized, AuthUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "접근 권한이 없습니다"
	}
	RespondWithError(c, http.StatusForbidden, AuthzForbidden, message)
}

func BadRequest(c *gin.Context, errorCode string, message string) {
	RespondWithError(c, http.StatusBadRequest, errorCode, message)
}

func NotFound(c *gin.Context, errorCode string, message string) {
	RespondWithError(c, http.StatusNotFound, errorCode, message)
}

func Conflict(c *gin.Context, errorCode string, message string) {
	RespondWithError(c, http.StatusConflict, errorCode, message)
}

func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "서버 오류가 발생했습니다. 잠시 후 다시 시도해주세요"
	}
	RespondWithError(c, http.StatusInternalServerError, InternalServerError, message)
}

// ValidationError 검증 에러 (선택: 여러 필드 검증 오류)
type ValidationError struct {
	Error   string              `json:"error"`
	Message string              `json:"message"`
	Fields  map[string]string   `json:"fields,omitempty"` // 필드별 오류 메시지
}

func RespondWithValidationError(c *gin.Context, fields map[string]string) {
	c.JSON(http.StatusBadRequest, ValidationError{
		Error:   ValidationInvalidInput,
		Message: "입력값이 올바르지 않습니다",
		Fields:  fields,
	})
}
