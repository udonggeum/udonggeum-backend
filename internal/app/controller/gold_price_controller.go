package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	apperrors "github.com/ikkim/udonggeum-backend/internal/errors"
)

// GoldPriceController 금 시세 컨트롤러
type GoldPriceController struct {
	goldPriceService service.GoldPriceService
}

// NewGoldPriceController 금 시세 컨트롤러 생성
func NewGoldPriceController(goldPriceService service.GoldPriceService) *GoldPriceController {
	return &GoldPriceController{
		goldPriceService: goldPriceService,
	}
}

// CreateGoldPriceRequest 금 시세 생성 요청
type CreateGoldPriceRequest struct {
	Type        model.GoldPriceType `json:"type" binding:"required"`
	BuyPrice    float64             `json:"buy_price" binding:"required,gt=0"`
	SellPrice   float64             `json:"sell_price" binding:"required,gt=0"`
	Source      string              `json:"source"`
	Description string              `json:"description"`
}

// UpdateGoldPriceRequest 금 시세 업데이트 요청
type UpdateGoldPriceRequest struct {
	BuyPrice    *float64 `json:"buy_price,omitempty" binding:"omitempty,gt=0"`
	SellPrice   *float64 `json:"sell_price,omitempty" binding:"omitempty,gt=0"`
	Source      *string  `json:"source,omitempty"`
	Description *string  `json:"description,omitempty"`
}

// GetLatestPrices 최신 금 시세 조회 (모든 유형)
// @Summary 최신 금 시세 조회
// @Description 각 금 유형별 최신 시세를 조회합니다
// @Tags gold-price
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/gold-prices/latest [get]
func (ctrl *GoldPriceController) GetLatestPrices(c *gin.Context) {
	prices, err := ctrl.goldPriceService.GetLatestPrices()
	if err != nil {
		apperrors.InternalError(c, "금 시세 정보를 가져오는데 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    prices,
	})
}

// GetPriceByType 특정 유형의 최신 금 시세 조회
// @Summary 특정 유형의 금 시세 조회
// @Description 24K, 18K, 14K 중 특정 유형의 최신 시세를 조회합니다
// @Tags gold-price
// @Accept json
// @Produce json
// @Param type path string true "금 유형 (24K, 18K, 14K)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/gold-prices/type/{type} [get]
func (ctrl *GoldPriceController) GetPriceByType(c *gin.Context) {
	priceType := model.GoldPriceType(c.Param("type"))

	// 유효한 금 유형인지 확인
	if !isValidGoldPriceType(priceType) {
		apperrors.BadRequest(c, apperrors.GoldInvalidType, "잘못된 금 종류입니다")
		return
	}

	price, err := ctrl.goldPriceService.GetPriceByType(priceType)
	if err != nil {
		if err == service.ErrGoldPriceNotFound {
			apperrors.NotFound(c, apperrors.GoldPriceNotFound, "금 시세 정보를 찾을 수 없습니다")
			return
		}
		apperrors.InternalError(c, "금 시세 정보를 가져오는데 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    price,
	})
}

// GetPriceHistory 과거 시세 이력 조회
// @Summary 과거 시세 이력 조회
// @Description 특정 금 유형의 과거 시세 이력을 조회합니다
// @Tags gold-price
// @Accept json
// @Produce json
// @Param type path string true "금 유형 (24K, 18K, 14K, Platinum, Silver)"
// @Param period query string false "조회 기간 (1주, 1개월, 3개월, 1년, 전체)" default(1개월)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/gold-prices/history/{type} [get]
func (ctrl *GoldPriceController) GetPriceHistory(c *gin.Context) {
	priceType := model.GoldPriceType(c.Param("type"))
	period := c.DefaultQuery("period", "1개월")

	// 유효한 금 유형인지 확인
	if !isValidGoldPriceType(priceType) {
		apperrors.BadRequest(c, apperrors.GoldInvalidType, "잘못된 금 종류입니다")
		return
	}

	history, err := ctrl.goldPriceService.GetPriceHistory(priceType, period)
	if err != nil {
		apperrors.InternalError(c, "금 시세 이력을 가져오는데 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// UpdateFromExternalAPI 외부 API에서 금 시세 업데이트
// @Summary 외부 API로부터 금 시세 업데이트
// @Description 외부 금 시세 API를 호출하여 최신 시세로 업데이트합니다 (관리자 전용)
// @Tags gold-price
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/gold-prices/update [post]
func (ctrl *GoldPriceController) UpdateFromExternalAPI(c *gin.Context) {
	err := ctrl.goldPriceService.UpdatePricesFromExternalAPI()
	if err != nil {
		apperrors.RespondWithError(c, http.StatusInternalServerError, apperrors.InternalExternalAPI, "외부 API에서 금 시세를 업데이트하는데 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "금 시세가 성공적으로 업데이트되었습니다",
	})
}

// CreatePrice 금 시세 생성 (관리자 전용)
// @Summary 금 시세 생성
// @Description 새로운 금 시세 데이터를 생성합니다 (관리자 전용)
// @Tags gold-price
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateGoldPriceRequest true "금 시세 생성 요청"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/gold-prices [post]
func (ctrl *GoldPriceController) CreatePrice(c *gin.Context) {
	var req CreateGoldPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidInput, "잘못된 요청입니다")
		return
	}

	// 유효한 금 유형인지 확인
	if !isValidGoldPriceType(req.Type) {
		apperrors.BadRequest(c, apperrors.GoldInvalidType, "잘못된 금 종류입니다")
		return
	}

	goldPrice := &model.GoldPrice{
		Type:        req.Type,
		BuyPrice:    req.BuyPrice,
		SellPrice:   req.SellPrice,
		Source:      req.Source,
		Description: req.Description,
	}

	if err := ctrl.goldPriceService.CreatePrice(goldPrice); err != nil {
		apperrors.InternalError(c, "금 시세를 생성하는데 실패했습니다")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "금 시세가 성공적으로 생성되었습니다",
		"data":    goldPrice,
	})
}

// UpdatePrice 금 시세 업데이트 (관리자 전용)
// @Summary 금 시세 업데이트
// @Description 기존 금 시세 데이터를 업데이트합니다 (관리자 전용)
// @Tags gold-price
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "금 시세 ID"
// @Param request body UpdateGoldPriceRequest true "금 시세 업데이트 요청"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/gold-prices/{id} [put]
func (ctrl *GoldPriceController) UpdatePrice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidID, "잘못된 금 시세 ID입니다")
		return
	}

	var req UpdateGoldPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidInput, "잘못된 요청입니다")
		return
	}

	// 기존 데이터 조회
	goldPrice, err := ctrl.goldPriceService.GetPriceByID(uint(id))
	if err != nil {
		apperrors.NotFound(c, "GOLD_PRICE_NOT_FOUND", "금 시세를 찾을 수 없습니다")
		return
	}

	// 포인터 필드 확인 후 업데이트
	if req.BuyPrice != nil {
		goldPrice.BuyPrice = *req.BuyPrice
	}
	if req.SellPrice != nil {
		goldPrice.SellPrice = *req.SellPrice
	}
	if req.Source != nil {
		goldPrice.Source = *req.Source
	}
	if req.Description != nil {
		goldPrice.Description = *req.Description
	}

	if err := ctrl.goldPriceService.UpdatePrice(goldPrice); err != nil {
		apperrors.InternalError(c, "금 시세를 업데이트하는데 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "금 시세가 성공적으로 업데이트되었습니다",
	})
}

// ImportHistoricalDataRequest KRX 과거 데이터 임포트 요청
type ImportHistoricalDataRequest struct {
	StartDate string `json:"start_date" binding:"required"` // YYYYMMDD 형식
	EndDate   string `json:"end_date" binding:"required"`   // YYYYMMDD 형식
}

// ImportHistoricalData KRX API에서 과거 데이터 가져오기 (관리자 전용)
// @Summary KRX 과거 금 시세 데이터 임포트
// @Description KRX API를 통해 과거 금 시세 데이터를 가져와서 저장합니다 (관리자 전용)
// @Tags gold-price
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ImportHistoricalDataRequest true "임포트 요청"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/gold-prices/import [post]
func (ctrl *GoldPriceController) ImportHistoricalData(c *gin.Context) {
	var req ImportHistoricalDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidInput, "잘못된 요청입니다")
		return
	}

	// 날짜 형식 검증 (YYYYMMDD)
	if len(req.StartDate) != 8 || len(req.EndDate) != 8 {
		apperrors.BadRequest(c, apperrors.ValidationInvalidInput, "날짜 형식은 YYYYMMDD여야 합니다")
		return
	}

	count, err := ctrl.goldPriceService.ImportHistoricalDataFromKRX(req.StartDate, req.EndDate)
	if err != nil {
		apperrors.RespondWithError(c, http.StatusInternalServerError, apperrors.InternalExternalAPI,
			"KRX API에서 과거 데이터를 가져오는데 실패했습니다: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "KRX 과거 데이터를 성공적으로 가져왔습니다",
		"data": gin.H{
			"imported_count": count,
			"start_date":     req.StartDate,
			"end_date":       req.EndDate,
		},
	})
}

// isValidGoldPriceType 유효한 금 시세 유형인지 확인
func isValidGoldPriceType(priceType model.GoldPriceType) bool {
	switch priceType {
	case model.Gold24K, model.Gold18K, model.Gold14K, model.Platinum, model.Silver:
		return true
	default:
		return false
	}
}
