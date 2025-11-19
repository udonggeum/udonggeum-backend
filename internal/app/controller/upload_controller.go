package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type UploadController struct {
	uploadDir string
	baseURL   string
}

func NewUploadController(uploadDir, baseURL string) *UploadController {
	return &UploadController{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}
}

// UploadImage handles image file uploads
func (ctrl *UploadController) UploadImage(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		log.Warn("No file in request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No file provided",
		})
		return
	}

	// Validate file size (max 5MB)
	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		log.Warn("File too large", map[string]interface{}{
			"size":     file.Size,
			"max_size": maxFileSize,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File size exceeds 5MB limit",
		})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	if !allowedExts[ext] {
		log.Warn("Invalid file type", map[string]interface{}{
			"filename":  file.Filename,
			"extension": ext,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only image files (jpg, jpeg, png, gif, webp) are allowed",
		})
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filepath := filepath.Join(ctrl.uploadDir, filename)

	// Ensure upload directory exists
	if err := os.MkdirAll(ctrl.uploadDir, 0755); err != nil {
		log.Error("Failed to create upload directory", err, map[string]interface{}{
			"dir": ctrl.uploadDir,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
		})
		return
	}

	// Save file
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		log.Error("Failed to save uploaded file", err, map[string]interface{}{
			"filename": filename,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
		})
		return
	}

	// Generate URL
	fileURL := fmt.Sprintf("%s/uploads/%s", ctrl.baseURL, filename)

	log.Info("File uploaded successfully", map[string]interface{}{
		"filename":     filename,
		"original":     file.Filename,
		"size":         file.Size,
		"url":          fileURL,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":    "File uploaded successfully",
		"url":        fileURL,
		"filename":   filename,
		"size_bytes": file.Size,
	})
}
