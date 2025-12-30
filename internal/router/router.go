package router

import (
	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/internal/app/controller"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type Router struct {
	authController      *controller.AuthController
	storeController     *controller.StoreController
	goldPriceController *controller.GoldPriceController
	communityController *controller.CommunityController
	reviewController    *controller.ReviewController
	uploadController    *controller.UploadController
	tagController       *controller.TagController
	chatController      *controller.ChatController
	authMiddleware      *middleware.AuthMiddleware
	config              *config.Config
}

func NewRouter(
	authController *controller.AuthController,
	storeController *controller.StoreController,
	goldPriceController *controller.GoldPriceController,
	communityController *controller.CommunityController,
	reviewController *controller.ReviewController,
	uploadController *controller.UploadController,
	tagController *controller.TagController,
	chatController *controller.ChatController,
	authMiddleware *middleware.AuthMiddleware,
	cfg *config.Config,
) *Router {
	return &Router{
		authController:      authController,
		storeController:     storeController,
		goldPriceController: goldPriceController,
		communityController: communityController,
		reviewController:    reviewController,
		uploadController:    uploadController,
		tagController:       tagController,
		chatController:      chatController,
		authMiddleware:      authMiddleware,
		config:              cfg,
	}
}

func (r *Router) Setup() *gin.Engine {
	gin.SetMode(r.config.Server.GinMode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware())
	router.Use(corsMiddleware(r.config.CORS.AllowedOrigins))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"message": "UDONGGEUM API is running",
		})
	})

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", r.authController.Register)
			auth.POST("/login", r.authController.Login)
			auth.POST("/logout", r.authController.Logout)
			auth.POST("/refresh", r.authController.RefreshToken)
			auth.POST("/forgot-password", r.authController.ForgotPassword)
			auth.POST("/reset-password", r.authController.ResetPassword)
			auth.POST("/check-nickname", r.authController.CheckNickname)
			auth.GET("/me", r.authMiddleware.Authenticate(), r.authController.GetMe)
			auth.PUT("/me", r.authMiddleware.Authenticate(), r.authController.UpdateMe)

			// Kakao OAuth
			auth.GET("/kakao/login", r.authController.GetKakaoLoginURL)
			auth.GET("/kakao/callback", r.authController.KakaoCallback)
		}

		stores := v1.Group("/stores")
		{
			stores.GET("", r.authMiddleware.OptionalAuthenticate(), r.storeController.ListStores)
			stores.GET("/locations", r.storeController.ListLocations)
			stores.GET("/:id", r.authMiddleware.OptionalAuthenticate(), r.storeController.GetStoreByID)
			stores.POST("",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.storeController.CreateStore,
			)
			stores.PUT("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.storeController.UpdateStore,
			)
			stores.DELETE("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.storeController.DeleteStore,
			)

			// Store like
			stores.POST("/:id/like",
				r.authMiddleware.Authenticate(),
				r.storeController.ToggleStoreLike,
			)

			// Store reviews
			stores.POST("/:id/reviews",
				r.authMiddleware.Authenticate(),
				r.reviewController.CreateReview,
			)
			stores.GET("/:id/reviews", r.reviewController.GetStoreReviews)

			// Store statistics
			stores.GET("/:id/stats", r.reviewController.GetStoreStatistics)

			// Store gallery
			stores.GET("/:id/gallery", r.reviewController.GetStoreGallery)
		}

		// Users routes
		users := v1.Group("/users")
		{
			users.GET("/me/reviews",
				r.authMiddleware.Authenticate(),
				r.reviewController.GetUserReviews,
			)
			users.GET("/me/liked-stores",
				r.authMiddleware.Authenticate(),
				r.storeController.GetUserLikedStores,
			)
			users.GET("/me/store",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.storeController.GetMyStore,
			)
			users.PUT("/me/store",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.storeController.UpdateMyStore,
			)
		}

		// Reviews routes
		reviews := v1.Group("/reviews")
		{
			reviews.PUT("/:id",
				r.authMiddleware.Authenticate(),
				r.reviewController.UpdateReview,
			)
			reviews.DELETE("/:id",
				r.authMiddleware.Authenticate(),
				r.reviewController.DeleteReview,
			)
			reviews.POST("/:id/like",
				r.authMiddleware.Authenticate(),
				r.reviewController.ToggleReviewLike,
			)
		}

		goldPrices := v1.Group("/gold-prices")
		{
			// Public routes
			goldPrices.GET("/latest", r.goldPriceController.GetLatestPrices)
			goldPrices.GET("/type/:type", r.goldPriceController.GetPriceByType)
			goldPrices.GET("/history/:type", r.goldPriceController.GetPriceHistory)

			// Admin routes
			goldPrices.POST("",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.goldPriceController.CreatePrice,
			)
			goldPrices.PUT("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.goldPriceController.UpdatePrice,
			)
			goldPrices.POST("/update",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.goldPriceController.UpdateFromExternalAPI,
			)
		}

		// Upload routes
		upload := v1.Group("/upload")
		{
			upload.POST("/presigned-url",
				r.authMiddleware.Authenticate(),
				r.uploadController.GeneratePresignedURL,
			)
			upload.POST("/chat/presigned-url",
				r.authMiddleware.Authenticate(),
				r.uploadController.GenerateChatFilePresignedURL,
			)
		}

		// Tags routes
		tags := v1.Group("/tags")
		{
			tags.GET("", r.tagController.ListTags) // 태그 목록 조회 (카테고리 필터 가능)
		}

		// Chat routes
		chats := v1.Group("/chats")
		{
			// WebSocket 연결
			chats.GET("/ws",
				r.authMiddleware.Authenticate(),
				r.chatController.WebSocketHandler,
			)

			// 메시지 검색
			chats.GET("/search",
				r.authMiddleware.Authenticate(),
				r.chatController.SearchMessages,
			)

			// 채팅방 관련
			rooms := chats.Group("/rooms")
			rooms.Use(r.authMiddleware.Authenticate())
			{
				rooms.POST("", r.chatController.CreateChatRoom)                               // 채팅방 생성
				rooms.GET("", r.chatController.GetChatRooms)                                  // 채팅방 목록
				rooms.GET("/:id", r.chatController.GetChatRoom)                               // 채팅방 상세
				rooms.POST("/:id/join", r.chatController.JoinRoom)                            // 채팅방 참여
				rooms.POST("/:id/leave", r.chatController.LeaveRoom)                          // 채팅방 나가기
				rooms.POST("/:id/read", r.chatController.MarkAsRead)                          // 읽음 처리
				rooms.GET("/:id/messages", r.chatController.GetMessages)                      // 메시지 목록
				rooms.POST("/:id/messages", r.chatController.SendMessage)                     // 메시지 전송
				rooms.PATCH("/:id/messages/:messageId", r.chatController.UpdateMessage)       // 메시지 수정
				rooms.DELETE("/:id/messages/:messageId", r.chatController.DeleteMessage)      // 메시지 삭제
			}
		}

		// Community (금광산) routes
		community := v1.Group("/community")
		{
			// AI Content Generation
			community.POST("/generate-content",
				r.authMiddleware.Authenticate(),
				r.communityController.GenerateContent,
			)

			// Post routes
			posts := community.Group("/posts")
			{
				// Public routes (일부는 인증 선택)
				posts.GET("", r.communityController.GetPosts)               // 게시글 목록 (필터링)
				posts.GET("/:id", r.communityController.GetPost)            // 게시글 상세 조회

				// Authenticated routes
				posts.POST("",
					r.authMiddleware.Authenticate(),
					r.communityController.CreatePost,
				)
				posts.PUT("/:id",
					r.authMiddleware.Authenticate(),
					r.communityController.UpdatePost,
				)
				posts.DELETE("/:id",
					r.authMiddleware.Authenticate(),
					r.communityController.DeletePost,
				)

				// Like
				posts.POST("/:id/like",
					r.authMiddleware.Authenticate(),
					r.communityController.TogglePostLike,
				)

				// QnA - Accept answer
				posts.POST("/:id/accept/:comment_id",
					r.authMiddleware.Authenticate(),
					r.communityController.AcceptAnswer,
				)

				// Pin/Unpin
				posts.POST("/:id/pin",
					r.authMiddleware.Authenticate(),
					r.communityController.PinPost,
				)
				posts.POST("/:id/unpin",
					r.authMiddleware.Authenticate(),
					r.communityController.UnpinPost,
				)
			}

			// Gallery route
			community.GET("/gallery", r.communityController.GetStoreGallery)

			// Comment routes
			comments := community.Group("/comments")
			{
				// Public routes
				comments.GET("", r.communityController.GetComments)         // 댓글 목록

				// Authenticated routes
				comments.POST("",
					r.authMiddleware.Authenticate(),
					r.communityController.CreateComment,
				)
				comments.PUT("/:id",
					r.authMiddleware.Authenticate(),
					r.communityController.UpdateComment,
				)
				comments.DELETE("/:id",
					r.authMiddleware.Authenticate(),
					r.communityController.DeleteComment,
				)

				// Like
				comments.POST("/:id/like",
					r.authMiddleware.Authenticate(),
					r.communityController.ToggleCommentLike,
				)
			}
		}
	}

	return router
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
