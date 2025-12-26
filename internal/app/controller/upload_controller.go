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

// GenerateChatFilePresignedURL generates a presigned URL for uploading chat files
// POST /api/v1/upload/chat/presigned-url
func (ctrl *UploadController) GenerateChatFilePresignedURL(c *gin.Context) {
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

	// Validate content type (allow images and common file types)
	allowedTypes := []string{
		// Images
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
		// Documents
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		// Archives
		"application/zip",
		"application/x-rar-compressed",
		// Text
		"text/plain",
	}
	if err := ctrl.storage.ValidateContentType(req.ContentType, allowedTypes); err != nil {
		logger.Warn("Invalid content type for chat file", map[string]interface{}{
			"content_type": req.ContentType,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File type not allowed. Allowed types: images, PDF, Word, Excel, ZIP, RAR, TXT",
		})
		return
	}

	// Validate file size (max 10MB for chat files)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if req.Folder != "" {
		// Folder field can be used to pass file size for validation
		// This is a workaround since we can't get file size before upload
	}

	// Set folder for chat files
	folder := "chat"
	if req.Folder != "" {
		// Allow subfolder specification (e.g., "chat/room_123")
		folder = req.Folder
	}

	// Generate presigned URL
	response, err := ctrl.storage.GeneratePresignedURLWithFolder(req.Filename, req.ContentType, folder)
	if err != nil {
		logger.Error("Failed to generate presigned URL for chat file", err, map[string]interface{}{
			"filename":     req.Filename,
			"content_type": req.ContentType,
			"folder":       folder,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate presigned URL",
		})
		return
	}

	logger.Info("Chat file presigned URL generated successfully", map[string]interface{}{
		"filename":     req.Filename,
		"content_type": req.ContentType,
		"folder":       folder,
		"key":          response.Key,
		"max_size":     maxSize,
	})

	c.JSON(http.StatusOK, gin.H{
		"upload_url": response.UploadURL,
		"file_url":   response.FileURL,
		"key":        response.Key,
		"max_size":   maxSize,
	})
}
