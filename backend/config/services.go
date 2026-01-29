package config

import (
	"log"
	"time"

	"backend/controller"
	"backend/middleware"
	"backend/repository"
	"backend/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Services 包含所有初始化的服务
type Services struct {
	AuthService     service.AuthService
	ContentService  service.ContentService
	PurchaseService service.PurchaseService
	StripeService   service.StripeService
	PayPalService   service.PayPalService
}

// Controllers 包含所有初始化的控制器
type Controllers struct {
	AuthController     *controller.AuthController
	ContentController  *controller.ContentController
	PurchaseController *controller.PurchaseController
}

// Repositories 包含所有初始化的仓库
type Repositories struct {
	UserRepo           repository.UserRepository
	SessionRepo        repository.SessionRepository
	UserPermissionRepo repository.UserPermissionRepository
	WalletRepo         repository.WalletRepository
	PurchaseRepo       repository.PurchaseRepository
	ContentRepo        repository.ContentRepository
	ContentFileRepo    repository.ContentFileRepository
	TagRepo            repository.TagRepository
	ContentTagRepo     repository.ContentTagRepository
	ContentBookmarkRepo repository.ContentBookmarkRepository
	AnalyticsRepo      repository.AnalyticsRepository
}

// InitRepositories 初始化所有仓库
func InitRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		UserRepo:           repository.NewUserRepository(db),
		SessionRepo:        repository.NewSessionRepository(db),
		UserPermissionRepo: repository.NewUserPermissionRepository(db),
		WalletRepo:         repository.NewWalletRepository(db),
		PurchaseRepo:       repository.NewPurchaseRepository(db),
		ContentRepo:        repository.NewContentRepository(db),
		ContentFileRepo:    repository.NewContentFileRepository(db),
		TagRepo:            repository.NewTagRepository(db),
		ContentTagRepo:      repository.NewContentTagRepository(db),
		ContentBookmarkRepo: repository.NewContentBookmarkRepository(db),
		AnalyticsRepo:       repository.NewAnalyticsRepository(db),
	}
}

// InitServices 初始化所有服务
func InitServices(db *gorm.DB, repos *Repositories) (*Services, error) {
	services := &Services{}

	// 初始化认证服务（必需）
	permissionRepo := repository.NewPermissionRepository(db)
	services.AuthService = service.NewAuthService(
		db,
		repos.UserRepo,
		repos.SessionRepo,
		repos.WalletRepo,
		repos.PurchaseRepo,
		repos.UserPermissionRepo,
		permissionRepo,
	)

	// 初始化 S3 服务（可选）
	s3Service, err := service.NewS3Service()
	if err != nil {
		log.Printf("Warning: Failed to initialize S3 service: %v. Video upload will not work.", err)
		s3Service = nil
	}

	// 初始化 Kafka 服务（可选）
	kafkaService, err := service.NewKafkaService()
	if err != nil {
		log.Printf("Warning: Failed to initialize Kafka service: %v. Video processing will not work.", err)
		kafkaService = nil
	}

	// 初始化内容服务（如果 S3 和 Kafka 可用）
	if s3Service != nil && kafkaService != nil {
		services.ContentService = service.NewContentService(
			repos.ContentRepo,
			repos.ContentFileRepo,
			repos.TagRepo,
			repos.ContentTagRepo,
			repos.ContentBookmarkRepo,
			repos.AnalyticsRepo,
			repos.PurchaseRepo,
			repos.UserRepo,
			s3Service,
			kafkaService,
		)
	}

	// 初始化 Stripe 服务（可选）
	stripeService, err := service.NewStripeService()
	if err != nil {
		log.Printf("Warning: Failed to initialize Stripe service: %v. Purchase features will not work.", err)
		stripeService = nil
	} else {
		services.StripeService = stripeService
	}

	// 初始化 PayPal 服务（可选，备用支付）
	paypalService, err := service.NewPayPalService()
	if err != nil {
		log.Printf("Warning: Failed to initialize PayPal service: %v. PayPal payment will not work.", err)
		paypalService = nil
	} else {
		services.PayPalService = paypalService
	}

	// 初始化购买服务（需要至少 Stripe 或 PayPal 之一；当前以 Stripe 为主）
	if stripeService != nil {
		services.PurchaseService = service.NewPurchaseService(
			stripeService,
			paypalService,
			repos.PurchaseRepo,
			repos.ContentRepo,
			repos.WalletRepo,
			repos.AnalyticsRepo,
		)
	}

	// 启动 Kafka 消费者（如果服务可用）
	if kafkaService != nil && s3Service != nil {
		videoProcessingService, err := service.NewVideoProcessingService()
		if err != nil {
			log.Printf("Warning: Failed to initialize video processing service: %v. Video processing will not work.", err)
		} else if repos.ContentFileRepo != nil {
			videoProcessorConsumer := service.NewVideoProcessorConsumer(
				repos.ContentFileRepo,
				repos.ContentRepo,
				s3Service,
				videoProcessingService,
				kafkaService,
			)
			if err := videoProcessorConsumer.Start(); err != nil {
				log.Printf("Warning: Failed to start video processor consumer: %v", err)
			} else {
				log.Println("Video processor consumer started successfully")
			}
		}
	}

	return services, nil
}

// InitControllers 初始化所有控制器
func InitControllers(services *Services, repos *Repositories) *Controllers {
	controllers := &Controllers{}

	// 初始化认证控制器（必需）
	controllers.AuthController = controller.NewAuthController(services.AuthService)

	// 初始化内容控制器（如果内容服务可用）
	if services.ContentService != nil {
		controllers.ContentController = controller.NewContentController(services.ContentService, repos.UserPermissionRepo, repos.PurchaseRepo)
	}

	// 初始化购买控制器（如果购买服务和 Stripe 服务可用）
	if services.PurchaseService != nil && services.StripeService != nil {
		controllers.PurchaseController = controller.NewPurchaseController(
			services.PurchaseService,
			services.StripeService,
		)
	}

	return controllers
}

// InitMiddleware 初始化中间件
func InitMiddleware(redisClient *redis.Client) *middleware.CacheMiddleware {
	// 初始化 Redis 限流器
	middleware.DefaultRateLimiter = middleware.NewRedisRateLimiter(redisClient, 100, 1*time.Minute)
	middleware.StrictRateLimiter = middleware.NewRedisRateLimiter(redisClient, 10, 1*time.Minute)
	middleware.AuthRateLimiter = middleware.NewRedisRateLimiter(redisClient, 20, 1*time.Minute)
	middleware.RefreshRateLimiter = middleware.NewRedisRateLimiter(redisClient, 60, 1*time.Minute)

	// 初始化 Redis 缓存中间件
	cacheMiddleware := middleware.NewCacheMiddleware(redisClient, "api_cache:", 5*time.Minute)
	middleware.GlobalCacheMiddleware = cacheMiddleware

	return cacheMiddleware
}
