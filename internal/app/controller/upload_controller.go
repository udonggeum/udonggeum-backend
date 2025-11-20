package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	"github.com/ikkim/udonggeum-backend/internal/storage"
)

type UploadController struct {
	s3Storage *storage.S3Storage
}

func NewUploadController(s3Storage *storage.S3Storage) *UploadController {
	return &UploadController{
		s3Storage: s3Storage,
	}
}

type GeneratePresignedURLRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required"`
}

// GeneratePresignedURL generates a pre-signed URL for direct S3 upload
func (ctrl *UploadController) GeneratePresignedURL(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req GeneratePresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Validate file size (max 5MB)
	const maxFileSize = 5 * 1024 * 1024
	if err := ctrl.s3Storage.ValidateFileSize(req.FileSize, maxFileSize); err != nil {
		log.Warn("File size validation failed", map[string]interface{}{
			"size":     req.FileSize,
			"max_size": maxFileSize,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File size exceeds 5MB limit",
		})
		return
	}

	// Validate content type
	allowedTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	if err := ctrl.s3Storage.ValidateContentType(req.ContentType, allowedTypes); err != nil {
		log.Warn("Content type validation failed", map[string]interface{}{
			"content_type": req.ContentType,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only image files (jpeg, png, gif, webp) are allowed",
		})
		return
	}

	// Generate presigned URL
	result, err := ctrl.s3Storage.GeneratePresignedURL(req.Filename, req.ContentType)
	if err != nil {
		log.Error("Failed to generate presigned URL", err, map[string]interface{}{
			"filename": req.Filename,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate upload URL",
		})
		return
	}

	log.Info("Presigned URL generated successfully", map[string]interface{}{
		"filename": req.Filename,
		"key":      result.Key,
	})

	c.JSON(http.StatusOK, gin.H{
		"upload_url": result.UploadURL,
		"file_url":   result.FileURL,
		"key":        result.Key,
	})
}
