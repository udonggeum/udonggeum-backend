package db

import (
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
		&model.Product{},
		&model.ProductOption{},
		&model.Order{},
		&model.OrderItem{},
		&model.CartItem{},
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
		logger.Info("Database already seeded, skipping...", map[string]interface{}{
			"existing_stores": storeCount,
		})
		return nil
	}

	adminPassword, err := util.HashPassword("password123!")
	if err != nil {
		logger.Error("Failed to hash admin password", err)
		return err
	}

	admin := model.User{
		Email:        "admin@example.com",
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

	type productSeed struct {
		Product model.Product
		Options []model.ProductOption
	}

	stores := []struct {
		Store    model.Store
		Products []productSeed
	}{
		{
			Store: model.Store{
				Name:        "서울 강남 프리미엄점",
				Region:      "서울특별시",
				District:    "강남구",
				Address:     "서울특별시 강남구 테헤란로 231",
				PhoneNumber: "02-6201-1100",
				ImageURL:    "https://cdn.udonggeum.com/stores/seoul-gangnam.jpg",
				Description: "프리미엄 골드와 다이아몬드 라인을 전문으로 소개하는 플래그십 스토어",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "24K 클래식 골드 반지",
						Description:     "강남 프리미엄 라인을 대표하는 24K 순금 반지",
						Price:           980000,
						Weight:          7.2,
						Purity:          "24K",
						Category:        model.CategoryRing,
						Material:        model.MaterialGold,
						StockQuantity:   12,
						ImageURL:        "https://cdn.udonggeum.com/products/24k-classic-ring.jpg",
						PopularityScore: 92,
					},
					Options: []model.ProductOption{
						{
							Name:            "사이즈",
							Value:           "11호",
							AdditionalPrice: 0,
							StockQuantity:   4,
							IsDefault:       true,
						},
						{
							Name:            "사이즈",
							Value:           "13호",
							AdditionalPrice: 25000,
							StockQuantity:   4,
						},
						{
							Name:            "사이즈",
							Value:           "15호",
							AdditionalPrice: 40000,
							StockQuantity:   4,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "18K 로즈골드 체인 팔찌",
						Description:     "은은한 로즈골드 컬러의 섬세한 체인 팔찌",
						Price:           420000,
						Weight:          5.1,
						Purity:          "18K",
						Category:        model.CategoryBracelet,
						Material:        model.MaterialGold,
						StockQuantity:   15,
						ImageURL:        "https://cdn.udonggeum.com/products/18k-rose-bracelet.jpg",
						PopularityScore: 88,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "16cm",
							AdditionalPrice: 0,
							StockQuantity:   5,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "18cm",
							AdditionalPrice: 15000,
							StockQuantity:   5,
						},
						{
							Name:            "길이",
							Value:           "20cm",
							AdditionalPrice: 25000,
							StockQuantity:   5,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "다이아몬드 펜던트 목걸이",
						Description:     "라운드 브릴리언트 컷 다이아몬드를 세팅한 18K 목걸이",
						Price:           1250000,
						Weight:          3.4,
						Purity:          "18K",
						Category:        model.CategoryNecklace,
						Material:        model.MaterialGold,
						StockQuantity:   9,
						ImageURL:        "https://cdn.udonggeum.com/products/diamond-pendant-necklace.jpg",
						PopularityScore: 95,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "40cm",
							AdditionalPrice: 0,
							StockQuantity:   3,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "45cm",
							AdditionalPrice: 30000,
							StockQuantity:   3,
						},
						{
							Name:            "길이",
							Value:           "50cm",
							AdditionalPrice: 60000,
							StockQuantity:   3,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "서울 마포 라이프스타일점",
				Region:      "서울특별시",
				District:    "마포구",
				Address:     "서울특별시 마포구 와우산로 94",
				PhoneNumber: "02-6284-9077",
				ImageURL:    "https://cdn.udonggeum.com/stores/seoul-mapogu.jpg",
				Description: "데일리 주얼리와 커플 라인을 중심으로 구성된 라이프스타일 매장",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "925 실버 커브 팔찌",
						Description:     "데일리로 착용하기 좋은 커브 체인 디자인의 실버 팔찌",
						Price:           98000,
						Weight:          4.8,
						Purity:          "925",
						Category:        model.CategoryBracelet,
						Material:        model.MaterialSilver,
						StockQuantity:   18,
						ImageURL:        "https://cdn.udonggeum.com/products/925-curve-bracelet.jpg",
						PopularityScore: 84,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "17cm",
							AdditionalPrice: 0,
							StockQuantity:   6,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "19cm",
							AdditionalPrice: 7000,
							StockQuantity:   6,
						},
						{
							Name:            "길이",
							Value:           "21cm",
							AdditionalPrice: 12000,
							StockQuantity:   6,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "실버 오벌 스터드 귀걸이",
						Description:     "은은한 광택의 타원형 실버 스터드 귀걸이",
						Price:           65000,
						Weight:          2.1,
						Purity:          "925",
						Category:        model.CategoryEarring,
						Material:        model.MaterialSilver,
						StockQuantity:   24,
						ImageURL:        "https://cdn.udonggeum.com/products/silver-oval-stud.jpg",
						PopularityScore: 79,
					},
					Options: []model.ProductOption{
						{
							Name:            "마감",
							Value:           "실버",
							AdditionalPrice: 0,
							StockQuantity:   8,
							IsDefault:       true,
						},
						{
							Name:            "마감",
							Value:           "옐로우골드 도금",
							AdditionalPrice: 9000,
							StockQuantity:   8,
						},
						{
							Name:            "마감",
							Value:           "로즈골드 도금",
							AdditionalPrice: 9000,
							StockQuantity:   8,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "부산 해운대 시그니처점",
				Region:      "부산광역시",
				District:    "해운대구",
				Address:     "부산광역시 해운대구 해운대로 570",
				PhoneNumber: "051-730-1122",
				ImageURL:    "https://cdn.udonggeum.com/stores/busan-haeundae.jpg",
				Description: "해변 감성을 담은 프리미엄 주얼리와 웨딩 라인 전문점",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "18K 골드 웨이브 반지",
						Description:     "파도에서 영감을 받은 웨이브 라인의 골드 반지",
						Price:           720000,
						Weight:          5.6,
						Purity:          "18K",
						Category:        model.CategoryRing,
						Material:        model.MaterialGold,
						StockQuantity:   15,
						ImageURL:        "https://cdn.udonggeum.com/products/18k-wave-ring.jpg",
						PopularityScore: 86,
					},
					Options: []model.ProductOption{
						{
							Name:            "사이즈",
							Value:           "9호",
							AdditionalPrice: 0,
							StockQuantity:   5,
							IsDefault:       true,
						},
						{
							Name:            "사이즈",
							Value:           "11호",
							AdditionalPrice: 20000,
							StockQuantity:   5,
						},
						{
							Name:            "사이즈",
							Value:           "13호",
							AdditionalPrice: 35000,
							StockQuantity:   5,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "14K 미니멀 드롭 귀걸이",
						Description:     "섬세한 드롭 형태의 14K 미니멀 귀걸이",
						Price:           210000,
						Weight:          2.4,
						Purity:          "14K",
						Category:        model.CategoryEarring,
						Material:        model.MaterialGold,
						StockQuantity:   21,
						ImageURL:        "https://cdn.udonggeum.com/products/14k-drop-earring.jpg",
						PopularityScore: 82,
					},
					Options: []model.ProductOption{
						{
							Name:            "컬러",
							Value:           "옐로우골드",
							AdditionalPrice: 0,
							StockQuantity:   7,
							IsDefault:       true,
						},
						{
							Name:            "컬러",
							Value:           "로즈골드",
							AdditionalPrice: 12000,
							StockQuantity:   7,
						},
						{
							Name:            "컬러",
							Value:           "화이트골드",
							AdditionalPrice: 15000,
							StockQuantity:   7,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "실버 조개 펜던트 목걸이",
						Description:     "제철 조개에서 영감을 받은 해운대 감성 목걸이",
						Price:           89000,
						Weight:          3.2,
						Purity:          "925",
						Category:        model.CategoryNecklace,
						Material:        model.MaterialSilver,
						StockQuantity:   18,
						ImageURL:        "https://cdn.udonggeum.com/products/silver-shell-necklace.jpg",
						PopularityScore: 78,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "42cm",
							AdditionalPrice: 0,
							StockQuantity:   6,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "45cm",
							AdditionalPrice: 6000,
							StockQuantity:   6,
						},
						{
							Name:            "길이",
							Value:           "50cm",
							AdditionalPrice: 12000,
							StockQuantity:   6,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "부산 남포 전통점",
				Region:      "부산광역시",
				District:    "중구",
				Address:     "부산광역시 중구 광복로 55",
				PhoneNumber: "051-245-7755",
				ImageURL:    "https://cdn.udonggeum.com/stores/busan-nampo.jpg",
				Description: "전통 금장과 예물 세트에 특화된 남포동 대표 매장",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "순금 용 문양 팔찌",
						Description:     "정교한 용 문양이 새겨진 순금 팔찌",
						Price:           1350000,
						Weight:          10.5,
						Purity:          "24K",
						Category:        model.CategoryBracelet,
						Material:        model.MaterialGold,
						StockQuantity:   9,
						ImageURL:        "https://cdn.udonggeum.com/products/dragon-gold-bracelet.jpg",
						PopularityScore: 93,
					},
					Options: []model.ProductOption{
						{
							Name:            "중량",
							Value:           "5돈",
							AdditionalPrice: 0,
							StockQuantity:   3,
							IsDefault:       true,
						},
						{
							Name:            "중량",
							Value:           "7돈",
							AdditionalPrice: 180000,
							StockQuantity:   3,
						},
						{
							Name:            "중량",
							Value:           "10돈",
							AdditionalPrice: 420000,
							StockQuantity:   3,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "925 실버 커플 페어링",
						Description:     "심플한 라인의 커플 실버 반지 세트",
						Price:           158000,
						Weight:          6.0,
						Purity:          "925",
						Category:        model.CategoryRing,
						Material:        model.MaterialSilver,
						StockQuantity:   12,
						ImageURL:        "https://cdn.udonggeum.com/products/925-couple-ring-set.jpg",
						PopularityScore: 85,
					},
					Options: []model.ProductOption{
						{
							Name:            "세트 사이즈",
							Value:           "남자 17호 / 여자 13호",
							AdditionalPrice: 0,
							StockQuantity:   4,
							IsDefault:       true,
						},
						{
							Name:            "세트 사이즈",
							Value:           "남자 19호 / 여자 15호",
							AdditionalPrice: 8000,
							StockQuantity:   4,
						},
						{
							Name:            "세트 사이즈",
							Value:           "남자 21호 / 여자 17호",
							AdditionalPrice: 15000,
							StockQuantity:   4,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "대구 동성로 트렌드점",
				Region:      "대구광역시",
				District:    "중구",
				Address:     "대구광역시 중구 동성로4길 91",
				PhoneNumber: "053-222-4411",
				ImageURL:    "https://cdn.udonggeum.com/stores/daegu-dongseongno.jpg",
				Description: "최신 트렌드 주얼리를 빠르게 소개하는 젊은 감성 매장",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "14K 옐로우골드 레이어드 목걸이",
						Description:     "레이어드 연출이 쉬운 14K 옐로우골드 목걸이",
						Price:           398000,
						Weight:          3.9,
						Purity:          "14K",
						Category:        model.CategoryNecklace,
						Material:        model.MaterialGold,
						StockQuantity:   15,
						ImageURL:        "https://cdn.udonggeum.com/products/14k-layered-necklace.jpg",
						PopularityScore: 83,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "38cm",
							AdditionalPrice: 0,
							StockQuantity:   5,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "42cm",
							AdditionalPrice: 12000,
							StockQuantity:   5,
						},
						{
							Name:            "길이",
							Value:           "45cm",
							AdditionalPrice: 18000,
							StockQuantity:   5,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "실버 하프라인 반지",
						Description:     "반원 형태의 라인이 돋보이는 실버 반지",
						Price:           72000,
						Weight:          4.1,
						Purity:          "925",
						Category:        model.CategoryRing,
						Material:        model.MaterialSilver,
						StockQuantity:   18,
						ImageURL:        "https://cdn.udonggeum.com/products/silver-halfline-ring.jpg",
						PopularityScore: 77,
					},
					Options: []model.ProductOption{
						{
							Name:            "사이즈",
							Value:           "10호",
							AdditionalPrice: 0,
							StockQuantity:   6,
							IsDefault:       true,
						},
						{
							Name:            "사이즈",
							Value:           "12호",
							AdditionalPrice: 5000,
							StockQuantity:   6,
						},
						{
							Name:            "사이즈",
							Value:           "14호",
							AdditionalPrice: 9000,
							StockQuantity:   6,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "대구 수성 클래식점",
				Region:      "대구광역시",
				District:    "수성구",
				Address:     "대구광역시 수성구 달구벌대로 2440",
				PhoneNumber: "053-791-5599",
				ImageURL:    "https://cdn.udonggeum.com/stores/daegu-suseong.jpg",
				Description: "클래식 라인과 프리미엄 소재를 중심으로 구성된 컬렉션",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "18K 클래식 진주 귀걸이",
						Description:     "고급 담수 진주를 세팅한 18K 클래식 귀걸이",
						Price:           268000,
						Weight:          2.9,
						Purity:          "18K",
						Category:        model.CategoryEarring,
						Material:        model.MaterialGold,
						StockQuantity:   18,
						ImageURL:        "https://cdn.udonggeum.com/products/18k-classic-pearl-earring.jpg",
						PopularityScore: 89,
					},
					Options: []model.ProductOption{
						{
							Name:            "진주 크기",
							Value:           "6mm",
							AdditionalPrice: 0,
							StockQuantity:   6,
							IsDefault:       true,
						},
						{
							Name:            "진주 크기",
							Value:           "8mm",
							AdditionalPrice: 17000,
							StockQuantity:   6,
						},
						{
							Name:            "진주 크기",
							Value:           "10mm",
							AdditionalPrice: 34000,
							StockQuantity:   6,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "24K 승리의 반지",
						Description:     "승리를 상징하는 엠블럼을 새긴 24K 반지",
						Price:           1150000,
						Weight:          8.4,
						Purity:          "24K",
						Category:        model.CategoryRing,
						Material:        model.MaterialGold,
						StockQuantity:   12,
						ImageURL:        "https://cdn.udonggeum.com/products/24k-victory-ring.jpg",
						PopularityScore: 94,
					},
					Options: []model.ProductOption{
						{
							Name:            "사이즈",
							Value:           "13호",
							AdditionalPrice: 0,
							StockQuantity:   4,
							IsDefault:       true,
						},
						{
							Name:            "사이즈",
							Value:           "15호",
							AdditionalPrice: 28000,
							StockQuantity:   4,
						},
						{
							Name:            "사이즈",
							Value:           "17호",
							AdditionalPrice: 42000,
							StockQuantity:   4,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "가죽 & 골드 믹스 팔찌",
						Description:     "천연 가죽과 골드 포인트를 조합한 믹스 팔찌",
						Price:           189000,
						Weight:          9.8,
						Purity:          "가죽/18K",
						Category:        model.CategoryBracelet,
						Material:        model.MaterialOther,
						StockQuantity:   21,
						ImageURL:        "https://cdn.udonggeum.com/products/leather-gold-mix-bracelet.jpg",
						PopularityScore: 81,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "18cm",
							AdditionalPrice: 0,
							StockQuantity:   7,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "20cm",
							AdditionalPrice: 8000,
							StockQuantity:   7,
						},
						{
							Name:            "길이",
							Value:           "22cm",
							AdditionalPrice: 15000,
							StockQuantity:   7,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "광주 충장 예술점",
				Region:      "광주광역시",
				District:    "동구",
				Address:     "광주광역시 동구 충장로 73",
				PhoneNumber: "062-228-9090",
				ImageURL:    "https://cdn.udonggeum.com/stores/gwangju-chungjang.jpg",
				Description: "작가 협업 컬렉션과 예술적 감성의 주얼리를 선보이는 매장",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "실버 타이다이 목걸이",
						Description:     "수공예 타이다이 패턴을 적용한 실버 펜던트 목걸이",
						Price:           115000,
						Weight:          4.5,
						Purity:          "925",
						Category:        model.CategoryNecklace,
						Material:        model.MaterialSilver,
						StockQuantity:   18,
						ImageURL:        "https://cdn.udonggeum.com/products/silver-tiedye-necklace.jpg",
						PopularityScore: 76,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "40cm",
							AdditionalPrice: 0,
							StockQuantity:   6,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "45cm",
							AdditionalPrice: 6000,
							StockQuantity:   6,
						},
						{
							Name:            "길이",
							Value:           "50cm",
							AdditionalPrice: 12000,
							StockQuantity:   6,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "14K 컬러 큐빅 반지",
						Description:     "다채로운 컬러 큐빅을 세팅한 포인트 반지",
						Price:           248000,
						Weight:          3.3,
						Purity:          "14K",
						Category:        model.CategoryRing,
						Material:        model.MaterialGold,
						StockQuantity:   15,
						ImageURL:        "https://cdn.udonggeum.com/products/14k-color-cubic-ring.jpg",
						PopularityScore: 80,
					},
					Options: []model.ProductOption{
						{
							Name:            "컬러",
							Value:           "블루",
							AdditionalPrice: 0,
							StockQuantity:   5,
							IsDefault:       true,
						},
						{
							Name:            "컬러",
							Value:           "그린",
							AdditionalPrice: 10000,
							StockQuantity:   5,
						},
						{
							Name:            "컬러",
							Value:           "핑크",
							AdditionalPrice: 10000,
							StockQuantity:   5,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "광주 상무 비즈니스점",
				Region:      "광주광역시",
				District:    "서구",
				Address:     "광주광역시 서구 상무자유로 173",
				PhoneNumber: "062-415-3322",
				ImageURL:    "https://cdn.udonggeum.com/stores/gwangju-sangmu.jpg",
				Description: "비즈니스 캐주얼에 어울리는 주얼리를 제안하는 매장",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "18K 라운드 커프 팔찌",
						Description:     "부드러운 곡선으로 마감된 라운드 커프 팔찌",
						Price:           560000,
						Weight:          6.2,
						Purity:          "18K",
						Category:        model.CategoryBracelet,
						Material:        model.MaterialGold,
						StockQuantity:   12,
						ImageURL:        "https://cdn.udonggeum.com/products/18k-round-cuff-bracelet.jpg",
						PopularityScore: 85,
					},
					Options: []model.ProductOption{
						{
							Name:            "사이즈",
							Value:           "S",
							AdditionalPrice: 0,
							StockQuantity:   4,
							IsDefault:       true,
						},
						{
							Name:            "사이즈",
							Value:           "M",
							AdditionalPrice: 18000,
							StockQuantity:   4,
						},
						{
							Name:            "사이즈",
							Value:           "L",
							AdditionalPrice: 32000,
							StockQuantity:   4,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "티타늄 데일리 귀걸이",
						Description:     "알레르기 걱정 없는 티타늄 소재의 데일리 귀걸이",
						Price:           42000,
						Weight:          1.8,
						Purity:          "티타늄",
						Category:        model.CategoryEarring,
						Material:        model.MaterialOther,
						StockQuantity:   30,
						ImageURL:        "https://cdn.udonggeum.com/products/titanium-daily-earring.jpg",
						PopularityScore: 72,
					},
					Options: []model.ProductOption{
						{
							Name:            "마감",
							Value:           "브러쉬드",
							AdditionalPrice: 0,
							StockQuantity:   10,
							IsDefault:       true,
						},
						{
							Name:            "마감",
							Value:           "미러",
							AdditionalPrice: 5000,
							StockQuantity:   10,
						},
						{
							Name:            "마감",
							Value:           "샌드",
							AdditionalPrice: 7000,
							StockQuantity:   10,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "925 실버 타원 반지",
						Description:     "타원형 라인이 돋보이는 두께감 있는 실버 반지",
						Price:           83000,
						Weight:          5.2,
						Purity:          "925",
						Category:        model.CategoryRing,
						Material:        model.MaterialSilver,
						StockQuantity:   21,
						ImageURL:        "https://cdn.udonggeum.com/products/925-oval-ring.jpg",
						PopularityScore: 78,
					},
					Options: []model.ProductOption{
						{
							Name:            "사이즈",
							Value:           "11호",
							AdditionalPrice: 0,
							StockQuantity:   7,
							IsDefault:       true,
						},
						{
							Name:            "사이즈",
							Value:           "13호",
							AdditionalPrice: 6000,
							StockQuantity:   7,
						},
						{
							Name:            "사이즈",
							Value:           "15호",
							AdditionalPrice: 9000,
							StockQuantity:   7,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "제주 신제주 라운지점",
				Region:      "제주특별자치도",
				District:    "제주시",
				Address:     "제주특별자치도 제주시 연북로 567",
				PhoneNumber: "064-723-5565",
				ImageURL:    "https://cdn.udonggeum.com/stores/jeju-shinjeju.jpg",
				Description: "제주의 자연을 모티브로 한 감성 주얼리 전문 매장",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "제주 오름 실버 목걸이",
						Description:     "제주의 오름 실루엣을 형상화한 실버 목걸이",
						Price:           108000,
						Weight:          4.0,
						Purity:          "925",
						Category:        model.CategoryNecklace,
						Material:        model.MaterialSilver,
						StockQuantity:   18,
						ImageURL:        "https://cdn.udonggeum.com/products/jeju-oreum-necklace.jpg",
						PopularityScore: 82,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "42cm",
							AdditionalPrice: 0,
							StockQuantity:   6,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "45cm",
							AdditionalPrice: 6000,
							StockQuantity:   6,
						},
						{
							Name:            "길이",
							Value:           "50cm",
							AdditionalPrice: 12000,
							StockQuantity:   6,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "14K 산호 참 반지",
						Description:     "제주 산호를 닮은 참 장식을 더한 14K 반지",
						Price:           356000,
						Weight:          4.6,
						Purity:          "14K",
						Category:        model.CategoryRing,
						Material:        model.MaterialGold,
						StockQuantity:   12,
						ImageURL:        "https://cdn.udonggeum.com/products/14k-coral-charm-ring.jpg",
						PopularityScore: 88,
					},
					Options: []model.ProductOption{
						{
							Name:            "사이즈",
							Value:           "9호",
							AdditionalPrice: 0,
							StockQuantity:   4,
							IsDefault:       true,
						},
						{
							Name:            "사이즈",
							Value:           "11호",
							AdditionalPrice: 15000,
							StockQuantity:   4,
						},
						{
							Name:            "사이즈",
							Value:           "13호",
							AdditionalPrice: 25000,
							StockQuantity:   4,
						},
					},
				},
			},
		},
		{
			Store: model.Store{
				Name:        "제주 서귀포 마린점",
				Region:      "제주특별자치도",
				District:    "서귀포시",
				Address:     "제주특별자치도 서귀포시 중문관광로 72",
				PhoneNumber: "064-739-8844",
				ImageURL:    "https://cdn.udonggeum.com/stores/jeju-seogwipo.jpg",
				Description: "바다를 모티브로 한 주얼리를 만날 수 있는 해안가 매장",
			},
			Products: []productSeed{
				{
					Product: model.Product{
						Name:            "진주 드롭 체인 귀걸이",
						Description:     "바다의 진주를 닮은 드롭 체인 디자인 귀걸이",
						Price:           178000,
						Weight:          3.1,
						Purity:          "925",
						Category:        model.CategoryEarring,
						Material:        model.MaterialOther,
						StockQuantity:   21,
						ImageURL:        "https://cdn.udonggeum.com/products/pearl-chain-earring.jpg",
						PopularityScore: 86,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "4cm",
							AdditionalPrice: 0,
							StockQuantity:   7,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "6cm",
							AdditionalPrice: 6000,
							StockQuantity:   7,
						},
						{
							Name:            "길이",
							Value:           "8cm",
							AdditionalPrice: 12000,
							StockQuantity:   7,
						},
					},
				},
				{
					Product: model.Product{
						Name:            "실버 파도 팔찌",
						Description:     "서귀포 앞바다의 파도를 형상화한 실버 팔찌",
						Price:           98000,
						Weight:          5.0,
						Purity:          "925",
						Category:        model.CategoryBracelet,
						Material:        model.MaterialSilver,
						StockQuantity:   18,
						ImageURL:        "https://cdn.udonggeum.com/products/silver-wave-bracelet.jpg",
						PopularityScore: 80,
					},
					Options: []model.ProductOption{
						{
							Name:            "길이",
							Value:           "16cm",
							AdditionalPrice: 0,
							StockQuantity:   6,
							IsDefault:       true,
						},
						{
							Name:            "길이",
							Value:           "18cm",
							AdditionalPrice: 6000,
							StockQuantity:   6,
						},
						{
							Name:            "길이",
							Value:           "20cm",
							AdditionalPrice: 12000,
							StockQuantity:   6,
						},
					},
				},
			},
		},
	}

	totalStores := 0
	totalProducts := 0
	totalOptions := 0

	for _, seed := range stores {
		store := seed.Store
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

		for _, productSeed := range seed.Products {
			product := productSeed.Product
			product.StoreID = createdStore.ID

			var createdProduct model.Product
			if err := DB.Where("store_id = ? AND name = ?", createdStore.ID, product.Name).
				Attrs(product).
				FirstOrCreate(&createdProduct).Error; err != nil {
				logger.Error("Failed to seed product", err, map[string]interface{}{
					"product": product.Name,
					"store":   createdStore.Name,
				})
				return err
			}
			totalProducts++

			for _, optionSeed := range productSeed.Options {
				option := optionSeed
				option.ProductID = createdProduct.ID

				var createdOption model.ProductOption
				if err := DB.Where("product_id = ? AND name = ? AND value = ?", createdProduct.ID, option.Name, option.Value).
					Attrs(option).
					FirstOrCreate(&createdOption).Error; err != nil {
					logger.Error("Failed to seed product option", err, map[string]interface{}{
						"product": createdProduct.Name,
						"option":  option.Name,
						"value":   option.Value,
					})
					return err
				}
				totalOptions++
			}
		}
	}

	logger.Info("Database seeded successfully", map[string]interface{}{
		"stores_count":   totalStores,
		"products_count": totalProducts,
		"options_count":  totalOptions,
	})
	return nil
}
