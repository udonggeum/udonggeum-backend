package scheduler

import (
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/robfig/cron/v3"
)

// GoldPriceScheduler 금 시세 자동 업데이트 스케줄러
type GoldPriceScheduler struct {
	cron             *cron.Cron
	goldPriceService service.GoldPriceService
}

// NewGoldPriceScheduler 금 시세 스케줄러 생성
func NewGoldPriceScheduler(goldPriceService service.GoldPriceService) *GoldPriceScheduler {
	return &GoldPriceScheduler{
		cron:             cron.New(),
		goldPriceService: goldPriceService,
	}
}

// Start 스케줄러 시작
func (s *GoldPriceScheduler) Start() error {
	// 매일 오전 9시에 금 시세 업데이트 (KST 기준)
	// cron 표현식: "0 9 * * *" = 매일 9시 0분
	_, err := s.cron.AddFunc("0 9 * * *", func() {
		logger.Info("Starting scheduled gold price update", nil)

		if err := s.goldPriceService.UpdatePricesFromExternalAPI(); err != nil {
			logger.Error("Failed to update gold prices from scheduler", err)
			return
		}

		logger.Info("Successfully updated gold prices from scheduler", nil)
	})

	if err != nil {
		logger.Error("Failed to add cron job for gold price update", err)
		return err
	}

	s.cron.Start()
	logger.Info("Gold price scheduler started successfully (daily at 9:00 AM)", nil)

	return nil
}

// Stop 스케줄러 중지
func (s *GoldPriceScheduler) Stop() {
	logger.Info("Stopping gold price scheduler...", nil)
	s.cron.Stop()
	logger.Info("Gold price scheduler stopped", nil)
}
