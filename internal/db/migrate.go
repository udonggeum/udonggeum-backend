package db

import (
	"math/rand"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/ikkim/udonggeum-backend/pkg/util"
)

// Migrate runs database migrations
func Migrate() error {
	logger.Info("Running database migrations...")

	models := []interface{}{
		&model.Store{},
		&model.User{},
		&model.PasswordReset{},
		&model.GoldPrice{},
		&model.CommunityPost{},
		&model.CommunityComment{},
		&model.PostLike{},
		&model.CommentLike{},
		&model.StoreReview{},
		&model.ReviewLike{},
		&model.StoreLike{},
		&model.Tag{},
		&model.StoreTag{},
		&model.ChatRoom{},
		&model.Message{},
	}

	if err := DB.AutoMigrate(models...); err != nil {
		logger.Error("Failed to run migrations", err)
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

// Seed adds initial data to the database (optional)
func Seed() error {
	return seedInitialData()
}

func seedInitialData() error {
	logger.Info("Seeding database...")

	var storeCount int64
	if err := DB.Model(&model.Store{}).Count(&storeCount).Error; err != nil {
		logger.Error("Failed to check store count before seeding", err)
		return err
	}

	if storeCount > 0 {
		logger.Info("Store data already seeded, skipping store seeding...", map[string]interface{}{
			"existing_stores": storeCount,
		})
		// Store 데이터는 스킵하지만 금 시세는 별도로 체크
		if err := seedGoldPrices(); err != nil {
			logger.Error("Failed to seed gold prices", err)
			return err
		}
		return nil
	}

	adminPassword, err := util.HashPassword("test12!@")
	if err != nil {
		logger.Error("Failed to hash admin password", err)
		return err
	}

	admin := model.User{
		Email:        "test@test.com",
		PasswordHash: adminPassword,
		Name:         "관리자",
		Role:         model.RoleAdmin,
	}

	if err := DB.Where("email = ?", admin.Email).
		Attrs(admin).
		FirstOrCreate(&admin).Error; err != nil {
		logger.Error("Failed to seed admin user", err)
		return err
	}

	stores := []model.Store{
		{
			Name:           "서울 강남 금은방",
			Region:         "서울특별시",
			District:       "강남구",
			Address:        "서울특별시 강남구 테헤란로 231",
			Latitude:       func() *float64 { lat := 37.5029; return &lat }(),
			Longitude:      func() *float64 { lng := 127.0398; return &lng }(),
			PhoneNumber:    "02-6201-1100",
			ImageURL:       "https://cdn.udonggeum.com/stores/seoul-gangnam.jpg",
			Description:    "프리미엄 금 매입 전문점",
			BuyingGold:     true,
			BuyingPlatinum: true,
			BuyingSilver:   true,
		},
		{
			Name:           "서울 마포 금은방",
			Region:         "서울특별시",
			District:       "마포구",
			Address:        "서울특별시 마포구 와우산로 94",
			Latitude:       func() *float64 { lat := 37.5490; return &lat }(),
			Longitude:      func() *float64 { lng := 126.9251; return &lng }(),
			PhoneNumber:    "02-6284-9077",
			ImageURL:       "https://cdn.udonggeum.com/stores/seoul-mapo.jpg",
			Description:    "지역 밀착형 금 매입 전문점",
			BuyingGold:     true,
			BuyingPlatinum: false,
			BuyingSilver:   true,
		},
		{
			Name:           "부산 해운대 금은방",
			Region:         "부산광역시",
			District:       "해운대구",
			Address:        "부산광역시 해운대구 해운대로 570",
			Latitude:       func() *float64 { lat := 35.1586; return &lat }(),
			Longitude:      func() *float64 { lng := 129.1603; return &lng }(),
			PhoneNumber:    "051-730-1122",
			ImageURL:       "https://cdn.udonggeum.com/stores/busan-haeundae.jpg",
			Description:    "부산 최대 규모 금 매입 전문점",
			BuyingGold:     true,
			BuyingPlatinum: true,
			BuyingSilver:   true,
		},
		{
			Name:           "부산 남포 금은방",
			Region:         "부산광역시",
			District:       "중구",
			Address:        "부산광역시 중구 광복로 55",
			Latitude:       func() *float64 { lat := 35.0985; return &lat }(),
			Longitude:      func() *float64 { lng := 129.0297; return &lng }(),
			PhoneNumber:    "051-245-7755",
			ImageURL:       "https://cdn.udonggeum.com/stores/busan-nampo.jpg",
			Description:    "전통 금 매입 전문점",
			BuyingGold:     true,
			BuyingPlatinum: true,
			BuyingSilver:   false,
		},
		{
			Name:           "대구 동성로 금은방",
			Region:         "대구광역시",
			District:       "중구",
			Address:        "대구광역시 중구 동성로4길 91",
			Latitude:       func() *float64 { lat := 35.8691; return &lat }(),
			Longitude:      func() *float64 { lng := 128.5936; return &lng }(),
			PhoneNumber:    "053-222-4411",
			ImageURL:       "https://cdn.udonggeum.com/stores/daegu-dongseongno.jpg",
			Description:    "대구 중심가 금 매입 전문점",
			BuyingGold:     true,
			BuyingPlatinum: false,
			BuyingSilver:   true,
		},
		{
			Name:           "광주 충장로 금은방",
			Region:         "광주광역시",
			District:       "동구",
			Address:        "광주광역시 동구 충장로 73",
			Latitude:       func() *float64 { lat := 35.1483; return &lat }(),
			Longitude:      func() *float64 { lng := 126.9174; return &lng }(),
			PhoneNumber:    "062-228-9090",
			ImageURL:       "https://cdn.udonggeum.com/stores/gwangju-chungjang.jpg",
			Description:    "광주 대표 금 매입 전문점",
			BuyingGold:     true,
			BuyingPlatinum: true,
			BuyingSilver:   true,
		},
		{
			Name:           "제주 신제주 금은방",
			Region:         "제주특별자치도",
			District:       "제주시",
			Address:        "제주특별자치도 제주시 연북로 567",
			Latitude:       func() *float64 { lat := 33.4890; return &lat }(),
			Longitude:      func() *float64 { lng := 126.4914; return &lng }(),
			PhoneNumber:    "064-723-5565",
			ImageURL:       "https://cdn.udonggeum.com/stores/jeju-shinjeju.jpg",
			Description:    "제주 유일 금 매입 전문점",
			BuyingGold:     true,
			BuyingPlatinum: false,
			BuyingSilver:   true,
		},
	}

	totalStores := 0

	for _, store := range stores {
		store.UserID = admin.ID

		var createdStore model.Store
		if err := DB.Where("name = ?", store.Name).
			Attrs(store).
			FirstOrCreate(&createdStore).Error; err != nil {
			logger.Error("Failed to seed store", err, map[string]interface{}{
				"store": store.Name,
			})
			return err
		}
		totalStores++
	}

	// 금 시세 더미 데이터 생성 (최근 30일)
	if err := seedGoldPrices(); err != nil {
		logger.Error("Failed to seed gold prices", err)
		return err
	}

	// 태그 데이터 생성
	if err := seedTags(); err != nil {
		logger.Error("Failed to seed tags", err)
		return err
	}

	logger.Info("Database seeded successfully", map[string]interface{}{
		"stores_count": totalStores,
	})
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
