package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// PostCategory 게시글 카테고리 타입
type PostCategory string

const (
	CategoryGoldTrade PostCategory = "gold_trade" // 금거래
	CategoryGoldNews  PostCategory = "gold_news"  // 금소식
	CategoryQnA       PostCategory = "qna"        // QnA
)

// PostType 게시글 타입 (금거래 세부 분류)
type PostType string

const (
	// 금거래 - 일반 사용자 (금 매수/판매)
	TypeSellGold PostType = "sell_gold" // 금 매수(내 금 팔기)

	// 금거래 - 금은방 사장님 (금 매입)
	TypeBuyGold PostType = "buy_gold" // 금 매입 홍보

	// 금소식
	TypeProductNews PostType = "product_news" // 상품 소식
	TypeStoreNews   PostType = "store_news"   // 매장 소식
	TypeOther       PostType = "other"        // 기타

	// QnA
	TypeQuestion PostType = "question" // 질문
	TypeFAQ      PostType = "faq"      // FAQ (관리자만 작성)
)

// PostStatus 게시글 상태
type PostStatus string

const (
	StatusActive   PostStatus = "active"   // 활성
	StatusInactive PostStatus = "inactive" // 비활성 (작성자 숨김)
	StatusDeleted  PostStatus = "deleted"  // 삭제됨
	StatusReported PostStatus = "reported" // 신고됨 (관리자 검토 필요)
)

// CommunityPost 커뮤니티 게시글 모델
type CommunityPost struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 게시글 기본 정보
	Title    string       `gorm:"type:varchar(200);not null" json:"title"`      // 제목
	Content  string       `gorm:"type:text;not null" json:"content"`            // 내용
	Category PostCategory `gorm:"type:varchar(20);not null" json:"category"`    // 카테고리
	Type     PostType     `gorm:"type:varchar(20);not null" json:"type"`        // 게시글 타입
	Status   PostStatus   `gorm:"type:varchar(20);default:'active'" json:"status"` // 상태

	// 작성자 정보
	UserID uint `gorm:"not null;index" json:"user_id"` // 작성자 ID
	User   User `gorm:"foreignKey:UserID" json:"user"` // 작성자 정보

	// 금거래 관련 필드 (금거래 카테고리일 때만 사용)
	GoldType    *string  `gorm:"type:varchar(50)" json:"gold_type,omitempty"`    // 금 종류 (24K, 18K, 14K 등)
	Weight      *float64 `json:"weight,omitempty"`                               // 중량 (g)
	Price       *int64   `json:"price,omitempty"`                                // 희망가격/매입가격 (원)
	Location    *string  `gorm:"type:varchar(100)" json:"location,omitempty"`    // 거래 희망 지역
	Region      *string  `gorm:"type:varchar(50)" json:"region,omitempty"`       // 시/도 (알림 필터링용)
	District    *string  `gorm:"type:varchar(50)" json:"district,omitempty"`     // 시/군/구 (알림 필터링용)
	StoreID     *uint    `gorm:"index" json:"store_id,omitempty"`                // 매장 ID (사장님 글일 때)
	Store       *Store   `gorm:"foreignKey:StoreID" json:"store,omitempty"`      // 매장 정보

	// 예약 및 거래 완료 관련 필드 (금거래만)
	ReservationStatus *string    `gorm:"type:varchar(20)" json:"reservation_status,omitempty"` // 예약 상태 (null=판매중, reserved=예약중, completed=거래완료)
	ReservedByUserID  *uint      `gorm:"index" json:"reserved_by_user_id,omitempty"`          // 예약한 사용자 ID
	ReservedByUser    *User      `gorm:"foreignKey:ReservedByUserID" json:"reserved_by_user,omitempty"` // 예약한 사용자 정보
	ReservedAt        *time.Time `json:"reserved_at,omitempty"`                               // 예약 시간
	CompletedAt       *time.Time `json:"completed_at,omitempty"`                              // 거래 완료 시간

	// QnA 관련 필드
	IsAnswered   bool  `gorm:"default:false" json:"is_answered"`              // 답변 완료 여부
	AcceptedAnswerID *uint `gorm:"index" json:"accepted_answer_id,omitempty"` // 채택된 답변 ID

	// 매장 게시글 관리
	IsPinned bool `gorm:"default:false;index" json:"is_pinned"` // 매장 페이지 상단 고정 여부

	// 통계
	ViewCount    int `gorm:"default:0" json:"view_count"`    // 조회수
	LikeCount    int `gorm:"default:0" json:"like_count"`    // 좋아요 수
	CommentCount int `gorm:"default:0" json:"comment_count"` // 댓글 수

	// 이미지
	ImageURLs pq.StringArray `gorm:"type:text[]" json:"image_urls,omitempty"` // 이미지 URL 배열

	// 관계
	Comments []CommunityComment `gorm:"foreignKey:PostID" json:"comments,omitempty"` // 댓글 목록
	Likes    []PostLike         `gorm:"foreignKey:PostID" json:"-"`                  // 좋아요 목록
}

func (CommunityPost) TableName() string {
	return "community_posts"
}

// PostLike 게시글 좋아요 모델
type PostLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	PostID uint `gorm:"not null;index:idx_post_user_like,unique" json:"post_id"` // 게시글 ID
	UserID uint `gorm:"not null;index:idx_post_user_like,unique" json:"user_id"` // 사용자 ID

	Post CommunityPost `gorm:"foreignKey:PostID" json:"-"`
	User User          `gorm:"foreignKey:UserID" json:"-"`
}

func (PostLike) TableName() string {
	return "post_likes"
}

// CreatePostRequest 게시글 생성 요청
type CreatePostRequest struct {
	Title    string       `json:"title" binding:"required,min=2,max=200"`
	Content  string       `json:"content" binding:"required,min=10"`
	Category PostCategory `json:"category" binding:"required,oneof=gold_trade gold_news qna"`
	Type     PostType     `json:"type" binding:"required"`

	// 금거래 옵션
	GoldType *string  `json:"gold_type,omitempty"`
	Weight   *float64 `json:"weight,omitempty"`
	Price    *int64   `json:"price,omitempty"`
	Location *string  `json:"location,omitempty"`
	Region   *string  `json:"region,omitempty"`   // 시/도
	District *string  `json:"district,omitempty"` // 시/군/구
	// StoreID는 사용자 입력으로 받지 않음 (보안 이슈)
	// buy_gold 타입일 때 백엔드에서 자동으로 사용자의 매장 ID를 설정

	// 이미지
	ImageURLs []string `json:"image_urls,omitempty"`
}

// UpdatePostRequest 게시글 수정 요청
type UpdatePostRequest struct {
	Title     *string    `json:"title,omitempty" binding:"omitempty,min=2,max=200"`
	Content   *string    `json:"content,omitempty" binding:"omitempty,min=10"`
	Status    *PostStatus `json:"status,omitempty" binding:"omitempty,oneof=active inactive"`
	GoldType  *string    `json:"gold_type,omitempty"`
	Weight    *float64   `json:"weight,omitempty"`
	Price     *int64     `json:"price,omitempty"`
	Location  *string    `json:"location,omitempty"`
	ImageURLs []string   `json:"image_urls,omitempty"`
}

// PostListQuery 게시글 목록 조회 쿼리
type PostListQuery struct {
	Category  *PostCategory `form:"category"`
	Type      *PostType     `form:"type"`
	Status    *PostStatus   `form:"status"`
	UserID    *uint         `form:"user_id"`
	StoreID   *uint         `form:"store_id"`
	IsAnswered *bool        `form:"is_answered"`
	Search    *string       `form:"search"` // 제목+내용 검색
	Page      int           `form:"page" binding:"min=1"`
	PageSize  int           `form:"page_size" binding:"min=1,max=100"`
	SortBy    string        `form:"sort_by" binding:"omitempty,oneof=created_at view_count like_count comment_count"`
	SortOrder string        `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// GenerateContentRequest AI 컨텐츠 생성 요청
type GenerateContentRequest struct {
	Type     PostType `json:"type" binding:"required"`
	Keywords []string `json:"keywords" binding:"required"`
	Title    *string  `json:"title,omitempty"`
	GoldType *string  `json:"gold_type,omitempty"`
	Weight   *float64 `json:"weight,omitempty"`
	Price    *int64   `json:"price,omitempty"`
	Location *string  `json:"location,omitempty"`
}

// GenerateContentResponse AI 컨텐츠 생성 응답
type GenerateContentResponse struct {
	Content     string `json:"content"`
	GeneratedAt string `json:"generated_at,omitempty"`
}
