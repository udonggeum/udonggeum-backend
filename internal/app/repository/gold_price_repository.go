package repository

import (
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

// GoldPriceRepository 금 시세 저장소 인터페이스
type GoldPriceRepository interface {
	Create(goldPrice *model.GoldPrice) error
	FindAll() ([]model.GoldPrice, error)
	FindByType(priceType model.GoldPriceType) (*model.GoldPrice, error)
	FindLatest() ([]model.GoldPrice, error)
	FindByTypeAndDate(priceType model.GoldPriceType, date time.Time) (*model.GoldPrice, error)
	FindByTypeAndDateRange(priceType model.GoldPriceType, startDate, endDate time.Time) ([]model.GoldPrice, error)
	Update(goldPrice *model.GoldPrice) error
	Delete(id uint) error
}

type goldPriceRepository struct {
	db *gorm.DB
}

// NewGoldPriceRepository 금 시세 저장소 생성
func NewGoldPriceRepository(db *gorm.DB) GoldPriceRepository {
	return &goldPriceRepository{db: db}
}

// Create 금 시세 생성
func (r *goldPriceRepository) Create(goldPrice *model.GoldPrice) error {
	if err := r.db.Create(goldPrice).Error; err != nil {
		logger.Error("Failed to create gold price", err)
		return err
	}
	return nil
}

// FindAll 모든 금 시세 조회
func (r *goldPriceRepository) FindAll() ([]model.GoldPrice, error) {
	var goldPrices []model.GoldPrice
	if err := r.db.Order("source_date DESC").Find(&goldPrices).Error; err != nil {
		logger.Error("Failed to find all gold prices", err)
		return nil, err
	}
	return goldPrices, nil
}

// FindByType 특정 유형의 최신 금 시세 조회
func (r *goldPriceRepository) FindByType(priceType model.GoldPriceType) (*model.GoldPrice, error) {
	var goldPrice model.GoldPrice
	if err := r.db.Where("type = ?", priceType).
		Order("source_date DESC").
		First(&goldPrice).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error("Failed to find gold price by type", err)
		return nil, err
	}
	return &goldPrice, nil
}

// FindLatest 각 유형별 최신 금 시세 조회
func (r *goldPriceRepository) FindLatest() ([]model.GoldPrice, error) {
	var goldPrices []model.GoldPrice

	// 각 타입별 최신 레코드를 조회하는 서브쿼리
	subQuery := r.db.Model(&model.GoldPrice{}).
		Select("type, MAX(source_date) as max_date").
		Group("type")

	if err := r.db.
		Joins("JOIN (?) as latest ON gold_prices.type = latest.type AND gold_prices.source_date = latest.max_date", subQuery).
		Order("type").
		Find(&goldPrices).Error; err != nil {
		logger.Error("Failed to find latest gold prices", err)
		return nil, err
	}

	return goldPrices, nil
}

// Update 금 시세 업데이트
func (r *goldPriceRepository) Update(goldPrice *model.GoldPrice) error {
	if err := r.db.Save(goldPrice).Error; err != nil {
		logger.Error("Failed to update gold price", err)
		return err
	}
	return nil
}

// FindByTypeAndDate 특정 유형의 특정 날짜 금 시세 조회
func (r *goldPriceRepository) FindByTypeAndDate(priceType model.GoldPriceType, date time.Time) (*model.GoldPrice, error) {
	var goldPrice model.GoldPrice
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	if err := r.db.Where("type = ? AND source_date >= ? AND source_date < ?", priceType, startOfDay, endOfDay).
		Order("source_date DESC").
		First(&goldPrice).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error("Failed to find gold price by type and date", err)
		return nil, err
	}
	return &goldPrice, nil
}

// FindByTypeAndDateRange 특정 유형의 기간별 금 시세 조회
func (r *goldPriceRepository) FindByTypeAndDateRange(priceType model.GoldPriceType, startDate, endDate time.Time) ([]model.GoldPrice, error) {
	var goldPrices []model.GoldPrice
	if err := r.db.Where("type = ? AND source_date >= ? AND source_date <= ?", priceType, startDate, endDate).
		Order("source_date ASC").
		Find(&goldPrices).Error; err != nil {
		logger.Error("Failed to find gold prices by type and date range", err)
		return nil, err
	}
	return goldPrices, nil
}

// Delete 금 시세 삭제
func (r *goldPriceRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.GoldPrice{}, id).Error; err != nil {
		logger.Error("Failed to delete gold price", err)
		return err
	}
	return nil
}
