package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

var (
	ErrGoldPriceNotFound     = errors.New("금 시세를 찾을 수 없습니다")
	ErrExternalAPIFailed     = errors.New("외부 API에서 금 시세를 가져오는데 실패했습니다")
	ErrInvalidGoldPriceType  = errors.New("잘못된 금 종류입니다")
)

// ExternalGoldPriceAPI 외부 금 시세 API 인터페이스
type ExternalGoldPriceAPI interface {
	FetchGoldPrices() (map[model.GoldPriceType]GoldPriceData, error)
}

// GoldPriceData 금 시세 데이터
type GoldPriceData struct {
	BuyPrice  float64
	SellPrice float64
}

// GoldPriceService 금 시세 서비스 인터페이스
type GoldPriceService interface {
	GetLatestPrices() ([]model.GoldPriceResponse, error)
	GetPriceByID(id uint) (*model.GoldPrice, error)
	GetPriceByType(priceType model.GoldPriceType) (*model.GoldPriceResponse, error)
	GetPriceHistory(priceType model.GoldPriceType, period string) ([]model.GoldPriceHistoryItem, error)
	UpdatePricesFromExternalAPI() error
	CreatePrice(goldPrice *model.GoldPrice) error
	UpdatePrice(goldPrice *model.GoldPrice) error
	ImportHistoricalDataFromKRX(startDate, endDate string) (int, error)
}

type goldPriceService struct {
	repo        repository.GoldPriceRepository
	externalAPI ExternalGoldPriceAPI
	krxAPIURL   string
	krxAPIKey   string
}

// NewGoldPriceService 금 시세 서비스 생성
func NewGoldPriceService(repo repository.GoldPriceRepository, externalAPI ExternalGoldPriceAPI, krxAPIURL, krxAPIKey string) GoldPriceService {
	return &goldPriceService{
		repo:        repo,
		externalAPI: externalAPI,
		krxAPIURL:   krxAPIURL,
		krxAPIKey:   krxAPIKey,
	}
}

// GetLatestPrices 최신 금 시세 조회 (모든 유형)
func (s *goldPriceService) GetLatestPrices() ([]model.GoldPriceResponse, error) {
	goldPrices, err := s.repo.FindLatest()
	if err != nil {
		logger.Error("Failed to get latest gold prices", err)
		return nil, err
	}

	responses := make([]model.GoldPriceResponse, 0, len(goldPrices))
	for _, gp := range goldPrices {
		response := model.GoldPriceResponse{
			Type:        gp.Type,
			BuyPrice:    gp.BuyPrice,
			SellPrice:   gp.SellPrice,
			Source:      gp.Source,
			SourceDate:  gp.SourceDate.Format(time.RFC3339),
			Description: gp.Description,
			UpdatedAt:   gp.UpdatedAt.Format(time.RFC3339),
		}

		// 전일 데이터 조회
		yesterday := time.Now().AddDate(0, 0, -1)
		previousPrice, err := s.repo.FindByTypeAndDate(gp.Type, yesterday)
		if err == nil && previousPrice != nil {
			// 전일 대비 변동률 계산
			changeAmount := gp.SellPrice - previousPrice.SellPrice
			changePercent := (changeAmount / previousPrice.SellPrice) * 100

			response.PreviousDayPrice = &previousPrice.SellPrice
			response.ChangeAmount = &changeAmount
			response.ChangePercent = &changePercent
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// GetPriceByType 특정 유형의 최신 금 시세 조회
func (s *goldPriceService) GetPriceByType(priceType model.GoldPriceType) (*model.GoldPriceResponse, error) {
	goldPrice, err := s.repo.FindByType(priceType)
	if err != nil {
		logger.Error("Failed to get gold price by type", err)
		return nil, err
	}

	if goldPrice == nil {
		return nil, ErrGoldPriceNotFound
	}

	response := &model.GoldPriceResponse{
		Type:        goldPrice.Type,
		BuyPrice:    goldPrice.BuyPrice,
		SellPrice:   goldPrice.SellPrice,
		Source:      goldPrice.Source,
		SourceDate:  goldPrice.SourceDate.Format(time.RFC3339),
		Description: goldPrice.Description,
		UpdatedAt:   goldPrice.UpdatedAt.Format(time.RFC3339),
	}

	return response, nil
}

// UpdatePricesFromExternalAPI 외부 API에서 금 시세 업데이트
func (s *goldPriceService) UpdatePricesFromExternalAPI() error {
	if s.externalAPI == nil {
		return errors.New("외부 API가 설정되지 않았습니다")
	}

	prices, err := s.externalAPI.FetchGoldPrices()
	if err != nil {
		logger.Error("Failed to fetch gold prices from external API", err)
		return ErrExternalAPIFailed
	}

	now := time.Now()
	for priceType, priceData := range prices {
		goldPrice := &model.GoldPrice{
			Type:       priceType,
			BuyPrice:   priceData.BuyPrice,
			SellPrice:  priceData.SellPrice,
			Source:     "External API",
			SourceDate: now,
		}

		if err := s.repo.Create(goldPrice); err != nil {
			logger.Error("Failed to save gold price", err)
			return err
		}
	}

	logger.Info("Successfully updated gold prices from external API", map[string]interface{}{
		"count": len(prices),
	})

	return nil
}

// CreatePrice 금 시세 생성
func (s *goldPriceService) CreatePrice(goldPrice *model.GoldPrice) error {
	if err := s.repo.Create(goldPrice); err != nil {
		logger.Error("Failed to create gold price", err)
		return err
	}
	return nil
}

// GetPriceHistory 과거 시세 이력 조회
func (s *goldPriceService) GetPriceHistory(priceType model.GoldPriceType, period string) ([]model.GoldPriceHistoryItem, error) {
	days := getPeriodDays(period)
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	prices, err := s.repo.FindByTypeAndDateRange(priceType, startDate, endDate)
	if err != nil {
		logger.Error("Failed to get price history", err)
		return nil, err
	}

	history := make([]model.GoldPriceHistoryItem, 0, len(prices))
	for _, price := range prices {
		history = append(history, model.GoldPriceHistoryItem{
			Date:      price.SourceDate.Format("2006-01-02"),
			SellPrice: price.SellPrice,
			BuyPrice:  price.BuyPrice,
		})
	}

	return history, nil
}

// getPeriodDays 기간 문자열을 일수로 변환
func getPeriodDays(period string) int {
	switch period {
	case "1주":
		return 7
	case "1개월":
		return 30
	case "3개월":
		return 90
	case "1년":
		return 365
	case "전체":
		return 730 // 2년
	default:
		return 30
	}
}

// GetPriceByID ID로 금 시세 조회
func (s *goldPriceService) GetPriceByID(id uint) (*model.GoldPrice, error) {
	goldPrice, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("Failed to get gold price by ID", err)
		return nil, err
	}
	if goldPrice == nil {
		return nil, fmt.Errorf("금 시세를 찾을 수 없습니다")
	}
	return goldPrice, nil
}

// UpdatePrice 금 시세 업데이트
func (s *goldPriceService) UpdatePrice(goldPrice *model.GoldPrice) error {
	if goldPrice == nil {
		return fmt.Errorf("goldPrice cannot be nil")
	}
	if err := s.repo.Update(goldPrice); err != nil {
		logger.Error("Failed to update gold price", err)
		return err
	}
	return nil
}

// DefaultGoldPriceAPI 기본 금 시세 API 구현체
type DefaultGoldPriceAPI struct {
	apiURL string
	apiKey string
}

// NewDefaultGoldPriceAPI 기본 금 시세 API 생성
func NewDefaultGoldPriceAPI(apiURL, apiKey string) *DefaultGoldPriceAPI {
	return &DefaultGoldPriceAPI{
		apiURL: apiURL,
		apiKey: apiKey,
	}
}

// GoldAPIResponse GOLD API 응답 구조체
type GoldAPIResponse struct {
	Timestamp      int64   `json:"timestamp"`
	Metal          string  `json:"metal"`
	Currency       string  `json:"currency"`
	Exchange       string  `json:"exchange"`
	Symbol         string  `json:"symbol"`
	OpenTime       int64   `json:"open_time"`
	Ask            float64 `json:"ask"`
	Bid            float64 `json:"bid"`
	Price          float64 `json:"price"`
	Ch             float64 `json:"ch"`
	PriceGram24K   float64 `json:"price_gram_24k"`
	PriceGram22K   float64 `json:"price_gram_22k"`
	PriceGram21K   float64 `json:"price_gram_21k"`
	PriceGram20K   float64 `json:"price_gram_20k"`
	PriceGram18K   float64 `json:"price_gram_18k"`
	PriceGram16K   float64 `json:"price_gram_16k"`
	PriceGram14K   float64 `json:"price_gram_14k"`
	PriceGram10K   float64 `json:"price_gram_10k"`
}

// FetchGoldPrices 외부 API에서 금 시세 조회 (GOLDAPI)
func (api *DefaultGoldPriceAPI) FetchGoldPrices() (map[model.GoldPriceType]GoldPriceData, error) {
	if api.apiURL == "" {
		return nil, errors.New("금 시세 API URL이 설정되지 않았습니다")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", api.apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// GOLDAPI는 헤더에 API 키를 전달
	if api.apiKey != "" {
		req.Header.Set("x-access-token", api.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// GOLDAPI 응답 파싱
	var apiResponse GoldAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// 금 시세 데이터로 변환
	// API는 현재가만 제공하므로, 매입/매도가를 현재가 기준으로 설정
	// 일반적으로 매입가는 현재가보다 낮고, 매도가는 높게 설정
	prices := make(map[model.GoldPriceType]GoldPriceData)

	// 24K 금 시세 (순금)
	if apiResponse.PriceGram24K > 0 {
		prices[model.Gold24K] = GoldPriceData{
			BuyPrice:  apiResponse.PriceGram24K * 0.98,  // 현재가의 98%를 매입가로
			SellPrice: apiResponse.PriceGram24K * 1.02,  // 현재가의 102%를 매도가로
		}
	}

	// 18K 금 시세
	if apiResponse.PriceGram18K > 0 {
		prices[model.Gold18K] = GoldPriceData{
			BuyPrice:  apiResponse.PriceGram18K * 0.98,
			SellPrice: apiResponse.PriceGram18K * 1.02,
		}
	}

	// 14K 금 시세
	if apiResponse.PriceGram14K > 0 {
		prices[model.Gold14K] = GoldPriceData{
			BuyPrice:  apiResponse.PriceGram14K * 0.98,
			SellPrice: apiResponse.PriceGram14K * 1.02,
		}
	}

	if len(prices) == 0 {
		return nil, errors.New("API로부터 유효한 금 시세 데이터를 받지 못했습니다")
	}

	logger.Info("Successfully fetched gold prices from GOLDAPI", map[string]interface{}{
		"24K": apiResponse.PriceGram24K,
		"18K": apiResponse.PriceGram18K,
		"14K": apiResponse.PriceGram14K,
	})

	return prices, nil
}

// KRXAPIResponse KRX API 응답 구조체
type KRXAPIResponse struct {
	Response struct {
		Header struct {
			ResultCode string `json:"resultCode"`
			ResultMsg  string `json:"resultMsg"`
		} `json:"header"`
		Body struct {
			NumOfRows  int    `json:"numOfRows"`
			PageNo     int    `json:"pageNo"`
			TotalCount int    `json:"totalCount"`
			Items      struct {
				Item []KRXGoldPriceItem `json:"item"`
			} `json:"items"`
		} `json:"body"`
	} `json:"response"`
}

// KRXGoldPriceItem KRX 금 시세 아이템
type KRXGoldPriceItem struct {
	BasDt   string  `json:"basDt"`   // 기준일자 (YYYYMMDD)
	SrtnCd  string  `json:"srtnCd"`  // 단축코드
	IsinCd  string  `json:"isinCd"`  // ISIN코드
	ItmsNm  string  `json:"itmsNm"`  // 종목명
	Clpr    string  `json:"clpr"`    // 종가
	Vs      string  `json:"vs"`      // 대비
	FltRt   string  `json:"fltRt"`   // 등락률
	Mkp     string  `json:"mkp"`     // 시가
	Hipr    string  `json:"hipr"`    // 고가
	Lopr    string  `json:"lopr"`    // 저가
	Trqu    string  `json:"trqu"`    // 거래량
	TrPrc   string  `json:"trPrc"`   // 거래대금
}

// ImportHistoricalDataFromKRX KRX API에서 과거 데이터 가져오기
func (s *goldPriceService) ImportHistoricalDataFromKRX(startDate, endDate string) (int, error) {
	if s.krxAPIURL == "" || s.krxAPIKey == "" {
		return 0, errors.New("KRX API URL 또는 API Key가 설정되지 않았습니다")
	}

	logger.Info("Starting KRX historical data import", map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	})

	importedCount := 0
	pageNo := 1
	numOfRows := 100

	for {
		// API 호출
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		req, err := http.NewRequest("GET", s.krxAPIURL, nil)
		if err != nil {
			return importedCount, fmt.Errorf("failed to create request: %w", err)
		}

		// 쿼리 파라미터 설정
		q := req.URL.Query()
		q.Add("serviceKey", s.krxAPIKey)
		q.Add("pageNo", fmt.Sprintf("%d", pageNo))
		q.Add("numOfRows", fmt.Sprintf("%d", numOfRows))
		q.Add("resultType", "json")
		q.Add("beginBasDt", startDate) // YYYYMMDD 형식
		q.Add("endBasDt", endDate)     // YYYYMMDD 형식
		req.URL.RawQuery = q.Encode()

		logger.Info("Fetching KRX data", map[string]interface{}{
			"page":  pageNo,
			"url":   req.URL.String(),
		})

		resp, err := client.Do(req)
		if err != nil {
			return importedCount, fmt.Errorf("failed to call KRX API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return importedCount, fmt.Errorf("KRX API returned status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return importedCount, fmt.Errorf("failed to read response body: %w", err)
		}

		// 응답 파싱
		var apiResponse KRXAPIResponse
		if err := json.Unmarshal(body, &apiResponse); err != nil {
			return importedCount, fmt.Errorf("failed to parse KRX API response: %w", err)
		}

		// 에러 체크
		if apiResponse.Response.Header.ResultCode != "00" {
			return importedCount, fmt.Errorf("KRX API error: %s - %s",
				apiResponse.Response.Header.ResultCode,
				apiResponse.Response.Header.ResultMsg)
		}

		// 데이터가 없으면 종료
		if len(apiResponse.Response.Body.Items.Item) == 0 {
			logger.Info("No more data to import from KRX", map[string]interface{}{
				"page":           pageNo,
				"imported_count": importedCount,
			})
			break
		}

		// 데이터 저장
		for _, item := range apiResponse.Response.Body.Items.Item {
			// 24K (순금) 데이터만 처리
			goldPrice24K, err := s.convertKRXItemToGoldPrice(item)
			if err != nil {
				logger.Warn("Failed to convert KRX item", map[string]interface{}{
					"item":  item.ItmsNm,
					"error": err.Error(),
				})
				continue
			}

			if goldPrice24K == nil {
				continue // 순금이 아닌 경우 스킵
			}

			// 날짜 파싱
			sourceDate, _ := time.Parse("20060102", item.BasDt)

			// 24K, 18K, 14K 데이터 생성
			goldPrices := []*model.GoldPrice{
				goldPrice24K, // 24K (순금)
				{
					Type:        model.Gold18K,
					BuyPrice:    goldPrice24K.BuyPrice * 0.75,  // 18K = 24K × (18/24)
					SellPrice:   goldPrice24K.SellPrice * 0.75,
					Source:      "KRX",
					SourceDate:  sourceDate,
					Description: fmt.Sprintf("KRX 금시장 시세 기반 계산 (18K) - %s", item.ItmsNm),
				},
				{
					Type:        model.Gold14K,
					BuyPrice:    goldPrice24K.BuyPrice * (14.0 / 24.0),  // 14K = 24K × (14/24)
					SellPrice:   goldPrice24K.SellPrice * (14.0 / 24.0),
					Source:      "KRX",
					SourceDate:  sourceDate,
					Description: fmt.Sprintf("KRX 금시장 시세 기반 계산 (14K) - %s", item.ItmsNm),
				},
			}

			// 각 금 종류별로 저장
			for _, gp := range goldPrices {
				// 중복 체크 (같은 날짜, 같은 타입의 데이터가 이미 있는지)
				existing, err := s.repo.FindByTypeAndDate(gp.Type, sourceDate)
				if err != nil {
					logger.Error("Failed to check existing data", err)
					continue
				}

				if existing != nil {
					logger.Info("Skipping duplicate data", map[string]interface{}{
						"type": gp.Type,
						"date": item.BasDt,
					})
					continue
				}

				// 데이터 저장
				if err := s.repo.Create(gp); err != nil {
					logger.Error("Failed to save KRX gold price", err, map[string]interface{}{
						"type": gp.Type,
						"date": item.BasDt,
					})
					continue
				}

				importedCount++
			}
		}

		logger.Info("Imported KRX data page", map[string]interface{}{
			"page":           pageNo,
			"items":          len(apiResponse.Response.Body.Items.Item),
			"imported_count": importedCount,
			"total_count":    apiResponse.Response.Body.TotalCount,
		})

		// 모든 데이터를 가져왔으면 종료
		if pageNo * numOfRows >= apiResponse.Response.Body.TotalCount {
			break
		}

		pageNo++
		time.Sleep(100 * time.Millisecond) // API 호출 제한 방지
	}

	logger.Info("Completed KRX historical data import", map[string]interface{}{
		"imported_count": importedCount,
		"start_date":     startDate,
		"end_date":       endDate,
	})

	return importedCount, nil
}

// convertKRXItemToGoldPrice KRX 아이템을 GoldPrice로 변환 (24K 순금만)
func (s *goldPriceService) convertKRXItemToGoldPrice(item KRXGoldPriceItem) (*model.GoldPrice, error) {
	// 종목명 분석 - 순금(99.99%, 24K)만 처리
	if !contains(item.ItmsNm, "99.99") && !contains(item.ItmsNm, "순금") && !contains(item.ItmsNm, "24K") {
		return nil, nil // 순금이 아닌 경우 스킵
	}

	// 종가를 float64로 변환
	var clpr float64
	if _, err := fmt.Sscanf(item.Clpr, "%f", &clpr); err != nil {
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	// KRX는 종가만 제공하므로, 매입가/매도가를 종가 기준으로 설정
	// 일반적으로 매입가는 시세보다 낮고, 매도가는 높음
	buyPrice := clpr * 0.98  // 종가의 98%를 매입가로
	sellPrice := clpr * 1.02 // 종가의 102%를 매도가로

	// 날짜 파싱 (YYYYMMDD -> time.Time)
	sourceDate, err := time.Parse("20060102", item.BasDt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}

	goldPrice := &model.GoldPrice{
		Type:        model.Gold24K,
		BuyPrice:    buyPrice,
		SellPrice:   sellPrice,
		Source:      "KRX",
		SourceDate:  sourceDate,
		Description: fmt.Sprintf("KRX 금시장 시세 (순금 99.99%%) - %s", item.ItmsNm),
	}

	return goldPrice, nil
}

// contains 문자열 포함 여부 체크
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
