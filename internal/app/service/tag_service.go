package service

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"gorm.io/gorm"
)

type TagService interface {
	ListTags() ([]model.Tag, error)
	GetTagsByCategory(category string) ([]model.Tag, error)
}

type tagService struct {
	db *gorm.DB
}

func NewTagService(db *gorm.DB) TagService {
	return &tagService{db: db}
}

// ListTags 모든 태그 목록 조회
func (s *tagService) ListTags() ([]model.Tag, error) {
	var tags []model.Tag
	if err := s.db.Order("category ASC, name ASC").Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// GetTagsByCategory 카테고리별 태그 조회
func (s *tagService) GetTagsByCategory(category string) ([]model.Tag, error) {
	var tags []model.Tag
	query := s.db.Order("name ASC")

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}
