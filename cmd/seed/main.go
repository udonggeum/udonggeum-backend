package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
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
	seenStores := make(map[string]bool)  // 중복 제거용
	slugCounter := make(map[string]int)  // slug 중복 처리용
	skippedCount := 0
	invalidCoordCount := 0

	// 첫 행은 헤더이므로 스킵
	for i, row := range rows {
		if i == 0 {
			// 헤더 출력 (디버깅용)
			fmt.Printf("Headers: %v\n", row)
			continue
		}

		// 필수 컬럼 수 확인 (총 39개 컬럼)
		if len(row) < 39 {
			skippedCount++
			continue
		}

		// 데이터 추출
		// 인덱스는 0부터 시작
		businessNumber := strings.TrimSpace(row[0]) // 상가업소번호
		name := strings.TrimSpace(row[1])          // 상호명
		branchName := strings.TrimSpace(row[2])    // 지점명
		region := strings.TrimSpace(row[12])       // 시도명
		district := strings.TrimSpace(row[14])     // 시군구명
		dong := strings.TrimSpace(row[16])         // 행정동명
		jibunAddr := strings.TrimSpace(row[24])    // 지번주소
		buildingName := strings.TrimSpace(row[30]) // 건물명 (수정: 29 → 30)
		roadAddr := strings.TrimSpace(row[31])     // 도로명주소 (수정: 30 → 31)
		postalCode := strings.TrimSpace(row[34])   // 신우편번호 (수정: 33 → 34)
		floor := strings.TrimSpace(row[35])        // 층정보 (수정: 34 → 35)
		unit := strings.TrimSpace(row[36])         // 호정보 (수정: 35 → 36)
		longitudeStr := strings.TrimSpace(row[37]) // 경도 (수정: 36 → 37)
		latitudeStr := strings.TrimSpace(row[38])  // 위도 (수정: 37 → 38)

		// 1. 기본 필수 항목 검사
		if businessNumber == "" || name == "" || region == "" || district == "" {
			skippedCount++
			continue
		}

		// 2. 상호명 품질 검증
		if !isValidStoreName(name) {
			skippedCount++
			continue
		}

		// 3. 주소 유효성 검증 (도로명주소나 지번주소 둘 중 하나는 필수)
		if roadAddr == "" && jibunAddr == "" {
			skippedCount++
			continue
		}

		// 주소 선택 (도로명 우선, 없으면 지번)
		address := roadAddr
		if address == "" {
			address = jibunAddr
		}

		// 4. 좌표 유효성 검증 (경도/위도 둘 다 필수)
		var longitude, latitude *float64
		lng, errLng := strconv.ParseFloat(longitudeStr, 64)
		lat, errLat := strconv.ParseFloat(latitudeStr, 64)

		if errLng != nil || errLat != nil || lng == 0 || lat == 0 {
			invalidCoordCount++
			skippedCount++
			continue
		}

		longitude = &lng
		latitude = &lat

		// 중복 체크 (이름+지역+주소 기준)
		key := fmt.Sprintf("%s|%s|%s|%s", name, region, district, address)
		if seenStores[key] {
			skippedCount++
			continue
		}
		seenStores[key] = true

		// Slug 생성 (중복 처리)
		baseSlug := generateSlug(region, district, name)
		slug := baseSlug
		if count, exists := slugCounter[baseSlug]; exists {
			slugCounter[baseSlug] = count + 1
			slug = fmt.Sprintf("%s-%d", baseSlug, count+1)
		} else {
			slugCounter[baseSlug] = 1
		}

		// Store 모델 생성
		store := model.Store{
			BusinessNumber: businessNumber,
			Name:           name,
			BranchName:     branchName,
			Slug:           slug, // 미리 생성한 slug 사용
			Region:         region,
			District:       district,
			Dong:           dong,
			Address:        address,
			BuildingName:   buildingName,
			Floor:          floor,
			Unit:           unit,
			PostalCode:     postalCode,
			Longitude:      longitude,
			Latitude:       latitude,
			UserID:         nil,   // 비관리 매장
			IsManaged:      false,
			IsVerified:     false,
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

// generateSlug는 매장명과 지역 정보로 URL용 slug를 생성합니다
func generateSlug(region, district, name string) string {
	// 공백을 하이픈으로 변경
	slug := fmt.Sprintf("%s-%s-%s", region, district, name)

	// 특수문자 제거 (한글, 영문, 숫자, 하이픈만 허용)
	reg := regexp.MustCompile(`[^\p{L}\p{N}-]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// 연속된 하이픈을 하나로
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// 앞뒤 하이픈 제거
	slug = strings.Trim(slug, "-")

	// 소문자로 변환 (영문만)
	slug = strings.ToLower(slug)

	return slug
}

// isValidStoreName은 상호명이 유효한지 검증합니다
func isValidStoreName(name string) bool {
	// 1. 최소 길이 체크 (3글자 미만 제외)
	nameRunes := []rune(name)
	if len(nameRunes) < 3 {
		return false
	}

	// 2. 숫자만 있는 경우 제외
	numOnlyReg := regexp.MustCompile(`^[0-9]+$`)
	if numOnlyReg.MatchString(name) {
		return false
	}

	// 3. 특수문자만 있는 경우 제외 (공백, 구두점, 기호만)
	specialOnlyReg := regexp.MustCompile(`^[\p{P}\p{S}\s]+$`)
	if specialOnlyReg.MatchString(name) {
		return false
	}

	return true
}
