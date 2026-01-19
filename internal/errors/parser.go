package errors

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ErrorInfo 에러 정보 구조
type ErrorInfo struct {
	Code    string // 에러 코드 (codes.go 참조)
	Message string // 사용자 친화적 메시지
}

// ParseError 에러를 파싱하여 사용자 친화적인 메시지와 코드로 변환
// 보안상 민감한 정보는 숨기되, 사용자가 문제를 해결할 수 있는 정보 제공
func ParseError(err error, context string) ErrorInfo {
	if err == nil {
		return ErrorInfo{
			Code:    InternalServerError,
			Message: "서버 오류가 발생했습니다",
		}
	}

	errStr := err.Error()
	errStrLower := strings.ToLower(errStr)

	// 1. GORM 기본 에러
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrorInfo{
			Code:    ResourceNotFound,
			Message: getNotFoundMessage(context),
		}
	}

	// 2. PostgreSQL 에러 파싱

	// 2-1. Unique constraint violation (23505)
	if strings.Contains(errStrLower, "duplicate key") || strings.Contains(errStrLower, "unique constraint") {
		return parseDuplicateKeyError(errStr, context)
	}

	// 2-2. Foreign key constraint violation (23503)
	if strings.Contains(errStrLower, "foreign key constraint") {
		return parseForeignKeyError(errStr, context)
	}

	// 2-3. Not null constraint violation (23502)
	if strings.Contains(errStrLower, "null value") && strings.Contains(errStrLower, "violates not-null constraint") {
		return parseNotNullError(errStr, context)
	}

	// 2-4. Check constraint violation (23514)
	if strings.Contains(errStrLower, "check constraint") {
		return parseCheckConstraintError(errStr, context)
	}

	// 3. 비즈니스 로직 에러 (service layer에서 정의된 에러)
	if strings.Contains(errStr, "매장을 찾을 수 없습니다") {
		return ErrorInfo{Code: StoreNotFound, Message: "매장을 찾을 수 없습니다"}
	}
	if strings.Contains(errStr, "매장 접근 권한이 없습니다") {
		return ErrorInfo{Code: AuthzForbidden, Message: "매장 접근 권한이 없습니다"}
	}
	if strings.Contains(errStr, "이미 관리 중인 매장이 있습니다") {
		return ErrorInfo{Code: StoreAlreadyManaged, Message: "이미 관리 중인 매장이 있습니다"}
	}

	// 4. 네트워크/연결 에러
	if strings.Contains(errStrLower, "connection refused") ||
	   strings.Contains(errStrLower, "no such host") ||
	   strings.Contains(errStrLower, "timeout") {
		return ErrorInfo{
			Code:    InternalExternalAPI,
			Message: "외부 서비스 연결에 실패했습니다. 잠시 후 다시 시도해주세요",
		}
	}

	// 5. 기본 내부 서버 오류
	return ErrorInfo{
		Code:    InternalServerError,
		Message: getDefaultErrorMessage(context),
	}
}

// parseDuplicateKeyError Unique constraint 위반 에러 파싱
func parseDuplicateKeyError(errStr string, context string) ErrorInfo {
	errLower := strings.ToLower(errStr)

	// 사업자번호 중복
	if strings.Contains(errLower, "business_number") || strings.Contains(errLower, "idx_stores_business_number") {
		return ErrorInfo{
			Code:    StoreBusinessNumberExists,
			Message: "이미 등록된 사업자번호입니다",
		}
	}

	// 매장 slug 중복
	if strings.Contains(errLower, "slug") || strings.Contains(errLower, "idx_stores_slug") {
		return ErrorInfo{
			Code:    ResourceAlreadyExists,
			Message: "이미 사용 중인 매장 식별자입니다",
		}
	}

	// 이메일 중복
	if strings.Contains(errLower, "email") || strings.Contains(errLower, "idx_users_email") {
		return ErrorInfo{
			Code:    AuthEmailAlreadyExists,
			Message: "이미 사용 중인 이메일입니다",
		}
	}

	// 닉네임 중복
	if strings.Contains(errLower, "nickname") || strings.Contains(errLower, "idx_users_nickname") {
		return ErrorInfo{
			Code:    AuthNicknameExists,
			Message: "이미 사용 중인 닉네임입니다",
		}
	}

	// 매장 ID 중복 (business_registrations)
	if strings.Contains(errLower, "store_id") || strings.Contains(errLower, "idx_business_registrations_store_id") {
		return ErrorInfo{
			Code:    StoreAlreadyOwned,
			Message: "이미 소유권이 등록된 매장입니다",
		}
	}

	// 리뷰 중복
	if strings.Contains(errLower, "reviews") && (strings.Contains(errLower, "user_id") || strings.Contains(errLower, "store_id")) {
		return ErrorInfo{
			Code:    ReviewAlreadyExists,
			Message: "이미 이 매장에 리뷰를 작성하셨습니다",
		}
	}

	// Primary key 중복
	if strings.Contains(errLower, "pkey") || strings.Contains(errLower, "primary key") {
		return ErrorInfo{
			Code:    ResourceAlreadyExists,
			Message: "이미 존재하는 데이터입니다. 다시 시도해주세요",
		}
	}

	// 기본 중복 메시지
	return ErrorInfo{
		Code:    ResourceAlreadyExists,
		Message: "이미 존재하는 데이터입니다",
	}
}

// parseForeignKeyError Foreign key constraint 위반 에러 파싱
func parseForeignKeyError(errStr string, context string) ErrorInfo {
	errLower := strings.ToLower(errStr)

	// 삭제 시 참조 중인 데이터가 있는 경우
	if strings.Contains(errLower, "still referenced") || strings.Contains(errLower, "is still referenced by") {
		if strings.Contains(context, "store") || strings.Contains(context, "매장") {
			return ErrorInfo{
				Code:    ResourceConflict,
				Message: "매장에 연결된 데이터가 있어 삭제할 수 없습니다",
			}
		}
		if strings.Contains(context, "user") || strings.Contains(context, "사용자") {
			return ErrorInfo{
				Code:    ResourceConflict,
				Message: "사용자에 연결된 데이터가 있어 삭제할 수 없습니다",
			}
		}
		return ErrorInfo{
			Code:    ResourceConflict,
			Message: "연결된 데이터가 있어 삭제할 수 없습니다",
		}
	}

	// 존재하지 않는 참조 데이터
	if strings.Contains(errLower, "user_id") || strings.Contains(errLower, "fk_users") {
		return ErrorInfo{
			Code:    ResourceNotFound,
			Message: "존재하지 않는 사용자입니다",
		}
	}
	if strings.Contains(errLower, "store_id") || strings.Contains(errLower, "fk_stores") {
		return ErrorInfo{
			Code:    StoreNotFound,
			Message: "존재하지 않는 매장입니다",
		}
	}
	if strings.Contains(errLower, "post_id") || strings.Contains(errLower, "fk_posts") {
		return ErrorInfo{
			Code:    PostNotFound,
			Message: "존재하지 않는 게시글입니다",
		}
	}

	return ErrorInfo{
		Code:    ResourceNotFound,
		Message: "참조하는 데이터를 찾을 수 없습니다",
	}
}

// parseNotNullError Not null constraint 위반 에러 파싱
func parseNotNullError(errStr string, context string) ErrorInfo {
	errLower := strings.ToLower(errStr)

	// 필드명 추출
	if strings.Contains(errLower, "email") {
		return ErrorInfo{Code: ValidationRequired, Message: "이메일은 필수 항목입니다"}
	}
	if strings.Contains(errLower, "password") {
		return ErrorInfo{Code: ValidationRequired, Message: "비밀번호는 필수 항목입니다"}
	}
	if strings.Contains(errLower, "name") {
		return ErrorInfo{Code: ValidationRequired, Message: "이름은 필수 항목입니다"}
	}
	if strings.Contains(errLower, "nickname") {
		return ErrorInfo{Code: ValidationRequired, Message: "닉네임은 필수 항목입니다"}
	}

	return ErrorInfo{
		Code:    ValidationRequired,
		Message: "필수 항목이 누락되었습니다",
	}
}

// parseCheckConstraintError Check constraint 위반 에러 파싱
func parseCheckConstraintError(errStr string, context string) ErrorInfo {
	errLower := strings.ToLower(errStr)

	if strings.Contains(errLower, "rating") {
		return ErrorInfo{
			Code:    ReviewInvalidRating,
			Message: "평점은 1~5 사이의 값이어야 합니다",
		}
	}

	if strings.Contains(errLower, "latitude") || strings.Contains(errLower, "longitude") {
		return ErrorInfo{
			Code:    ValidationInvalidRange,
			Message: "위도/경도 값이 유효하지 않습니다",
		}
	}

	return ErrorInfo{
		Code:    ValidationInvalidInput,
		Message: "입력값이 유효하지 않습니다",
	}
}

// getNotFoundMessage context에 따른 Not Found 메시지
func getNotFoundMessage(context string) string {
	contextLower := strings.ToLower(context)

	if strings.Contains(contextLower, "store") || strings.Contains(contextLower, "매장") {
		return "매장을 찾을 수 없습니다"
	}
	if strings.Contains(contextLower, "user") || strings.Contains(contextLower, "사용자") {
		return "사용자를 찾을 수 없습니다"
	}
	if strings.Contains(contextLower, "review") || strings.Contains(contextLower, "리뷰") {
		return "리뷰를 찾을 수 없습니다"
	}
	if strings.Contains(contextLower, "post") || strings.Contains(contextLower, "게시") {
		return "게시글을 찾을 수 없습니다"
	}
	if strings.Contains(contextLower, "comment") || strings.Contains(contextLower, "댓글") {
		return "댓글을 찾을 수 없습니다"
	}
	if strings.Contains(contextLower, "chat") || strings.Contains(contextLower, "채팅") {
		return "채팅방을 찾을 수 없습니다"
	}
	if strings.Contains(contextLower, "notification") || strings.Contains(contextLower, "알림") {
		return "알림을 찾을 수 없습니다"
	}

	return "요청한 데이터를 찾을 수 없습니다"
}

// getDefaultErrorMessage context에 따른 기본 에러 메시지
func getDefaultErrorMessage(context string) string {
	contextLower := strings.ToLower(context)

	if strings.Contains(contextLower, "create") || strings.Contains(contextLower, "생성") || strings.Contains(contextLower, "등록") {
		return "등록 중 오류가 발생했습니다. 잠시 후 다시 시도해주세요"
	}
	if strings.Contains(contextLower, "update") || strings.Contains(contextLower, "수정") {
		return "수정 중 오류가 발생했습니다. 잠시 후 다시 시도해주세요"
	}
	if strings.Contains(contextLower, "delete") || strings.Contains(contextLower, "삭제") {
		return "삭제 중 오류가 발생했습니다. 잠시 후 다시 시도해주세요"
	}
	if strings.Contains(contextLower, "claim") || strings.Contains(contextLower, "소유권") {
		return "매장 소유권 등록 중 오류가 발생했습니다. 잠시 후 다시 시도해주세요"
	}

	return "서버 오류가 발생했습니다. 잠시 후 다시 시도해주세요"
}

// ParseAndRespond 에러를 파싱하여 응답 반환 (헬퍼 함수)
// controller에서 간편하게 사용할 수 있도록
func ParseAndRespond(c interface{ JSON(int, interface{}) }, statusCode int, err error, context string) {
	errorInfo := ParseError(err, context)
	c.JSON(statusCode, ErrorResponse{
		Error:   errorInfo.Code,
		Message: errorInfo.Message,
	})
}
