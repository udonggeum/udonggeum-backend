package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/xuri/excelize/v2"
)

func main() {
	// 명령줄 인자 확인
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/seed/main.go <xlsx_file_path>")
	}

	filePath := os.Args[1]

	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// DB 연결
	if err := db.Initialize(&cfg.Database); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Repository 생성
	storeRepo := repository.NewStoreRepository(db.GetDB())

	// XLSX 파일 읽기
	fmt.Printf("Reading XLSX file: %s\n", filePath)
	stores, err := readStoresFromXLSX(filePath)
	if err != nil {
		log.Fatal("Failed to read XLSX:", err)
	}

	fmt.Printf("Total stores to import: %d\n", len(stores))

	// 사용자 확인
	fmt.Print("Do you want to proceed with the import? (yes/no): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "yes" && confirm != "y" {
		fmt.Println("Import cancelled.")
		return
	}

	// 배치로 저장
	batchSize := 1000
	fmt.Printf("Starting bulk import with batch size: %d\n", batchSize)
	if err := storeRepo.BulkCreate(stores, batchSize); err != nil {
		log.Fatal("Failed to bulk create stores:", err)
	}

	fmt.Println("Import completed successfully!")
	fmt.Printf("Total stores imported: %d\n", len(stores))
}

func readStoresFromXLSX(filePath string) ([]model.Store, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX file: %w", err)
	}
	defer f.Close()

	// 첫 번째 시트 이름 가져오기
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no sheets found in XLSX file")
	}

	fmt.Printf("Reading sheet: %s\n", sheetName)

	// 모든 행 읽기
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no data found in XLSX file")
	}

	var stores []model.Store
	seenStores := make(map[string]bool) // 중복 제거용
	skippedCount := 0
	invalidCoordCount := 0

	// 첫 행은 헤더이므로 스킵
	for i, row := range rows {
		if i == 0 {
			// 헤더 출력 (디버깅용)
			fmt.Printf("Headers: %v\n", row)
			continue
		}

		// 필수 컬럼 수 확인 (총 38개 컬럼)
		if len(row) < 38 {
			skippedCount++
			continue
		}

		// 데이터 추출
		// 인덱스는 0부터 시작
		name := strings.TrimSpace(row[1])          // 상호명
		branchName := strings.TrimSpace(row[2])    // 지점명
		region := strings.TrimSpace(row[12])       // 시도명
		district := strings.TrimSpace(row[14])     // 시군구명
		dong := strings.TrimSpace(row[16])         // 행정동명
		jibunAddr := strings.TrimSpace(row[24])    // 지번주소
		buildingName := strings.TrimSpace(row[29]) // 건물명
		roadAddr := strings.TrimSpace(row[30])     // 도로명주소
		postalCode := strings.TrimSpace(row[33])   // 신우편번호
		floor := strings.TrimSpace(row[34])        // 층정보
		unit := strings.TrimSpace(row[35])         // 호정보
		longitudeStr := strings.TrimSpace(row[36]) // 경도
		latitudeStr := strings.TrimSpace(row[37])  // 위도

		// 유효성 검사
		if name == "" || region == "" || district == "" {
			skippedCount++
			continue
		}

		// 주소 선택 (도로명 우선, 없으면 지번)
		address := roadAddr
		if address == "" {
			address = jibunAddr
		}

		// 좌표 파싱
		var longitude, latitude *float64
		if lng, err := strconv.ParseFloat(longitudeStr, 64); err == nil && lng != 0 {
			longitude = &lng
		} else {
			invalidCoordCount++
		}
		if lat, err := strconv.ParseFloat(latitudeStr, 64); err == nil && lat != 0 {
			latitude = &lat
		} else {
			invalidCoordCount++
		}

		// 중복 체크 (이름+지역+주소 기준)
		key := fmt.Sprintf("%s|%s|%s|%s", name, region, district, address)
		if seenStores[key] {
			skippedCount++
			continue
		}
		seenStores[key] = true

		// Store 모델 생성
		store := model.Store{
			Name:         name,
			BranchName:   branchName,
			Region:       region,
			District:     district,
			Dong:         dong,
			Address:      address,
			BuildingName: buildingName,
			Floor:        floor,
			Unit:         unit,
			PostalCode:   postalCode,
			Longitude:    longitude,
			Latitude:     latitude,
			UserID:       nil,   // 비관리 매장
			IsManaged:    false,
			IsVerified:   false,
			// Slug는 BeforeCreate에서 자동 생성됨
			// 기타 필드들은 기본값 사용
		}

		stores = append(stores, store)

		// 진행 상황 출력 (1000개마다)
		if len(stores)%1000 == 0 {
			fmt.Printf("Processed %d stores...\n", len(stores))
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total rows: %d\n", len(rows)-1)
	fmt.Printf("  Valid stores: %d\n", len(stores))
	fmt.Printf("  Skipped rows: %d\n", skippedCount)
	fmt.Printf("  Rows with invalid coordinates: %d\n", invalidCoordCount)

	return stores, nil
}
