package router

import (
	"net/http"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/handler"
	"ecommerce-api/internal/middleware"
	"ecommerce-api/internal/payment"
	"ecommerce-api/internal/repository"
	"ecommerce-api/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	userRepo := repository.NewUserRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	gateway := payment.NewMockGateway(cfg.PaymentMockBaseURL)
	authSvc := service.NewAuthService(userRepo, cfg)
	userSvc := service.NewUserService(userRepo, addressRepo)
	productSvc := service.NewProductService(productRepo, categoryRepo)
	cartSvc := service.NewCartService(db, cartRepo, productRepo)
	orderSvc := service.NewOrderService(db, orderRepo, cartRepo, productRepo, paymentRepo, addressRepo, gateway, cfg)
	paymentSvc := service.NewPaymentService(db, paymentRepo, orderRepo, productRepo)

	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(userSvc)
	productH := handler.NewProductHandler(productSvc)
	categoryH := handler.NewCategoryHandler(productSvc)
	cartH := handler.NewCartHandler(cartSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	paymentH := handler.NewPaymentHandler(paymentSvc, cfg.WebhookSecret)

	// 20 req/detik per IP untuk seluruh API, 5 login/menit per IP.
	publicLimiter := middleware.NewRateLimiter(20, 40)
	loginLimiter := middleware.NewRateLimiter(5.0/60.0, 5)

	r := gin.Default()
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ecommerce-api"})
	})
	// File gambar produk (salinan web, di-sync dari VM Dhodi).
	// Direktori ./uploads relatif terhadap WorkingDirectory service.
	r.Static("/uploads", "./uploads")

	v1 := r.Group("/api/v1")
	v1.Use(publicLimiter.Limit("terlalu banyak permintaan, coba lagi nanti"))
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ecommerce-api"})
	})

	// Public
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/login", loginLimiter.Limit("terlalu banyak percobaan login, coba lagi nanti"), authH.Login)
	v1.POST("/auth/refresh", authH.Refresh)
	v1.GET("/products", productH.ListProducts)
	v1.GET("/products/:id", productH.GetProduct)
	v1.GET("/categories", categoryH.ListCategories)
	v1.GET("/categories/:id", categoryH.GetCategory)
	v1.GET("/mock-pay/:transaction_id", paymentH.MockPayPage)
	v1.POST("/payments/webhook", paymentH.Webhook)

	// Protected
	auth := v1.Group("")
	auth.Use(middleware.AuthMiddleware(cfg.JWTAccessSecret))
	auth.POST("/auth/logout", authH.Logout)
	auth.GET("/users/me", userH.GetProfile)
	auth.PUT("/users/me", userH.UpdateProfile)
	auth.GET("/users/me/addresses", userH.ListAddresses)
	auth.POST("/users/me/addresses", userH.CreateAddress)
	auth.DELETE("/users/me/addresses/:id", userH.DeleteAddress)
	auth.PUT("/users/me/addresses/:id/default", userH.SetDefaultAddress)
	auth.GET("/cart", cartH.GetCart)
	auth.POST("/cart/items", cartH.AddItem)
	auth.PUT("/cart/items/:id", cartH.UpdateItemQty)
	auth.DELETE("/cart/items/:id", cartH.RemoveItem)
	auth.POST("/orders/checkout", orderH.Checkout)
	auth.GET("/orders", orderH.ListOrders)
	auth.GET("/orders/:id", orderH.GetOrder)
	auth.POST("/orders/:id/cancel", orderH.CancelOrder)

	// Admin
	admin := v1.Group("")
	admin.Use(middleware.AuthMiddleware(cfg.JWTAccessSecret))
	admin.Use(middleware.RequireRole(domain.RoleAdmin))
	admin.POST("/products", productH.CreateProduct)
	admin.PUT("/products/:id", productH.UpdateProduct)
	admin.DELETE("/products/:id", productH.DeleteProduct)
	admin.POST("/categories", categoryH.CreateCategory)
	admin.PUT("/categories/:id", categoryH.UpdateCategory)
	admin.DELETE("/categories/:id", categoryH.DeleteCategory)
	admin.GET("/admin/orders", orderH.ListAllOrders)
	admin.PATCH("/admin/orders/:id/status", orderH.UpdateOrderStatus)

	return r
}
