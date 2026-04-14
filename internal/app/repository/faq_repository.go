package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"gorm.io/gorm"
)

type FAQRepository interface {
	FindAll() ([]model.FAQ, error)
	FindByTarget(target model.FAQTarget) ([]model.FAQ, error)
	FindByID(id uint) (*model.FAQ, error)
	Create(faq *model.FAQ) error
	Update(faq *model.FAQ) error
	Delete(id uint) error
}

type faqRepository struct {
	db *gorm.DB
}

func NewFAQRepository(db *gorm.DB) FAQRepository {
	return &faqRepository{db: db}
}

func (r *faqRepository) FindAll() ([]model.FAQ, error) {
	var faqs []model.FAQ
	err := r.db.Order("target, sort_order, id").Find(&faqs).Error
	return faqs, err
}

func (r *faqRepository) FindByTarget(target model.FAQTarget) ([]model.FAQ, error) {
	var faqs []model.FAQ
	err := r.db.Where("target = ?", target).Order("sort_order, id").Find(&faqs).Error
	return faqs, err
}

func (r *faqRepository) FindByID(id uint) (*model.FAQ, error) {
	var faq model.FAQ
	err := r.db.First(&faq, id).Error
	if err != nil {
		return nil, err
	}
	return &faq, nil
}

func (r *faqRepository) Create(faq *model.FAQ) error {
	return r.db.Create(faq).Error
}

func (r *faqRepository) Update(faq *model.FAQ) error {
	return r.db.Save(faq).Error
}

func (r *faqRepository) Delete(id uint) error {
	return r.db.Delete(&model.FAQ{}, id).Error
}
