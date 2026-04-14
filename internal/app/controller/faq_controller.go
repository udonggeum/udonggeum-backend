package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
)

type FAQController struct {
	faqService service.FAQService
}

func NewFAQController(faqService service.FAQService) *FAQController {
	return &FAQController{faqService: faqService}
}

// GetFAQs GET /faqs?target=user|owner (공개)
func (c *FAQController) GetFAQs(ctx *gin.Context) {
	target := ctx.Query("target")

	if target == "user" || target == "owner" {
		faqs, err := c.faqService.GetByTarget(model.FAQTarget(target))
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "FAQ 조회에 실패했습니다"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"data": faqs})
		return
	}

	faqs, err := c.faqService.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "FAQ 조회에 실패했습니다"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": faqs})
}

type createFAQRequest struct {
	Target    string `json:"target" binding:"required,oneof=user owner"`
	Question  string `json:"question" binding:"required"`
	Answer    string `json:"answer" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// CreateFAQ POST /faqs (master only)
func (c *FAQController) CreateFAQ(ctx *gin.Context) {
	var req createFAQRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	faq := &model.FAQ{
		Target:    model.FAQTarget(req.Target),
		Question:  req.Question,
		Answer:    req.Answer,
		SortOrder: req.SortOrder,
	}

	if err := c.faqService.Create(faq); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "FAQ 생성에 실패했습니다"})
		return
	}

	ctx.JSON(http.StatusCreated, faq)
}

type updateFAQRequest struct {
	Question  string `json:"question" binding:"required"`
	Answer    string `json:"answer" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// UpdateFAQ PUT /faqs/:id (master only)
func (c *FAQController) UpdateFAQ(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "올바르지 않은 ID입니다"})
		return
	}

	var req updateFAQRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	faq, err := c.faqService.Update(uint(id), req.Question, req.Answer, req.SortOrder)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "FAQ 수정에 실패했습니다"})
		return
	}

	ctx.JSON(http.StatusOK, faq)
}

// DeleteFAQ DELETE /faqs/:id (master only)
func (c *FAQController) DeleteFAQ(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "올바르지 않은 ID입니다"})
		return
	}

	if err := c.faqService.Delete(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "FAQ 삭제에 실패했습니다"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "삭제되었습니다"})
}
