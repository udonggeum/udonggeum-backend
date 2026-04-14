package db

import (
	"math/rand"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

// Migrate runs database migrations
func Migrate() error {
	logger.Info("Running database migrations...")

	models := []interface{}{
		&model.User{},
		&model.PasswordReset{},
		&model.Store{},
		&model.BusinessRegistration{},
		&model.StoreVerification{},
		&model.GoldPrice{},
		&model.CommunityPost{},
		&model.CommunityComment{},
		&model.PostLike{},
		&model.CommentLike{},
		&model.StoreReview{},
		&model.ReviewLike{},
		&model.StoreLike{},
		&model.StoreRegistrationRequest{},
		&model.Tag{},
		&model.StoreTag{},
		&model.ChatRoom{},
		&model.Message{},
		&model.Notification{},
		&model.NotificationSettings{},
		&model.FAQ{},
	}

	if err := DB.AutoMigrate(models...); err != nil {
		logger.Error("Failed to run migrations", err)
		return err
	}

	if err := runCustomMigrations(); err != nil {
		logger.Error("Failed to run custom migrations", err)
		return err
	}

	if err := seedInitialData(); err != nil {
		logger.Error("Failed to seed initial data during migration", err)
		return err
	}

	logger.Info("Database migrations completed successfully", map[string]interface{}{
		"models_count": len(models),
	})
	return nil
}

// runCustomMigrations 이미 적용된 인덱스는 건너뛰고 필요한 것만 실행
func runCustomMigrations() error {
	type migration struct {
		name string
		sql  string
	}

	migrations := []migration{
		{
			name: "idx_stores_fts",
			sql: `CREATE INDEX IF NOT EXISTS idx_stores_fts ON stores USING GIN (
				to_tsvector(
					'simple',
					coalesce(name, '') || ' ' ||
					coalesce(region, '') || ' ' ||
					coalesce(district, '') || ' ' ||
					coalesce(dong, '') || ' ' ||
					coalesce(address, '')
				)
			)`,
		},
	}

	for _, m := range migrations {
		// pg_indexes로 이미 존재하는지 확인
		var count int64
		if err := DB.Raw("SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?", m.name).Scan(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			logger.Info("Custom migration already applied, skipping", map[string]interface{}{
				"migration": m.name,
			})
			continue
		}

		if err := DB.Exec(m.sql).Error; err != nil {
			logger.Error("Failed to apply custom migration", err, map[string]interface{}{
				"migration": m.name,
			})
			return err
		}

		logger.Info("Custom migration applied", map[string]interface{}{
			"migration": m.name,
		})
	}

	return nil
}

// Seed adds initial data to the database (optional)
func Seed() error {
	return seedInitialData()
}

func seedInitialData() error {
	logger.Info("Seeding initial data...")

	if err := seedTags(); err != nil {
		logger.Error("Failed to seed tags", err)
		return err
	}

	if err := seedFAQs(); err != nil {
		logger.Error("Failed to seed FAQs", err)
		return err
	}

	logger.Info("Initial data seeded successfully")
	return nil
}

func seedFAQs() error {
	var count int64
	if err := DB.Model(&model.FAQ{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		logger.Info("FAQs already seeded, skipping...")
		return nil
	}

	faqs := []model.FAQ{
		// 일반 사용자
		{Target: model.FAQTargetUser, SortOrder: 1, Question: "금 시세는 어떻게 결정되나요?", Answer: "금 시세는 국제 금 현물 가격(런던 금 시세)을 기준으로 환율과 국내 유통 마진을 반영해 결정됩니다. 우리동네금은방에서는 매일 업데이트되는 시세를 확인하실 수 있습니다."},
		{Target: model.FAQTargetUser, SortOrder: 2, Question: "순금(24K)과 18K, 14K의 차이가 뭔가요?", Answer: "K(캐럿)는 금의 순도를 나타냅니다.\n• 24K: 순금 (99.9% 이상)\n• 18K: 금 75% + 다른 금속 25%\n• 14K: 금 58.3% + 다른 금속 41.7%\n순도가 높을수록 가격이 높고 변색이 적습니다."},
		{Target: model.FAQTargetUser, SortOrder: 3, Question: "금을 팔 때 어떤 준비가 필요한가요?", Answer: "신분증(주민등록증 또는 운전면허증)을 지참하시면 됩니다. 금 판매는 실명 확인이 필요하며, 일부 매장에서는 구매 영수증이 있으면 더 유리한 가격을 받을 수 있습니다."},
		{Target: model.FAQTargetUser, SortOrder: 4, Question: "매장 리뷰는 누구나 작성할 수 있나요?", Answer: "로그인한 회원이라면 누구나 리뷰를 작성할 수 있습니다. 다만 허위 리뷰나 도배성 리뷰는 운영 정책에 따라 삭제될 수 있습니다."},
		{Target: model.FAQTargetUser, SortOrder: 5, Question: "금광산 게시판의 예약하기는 실제 계약인가요?", Answer: "예약하기는 거래 의사를 표시하는 기능으로, 법적 효력이 있는 계약은 아닙니다. 실제 거래는 판매자와 직접 연락하여 진행하시기 바랍니다."},
		{Target: model.FAQTargetUser, SortOrder: 6, Question: "사기 거래가 의심될 때 어떻게 하나요?", Answer: "게시글 내 신고 기능을 이용하거나, 1:1 문의를 통해 운영팀에 알려주세요. 빠르게 검토 후 조치하겠습니다. 금전 피해가 발생한 경우 경찰에 신고하시기 바랍니다."},
		{Target: model.FAQTargetUser, SortOrder: 7, Question: "탈퇴 후 재가입이 가능한가요?", Answer: "네, 탈퇴 후에도 동일한 이메일로 재가입이 가능합니다. 단, 탈퇴 시 작성한 게시글·댓글·리뷰 등의 데이터는 복구되지 않습니다."},
		// 금은방 사장님
		{Target: model.FAQTargetOwner, SortOrder: 1, Question: "매장 등록은 어떻게 하나요?", Answer: "상단 메뉴 또는 메인 페이지의 '매장 관리하기' 버튼을 눌러 내 매장을 검색하세요. 검색 후 소유권을 신청하거나, 목록에 없으면 직접 추가할 수 있습니다. 사업자등록번호 인증을 통해 자동으로 매장 관리자 권한이 부여됩니다."},
		{Target: model.FAQTargetOwner, SortOrder: 2, Question: "사업자 인증은 왜 필요한가요?", Answer: "플랫폼의 신뢰성을 위해 실제 사업자만 매장을 관리할 수 있도록 국세청 사업자등록번호 진위 확인을 거칩니다. 인증 완료 시 '인증된 매장' 뱃지가 표시되어 고객 신뢰도가 높아집니다."},
		{Target: model.FAQTargetOwner, SortOrder: 3, Question: "이미 다른 사람이 내 매장을 등록했어요.", Answer: "1:1 문의를 통해 사업자등록증 사본을 첨부하여 소유권 이전을 요청해 주세요. 확인 후 정당한 사업자에게 권한을 이전해 드립니다."},
		{Target: model.FAQTargetOwner, SortOrder: 4, Question: "매장 정보는 어떻게 수정하나요?", Answer: "로그인 후 '내 매장' 메뉴에서 영업시간, 주소, 전화번호, 매장 사진 등을 언제든지 수정할 수 있습니다."},
		{Target: model.FAQTargetOwner, SortOrder: 5, Question: "고객과 채팅은 어떻게 하나요?", Answer: "고객이 매장 상세 페이지에서 '채팅하기'를 누르면 채팅방이 생성됩니다. 상단 메뉴의 '메시지'에서 모든 채팅을 확인하고 답변할 수 있습니다."},
		{Target: model.FAQTargetOwner, SortOrder: 6, Question: "매장 사진은 몇 장까지 등록 가능한가요?", Answer: "현재 매장 대표 사진 1장을 등록할 수 있으며, 추후 갤러리 기능 업데이트를 통해 더 많은 사진을 등록할 수 있게 될 예정입니다."},
	}

	for i := range faqs {
		if err := DB.Create(&faqs[i]).Error; err != nil {
			return err
		}
	}

	logger.Info("FAQs seeded successfully", map[string]interface{}{"count": len(faqs)})
	return nil
}

// seedGoldPrices 금 시세 더미 데이터 생성 (최근 30일)
func seedGoldPrices() error {
	var count int64
	if err := DB.Model(&model.GoldPrice{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		logger.Info("Gold prices already seeded, skipping...", map[string]interface{}{
			"existing_count": count,
		})
		return nil
	}

	logger.Info("Seeding gold price data for last 30 days...")

	// 기준 시세 (원/g) - 최근 실제 시세 기준
	basePrice24K := 199611.37
	basePrice18K := 149708.53
	basePrice14K := 116439.97

	now := time.Now()
	totalInserted := 0

	// 최근 30일간 매일 시세 생성
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)

		// 날짜별로 약간의 변동성 추가 (-2% ~ +2%)
		variance := (rand.Float64() - 0.5) * 0.04 // -0.02 ~ +0.02

		price24K := basePrice24K * (1 + variance)
		price18K := basePrice18K * (1 + variance)
		price14K := basePrice14K * (1 + variance)

		goldPrices := []model.GoldPrice{
			{
				Type:        model.Gold24K,
				BuyPrice:    price24K * 0.98, // 매입가: 현재가의 98%
				SellPrice:   price24K * 1.02, // 매도가: 현재가의 102%
				Source:      "GOLDAPI",
				SourceDate:  date,
				Description: "자동 생성된 더미 데이터",
			},
			{
				Type:        model.Gold18K,
				BuyPrice:    price18K * 0.98,
				SellPrice:   price18K * 1.02,
				Source:      "GOLDAPI",
				SourceDate:  date,
				Description: "자동 생성된 더미 데이터",
			},
			{
				Type:        model.Gold14K,
				BuyPrice:    price14K * 0.98,
				SellPrice:   price14K * 1.02,
				Source:      "GOLDAPI",
				SourceDate:  date,
				Description: "자동 생성된 더미 데이터",
			},
		}

		for _, goldPrice := range goldPrices {
			if err := DB.Create(&goldPrice).Error; err != nil {
				logger.Error("Failed to create gold price", err)
				return err
			}
			totalInserted++
		}
	}

	logger.Info("Gold prices seeded successfully", map[string]interface{}{
		"total_records": totalInserted,
		"days":          30,
	})

	return nil
}

// seedTags 태그 데이터 생성
func seedTags() error {
	var count int64
	if err := DB.Model(&model.Tag{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		logger.Info("Tags already seeded, skipping...", map[string]interface{}{
			"existing_count": count,
		})
		return nil
	}

	logger.Info("Seeding tag data...")

	tags := []model.Tag{
		// 서비스 카테고리
		{Name: "24K 취급", Category: "서비스"},
		{Name: "18K 취급", Category: "서비스"},
		{Name: "14K 취급", Category: "서비스"},
		{Name: "금 매입", Category: "서비스"},
		{Name: "금 판매", Category: "서비스"},
		{Name: "수리가능", Category: "서비스"},
		{Name: "리폼", Category: "서비스"},
		{Name: "주얼리 제작", Category: "서비스"},

		// 상품 카테고리
		{Name: "다이아몬드", Category: "상품"},
		{Name: "백금", Category: "상품"},
		{Name: "은", Category: "상품"},
		{Name: "반지", Category: "상품"},
		{Name: "목걸이", Category: "상품"},
		{Name: "팔찌", Category: "상품"},

		// 특징 카테고리
		{Name: "친절한 상담", Category: "특징"},
		{Name: "빠른 매입", Category: "특징"},
		{Name: "현금 즉시 지급", Category: "특징"},
		{Name: "주차 가능", Category: "특징"},
		{Name: "오픈 30년 이상", Category: "특징"},
	}

	totalInserted := 0
	for _, tag := range tags {
		if err := DB.Create(&tag).Error; err != nil {
			logger.Error("Failed to create tag", err, map[string]interface{}{
				"tag": tag.Name,
			})
			return err
		}
		totalInserted++
	}

	// 매장에 태그 연결 (샘플 데이터)
	if err := seedStoreTags(); err != nil {
		logger.Error("Failed to seed store tags", err)
		return err
	}

	logger.Info("Tags seeded successfully", map[string]interface{}{
		"total_tags": totalInserted,
	})

	return nil
}

// seedStoreTags 매장-태그 연결 데이터 생성
func seedStoreTags() error {
	logger.Info("Seeding store tags...")

	// 모든 매장 조회
	var stores []model.Store
	if err := DB.Find(&stores).Error; err != nil {
		return err
	}

	// 모든 태그 조회
	var tags []model.Tag
	if err := DB.Find(&tags).Error; err != nil {
		return err
	}

	if len(stores) == 0 || len(tags) == 0 {
		logger.Info("No stores or tags found, skipping store tag seeding")
		return nil
	}

	// 각 매장에 랜덤으로 3-6개 태그 연결
	for _, store := range stores {
		// 랜덤 태그 개수 (3-6개)
		numTags := rand.Intn(4) + 3

		// 이미 할당된 태그 추적
		assignedTags := make(map[uint]bool)

		for i := 0; i < numTags && len(assignedTags) < len(tags); i++ {
			// 랜덤 태그 선택
			randomTag := tags[rand.Intn(len(tags))]

			// 중복 체크
			if assignedTags[randomTag.ID] {
				continue
			}

			// 매장-태그 연결
			storeTag := model.StoreTag{
				StoreID: store.ID,
				TagID:   randomTag.ID,
			}

			if err := DB.Create(&storeTag).Error; err != nil {
				logger.Error("Failed to create store tag", err, map[string]interface{}{
					"store_id": store.ID,
					"tag_id":   randomTag.ID,
				})
				return err
			}

			assignedTags[randomTag.ID] = true
		}
	}

	logger.Info("Store tags seeded successfully", map[string]interface{}{
		"stores_count": len(stores),
	})

	return nil
}
