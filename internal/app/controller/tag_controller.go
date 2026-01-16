package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	apperrors "github.com/ikkim/udonggeum-backend/internal/errors"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type TagController struct {
	tagService service.TagService
}

func NewTagController(tagService service.TagService) *TagController {
	return &TagController{tagService: tagService}
}

// ListTags 태그 목록 조회
// GET /api/v1/tags
// Query params:
//   - category: 카테고리로 필터링 (optional)
func (ctrl *TagController) ListTags(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	category := c.Query("category")

	var tags interface{}
	var err error

	if category != "" {
		tags, err = ctrl.tagService.GetTagsByCategory(category)
	} else {
		tags, err = ctrl.tagService.ListTags()
	}

	if err != nil {
		log.Error("Failed to list tags", err, map[string]interface{}{
			"category": category,
		})
		apperrors.InternalError(c, "태그 조회에 실패했습니다")
		return
	}

	log.Info("Tags listed", map[string]interface{}{
		"category": category,
	})

	c.JSON(http.StatusOK, gin.H{
		"tags": tags,
	})
}
