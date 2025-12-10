package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/storage"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

type UploadController struct {
	storage *storage.S3Storage
}

func NewUploadController(storage *storage.S3Storage) *UploadController {
	return &UploadController{
		storage: storage,
	}
}

type GeneratePresignedURLRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Folder      string `json:"folder"` // Optional: defaults to "uploads"
}

// GeneratePresignedURL generates a presigned URL for uploading files to S3
// POST /api/v1/upload/presigned-url
func (ctrl *UploadController) GeneratePresignedURL(c *gin.Context) {
	var req GeneratePresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid presigned URL request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Validate content type (only allow images)
	allowedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	if err := ctrl.storage.ValidateContentType(req.ContentType, allowedTypes); err != nil {
		logger.Warn("Invalid content type", map[string]interface{}{
			"content_type": req.ContentType,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only image files are allowed (JPEG, PNG, GIF, WEBP)",
		})
		return
	}

	// Set default folder if not provided
	folder := req.Folder
	if folder == "" {
		folder = "community"
	}

	// Generate presigned URL
	response, err := ctrl.storage.GeneratePresignedURLWithFolder(req.Filename, req.ContentType, folder)
	if err != nil {
		logger.Error("Failed to generate presigned URL", err, map[string]interface{}{
			"filename":     req.Filename,
			"content_type": req.ContentType,
			"folder":       folder,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate presigned URL",
		})
		return
	}

	logger.Info("Presigned URL generated successfully", map[string]interface{}{
		"filename":     req.Filename,
		"content_type": req.ContentType,
		"folder":       folder,
		"key":          response.Key,
	})

	c.JSON(http.StatusOK, gin.H{
		"upload_url": response.UploadURL,
		"file_url":   response.FileURL,
		"key":        response.Key,
	})
}
