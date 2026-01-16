package errors

// 에러 코드 상수 정의
// 형식: CATEGORY_SPECIFIC_DETAIL
// 프론트엔드에서 이 코드를 기반으로 메시지를 매핑함

const (
	// ==================== 인증 (AUTH_) ====================
	AuthUnauthorized        = "AUTH_UNAUTHORIZED"         // 로그인 필요
	AuthInvalidCredentials  = "AUTH_INVALID_CREDENTIALS"  // 잘못된 이메일/비밀번호
	AuthTokenExpired        = "AUTH_TOKEN_EXPIRED"        // 토큰 만료
	AuthTokenInvalid        = "AUTH_TOKEN_INVALID"        // 잘못된 토큰
	AuthTokenRevoked        = "AUTH_TOKEN_REVOKED"        // 토큰 폐기됨
	AuthEmailAlreadyExists  = "AUTH_EMAIL_EXISTS"         // 이메일 중복
	AuthNicknameExists      = "AUTH_NICKNAME_EXISTS"      // 닉네임 중복
	AuthPhoneNotVerified    = "AUTH_PHONE_NOT_VERIFIED"   // 휴대폰 미인증
	AuthEmailNotVerified    = "AUTH_EMAIL_NOT_VERIFIED"   // 이메일 미인증
	AuthCodeInvalid         = "AUTH_CODE_INVALID"         // 잘못된 인증코드
	AuthCodeExpired         = "AUTH_CODE_EXPIRED"         // 인증코드 만료
	AuthAlreadyVerified     = "AUTH_ALREADY_VERIFIED"     // 이미 인증됨

	// ==================== 인가/권한 (AUTHZ_) ====================
	AuthzForbidden        = "AUTHZ_FORBIDDEN"         // 접근 권한 없음
	AuthzAccessDenied     = "AUTHZ_ACCESS_DENIED"    // 작업 권한 없음
	AuthzRoleNotFound     = "AUTHZ_ROLE_NOT_FOUND"   // 권한 정보 없음
	AuthzAdminOnly        = "AUTHZ_ADMIN_ONLY"       // 관리자만 가능
	AuthzOwnerOnly        = "AUTHZ_OWNER_ONLY"       // 소유자만 가능

	// ==================== 검증 (VALIDATION_) ====================
	ValidationInvalidInput   = "VALIDATION_INVALID_INPUT"   // 잘못된 입력
	ValidationInvalidID      = "VALIDATION_INVALID_ID"      // 잘못된 ID
	ValidationInvalidFormat  = "VALIDATION_INVALID_FORMAT"  // 잘못된 형식
	ValidationInvalidRange   = "VALIDATION_INVALID_RANGE"   // 범위 초과
	ValidationTooShort       = "VALIDATION_TOO_SHORT"       // 너무 짧음
	ValidationTooLong        = "VALIDATION_TOO_LONG"        // 너무 길음
	ValidationRequired       = "VALIDATION_REQUIRED"        // 필수 항목

	// ==================== 리소스 (RESOURCE_) ====================
	ResourceNotFound       = "RESOURCE_NOT_FOUND"        // 리소스 없음
	ResourceAlreadyExists  = "RESOURCE_ALREADY_EXISTS"   // 이미 존재
	ResourceDeleted        = "RESOURCE_DELETED"          // 삭제됨
	ResourceConflict       = "RESOURCE_CONFLICT"         // 충돌

	// ==================== 매장 (STORE_) ====================
	StoreNotFound              = "STORE_NOT_FOUND"               // 매장 없음
	StoreAlreadyManaged        = "STORE_ALREADY_MANAGED"         // 이미 관리 중
	StoreAlreadyOwned          = "STORE_ALREADY_OWNED"           // 이미 소유 중
	StoreBusinessNumberExists  = "STORE_BUSINESS_NUMBER_EXISTS"  // 사업자번호 중복
	StoreVerificationFailed    = "STORE_VERIFICATION_FAILED"     // 사업자 인증 실패
	StoreVerificationPending   = "STORE_VERIFICATION_PENDING"    // 인증 심사 중
	StoreVerificationRejected  = "STORE_VERIFICATION_REJECTED"   // 인증 반려됨
	StoreAlreadyVerified       = "STORE_ALREADY_VERIFIED"        // 이미 인증됨

	// ==================== 리뷰 (REVIEW_) ====================
	ReviewNotFound         = "REVIEW_NOT_FOUND"          // 리뷰 없음
	ReviewInvalidRating    = "REVIEW_INVALID_RATING"     // 잘못된 평점
	ReviewTooShort         = "REVIEW_TOO_SHORT"          // 리뷰 너무 짧음
	ReviewAlreadyExists    = "REVIEW_ALREADY_EXISTS"     // 이미 리뷰 작성함

	// ==================== 게시글/댓글 (POST_) ====================
	PostNotFound           = "POST_NOT_FOUND"            // 게시글 없음
	PostDeleteFailed       = "POST_DELETE_FAILED"        // 삭제 실패
	PostEditFailed         = "POST_EDIT_FAILED"          // 수정 실패
	PostInvalidCategory    = "POST_INVALID_CATEGORY"     // 잘못된 카테고리
	CommentNotFound        = "COMMENT_NOT_FOUND"         // 댓글 없음
	CommentDeleteFailed    = "COMMENT_DELETE_FAILED"     // 댓글 삭제 실패

	// ==================== 채팅 (CHAT_) ====================
	ChatRoomNotFound       = "CHAT_ROOM_NOT_FOUND"       // 채팅방 없음
	ChatMessageNotFound    = "CHAT_MESSAGE_NOT_FOUND"    // 메시지 없음
	ChatCannotSendMessage  = "CHAT_CANNOT_SEND"          // 메시지 전송 불가
	ChatSelfRoomForbidden  = "CHAT_SELF_ROOM_FORBIDDEN"  // 자기 자신과 채팅 불가
	ChatMessageDeleted     = "CHAT_MESSAGE_DELETED"      // 메시지 이미 삭제됨
	ChatUpdateDeleted      = "CHAT_UPDATE_DELETED"       // 삭제된 메시지 수정 불가

	// ==================== 알림 (NOTIFICATION_) ====================
	NotificationNotFound   = "NOTIFICATION_NOT_FOUND"    // 알림 없음

	// ==================== 업로드 (UPLOAD_) ====================
	UploadInvalidFileType  = "UPLOAD_INVALID_FILE_TYPE"  // 잘못된 파일 형식
	UploadFileTooLarge     = "UPLOAD_FILE_TOO_LARGE"     // 파일 너무 큼
	UploadFailed           = "UPLOAD_FAILED"             // 업로드 실패

	// ==================== 금 시세 (GOLD_) ====================
	GoldPriceNotFound      = "GOLD_PRICE_NOT_FOUND"      // 시세 없음
	GoldInvalidType        = "GOLD_INVALID_TYPE"         // 잘못된 금 종류

	// ==================== 비즈니스 로직 (BUSINESS_) ====================
	BusinessStoreRequired      = "BUSINESS_STORE_REQUIRED"       // 매장 필요
	BusinessOneStorePerUser    = "BUSINESS_ONE_STORE_PER_USER"   // 한 계정당 하나만
	BusinessQnaOnly            = "BUSINESS_QNA_ONLY"             // QnA 전용 기능

	// ==================== 내부 오류 (INTERNAL_) ====================
	InternalServerError    = "INTERNAL_SERVER_ERROR"     // 서버 오류
	InternalDatabaseError  = "INTERNAL_DATABASE_ERROR"   // DB 오류
	InternalExternalAPI    = "INTERNAL_EXTERNAL_API"     // 외부 API 오류
	InternalConfigError    = "INTERNAL_CONFIG_ERROR"     // 설정 오류
)
