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
	ErrGoldPriceNotFound     = errors.New("gold price not found")
	ErrExternalAPIFailed     = errors.New("failed to fetch gold price from external API")
	ErrInvalidGoldPriceType  = errors.New("invalid gold price type")
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
	GetPriceByType(priceType model.GoldPriceType) (*model.GoldPriceResponse, error)
	UpdatePricesFromExternalAPI() error
	CreatePrice(goldPrice *model.GoldPrice) error
	UpdatePrice(goldPrice *model.GoldPrice) error
}

type goldPriceService struct {
	repo        repository.GoldPriceRepository
	externalAPI ExternalGoldPriceAPI
}

// NewGoldPriceService 금 시세 서비스 생성
func NewGoldPriceService(repo repository.GoldPriceRepository, externalAPI ExternalGoldPriceAPI) GoldPriceService {
	return &goldPriceService{
		repo:        repo,
		externalAPI: externalAPI,
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
		responses = append(responses, model.GoldPriceResponse{
			Type:        gp.Type,
			BuyPrice:    gp.BuyPrice,
			SellPrice:   gp.SellPrice,
			Source:      gp.Source,
			SourceDate:  gp.SourceDate.Format(time.RFC3339),
			Description: gp.Description,
			UpdatedAt:   gp.UpdatedAt.Format(time.RFC3339),
		})
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
		return errors.New("external API not configured")
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

// UpdatePrice 금 시세 업데이트
func (s *goldPriceService) UpdatePrice(goldPrice *model.GoldPrice) error {
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
		return nil, errors.New("gold price API URL is not configured")
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
		return nil, errors.New("no valid gold price data received from API")
	}

	logger.Info("Successfully fetched gold prices from GOLDAPI", map[string]interface{}{
		"24K": apiResponse.PriceGram24K,
		"18K": apiResponse.PriceGram18K,
		"14K": apiResponse.PriceGram14K,
	})

	return prices, nil
}
