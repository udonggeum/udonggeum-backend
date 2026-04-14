package service

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
)

type FAQService interface {
	GetAll() ([]model.FAQ, error)
	GetByTarget(target model.FAQTarget) ([]model.FAQ, error)
	Create(faq *model.FAQ) error
	Update(id uint, question, answer string, sortOrder int) (*model.FAQ, error)
	Delete(id uint) error
}

type faqService struct {
	faqRepo repository.FAQRepository
}

func NewFAQService(faqRepo repository.FAQRepository) FAQService {
	return &faqService{faqRepo: faqRepo}
}

func (s *faqService) GetAll() ([]model.FAQ, error) {
	return s.faqRepo.FindAll()
}

func (s *faqService) GetByTarget(target model.FAQTarget) ([]model.FAQ, error) {
	return s.faqRepo.FindByTarget(target)
}

func (s *faqService) Create(faq *model.FAQ) error {
	return s.faqRepo.Create(faq)
}

func (s *faqService) Update(id uint, question, answer string, sortOrder int) (*model.FAQ, error) {
	faq, err := s.faqRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	faq.Question = question
	faq.Answer = answer
	faq.SortOrder = sortOrder
	if err := s.faqRepo.Update(faq); err != nil {
		return nil, err
	}
	return faq, nil
}

func (s *faqService) Delete(id uint) error {
	return s.faqRepo.Delete(id)
}
