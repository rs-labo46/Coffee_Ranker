package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"coffee-ranker/controller"
	"coffee-ranker/db"
	"coffee-ranker/middleware"
	"coffee-ranker/migrate"
	"coffee-ranker/repository"
	"coffee-ranker/router"
	"coffee-ranker/usecase"
	"coffee-ranker/validator"
)

func main() {
	cfg := loadAppConfig()

	rdb := newRedisClient(cfg)
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Println(err)
		}
	}()

	database, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.CloseDB(database); err != nil {
			log.Println(err)
		}
	}()

	if err := migrate.AutoMigrate(database); err != nil {
		log.Fatal(err)
	}
	if err := migrate.Seed(database); err != nil {
		log.Fatal(err)
	}

	userRepository := repository.NewUserRepository(database)
	refreshTokenRepository := repository.NewRefreshTokenRepository(database)
	guestSessionRepository := repository.NewGuestSessionRepository(database)
	beanRepository := repository.NewBeanRepository(database)
	articleRepository := repository.NewArticleRepository(database)
	beanArticleRepository := repository.NewBeanArticleRepository(database)
	rankTargetRepository := repository.NewRankTargetRepository(database)
	actionEventRepository := repository.NewActionEventRepository(database)
	modalDisplayLogRepository := repository.NewModalDisplayLogRepository(database)
	modalBlockLogRepository := repository.NewModalBlockLogRepository(database)
	savedItemRepository := repository.NewSavedItemRepository(database)
	ratingRepository := repository.NewRatingRepository(database)
	contentMetricRepository := repository.NewContentMetricRepository(database)
	interestProfileRepository := repository.NewInterestProfileRepository(database)
	batchRunRepository := repository.NewBatchRunRepository(database)
	auditLogRepository := repository.NewIAuditLogRepository(database)
	txManager := usecase.NewTxManager(database)

	rateLimitRepository := repository.NewIRateLimitRepository(rdb)
	eventDedupRepository := repository.NewEventDedupRepository(rdb)
	modalSuppressionRepository := repository.NewModalSuppressionRepository(rdb)
	batchLockRepository := repository.NewBatchLockRepository(rdb)

	passwordHasher := repository.NewPasswordHasher()

	tokenManager, err := repository.NewTokenManager(
		cfg.JWTSecret,
		cfg.AccessTokenTTL,
		userRepository,
	)
	if err != nil {
		log.Fatal(err)
	}

	guestKeyManager := repository.NewGuestKeyManager()

	authUsecase := usecase.NewAuthUsecase(
		userRepository,
		refreshTokenRepository,
		auditLogRepository,
		txManager,
		passwordHasher,
		tokenManager,
		cfg.RefreshTokenTTL,
	)
	guestSessionUsecase := usecase.NewGuestSessionUsecase(
		guestSessionRepository,
		guestKeyManager,
		cfg.GuestSessionTTL,
	)
	beanUsecase := usecase.NewBeanUsecase(beanRepository, beanArticleRepository)
	articleUsecase := usecase.NewArticleUsecase(articleRepository, beanArticleRepository)
	searchUsecase := usecase.NewSearchUsecase(beanRepository, articleRepository)
	eventUsecase := usecase.NewEventUsecase(actionEventRepository, rankTargetRepository, eventDedupRepository)
	savedItemUsecase := usecase.NewSavedItemUsecase(savedItemRepository, rankTargetRepository, actionEventRepository)
	ratingUsecase := usecase.NewRatingUsecase(ratingRepository, rankTargetRepository, actionEventRepository)
	rankingUsecase := usecase.NewRankingUsecase(contentMetricRepository, beanRepository, articleRepository)
	recommendationUsecase := usecase.NewRecommendationUsecase(
		contentMetricRepository,
		interestProfileRepository,
		savedItemRepository,
		actionEventRepository,
		beanRepository,
		articleRepository,
	)
	modalUsecase := usecase.NewModalUsecaseWithMetrics(
		modalDisplayLogRepository,
		modalBlockLogRepository,
		rankTargetRepository,
		contentMetricRepository,
		beanRepository,
		articleRepository,
		savedItemRepository,
		actionEventRepository,
		modalSuppressionRepository,
	)
	rankingBatchUsecase := usecase.NewRankingBatchUsecase(
		actionEventRepository,
		batchRunRepository,
		batchLockRepository,
		auditLogRepository,
		txManager,
	)
	interestBatchUsecase := usecase.NewInterestBatchUsecase(
		actionEventRepository,
		interestProfileRepository,
		batchRunRepository,
		batchLockRepository,
		auditLogRepository,
		cfg.InterestProfileTTL,
	)
	cleanupUsecase := usecase.NewCleanupUsecase(
		refreshTokenRepository,
		guestSessionRepository,
		interestProfileRepository,
	)

	adminRateLimitUsecase := usecase.NewAdminRateLimitUsecase(rateLimitRepository, auditLogRepository)
	adminBeanUsecase := usecase.NewAdminBeanUsecase(beanRepository, auditLogRepository, txManager)
	adminArticleUsecase := usecase.NewAdminArticleUsecase(articleRepository, auditLogRepository, txManager)
	adminRelationUsecase := usecase.NewAdminRelationUsecase(
		beanRepository,
		articleRepository,
		beanArticleRepository,
		auditLogRepository,
		txManager,
	)
	adminBatchUsecase := usecase.NewAdminBatchUsecase(
		batchRunRepository,
		rankingBatchUsecase,
		interestBatchUsecase,
	)
	adminAuditUsecase := usecase.NewAdminAuditUsecase(auditLogRepository)
	rateLimitUsecase := usecase.NewRateLimitUsecase(rateLimitRepository)

	cookieSameSite := parseSameSite(cfg.CookieSameSite)
	if cfg.CookieSecure {
		cookieSameSite = http.SameSiteNoneMode
	}

	cookieConfig := controller.CookieConfig{
		Secure:          cfg.CookieSecure,
		Domain:          cfg.CookieDomain,
		SameSite:        cookieSameSite,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
		GuestSessionTTL: cfg.GuestSessionTTL,
	}

	logger := middleware.NewStdLogger(log.Default())

	e, err := router.NewRouter(
		router.Controllers{
			Health:         controller.NewHealthController(database, rdb),
			CSRF:           controller.NewCSRFController(cookieConfig),
			Auth:           controller.NewAuthController(authUsecase, validator.NewAuthValidator(), cookieConfig),
			Content:        controller.NewContentController(beanUsecase, articleUsecase, validator.NewContentValidator()),
			Search:         controller.NewSearchController(searchUsecase, validator.NewSearchValidator()),
			Event:          controller.NewEventController(eventUsecase, validator.NewEventValidator()),
			Saved:          controller.NewSavedItemController(savedItemUsecase, validator.NewSavedItemValidator()),
			Rating:         controller.NewRatingController(ratingUsecase, validator.NewRatingValidator()),
			Ranking:        controller.NewRankingController(rankingUsecase, validator.NewRankingValidator()),
			Recommendation: controller.NewRecommendationController(recommendationUsecase, validator.NewRecommendationValidator()),
			Modal:          controller.NewModalController(modalUsecase, validator.NewModalValidator()),
			GuestSession:   controller.NewGuestSessionController(guestSessionUsecase, cookieConfig),
			Cleanup:        controller.NewCleanupController(cleanupUsecase),
			AdminRateLimit: controller.NewAdminRateLimitController(adminRateLimitUsecase, validator.NewAdminRateLimitValidator()),
			AdminBean:      controller.NewAdminBeanController(adminBeanUsecase, validator.NewAdminBeanValidator()),
			AdminArticle:   controller.NewAdminArticleController(adminArticleUsecase, validator.NewAdminArticleValidator()),
			AdminRelation:  controller.NewAdminRelationController(adminRelationUsecase, validator.NewAdminRelationValidator()),
			AdminBatch:     controller.NewAdminBatchController(adminBatchUsecase, validator.NewAdminBatchValidator()),
			AdminAudit:     controller.NewAdminAuditController(adminAuditUsecase, validator.NewAdminAuditValidator()),
		},
		router.Middlewares{
			Recover:       middleware.RecoverMiddleware(logger),
			RequestID:     middleware.RequestIDMiddleware(),
			ClientIPHash:  middleware.ClientIPHashMiddleware(cfg.IPHashSecret),
			SecureHeaders: middleware.SecureHeadersMiddleware(),
			CORS: middleware.CORSMiddleware(middleware.CORSConfig{
				AllowOrigins:     cfg.FrontendOrigins,
				AllowCredentials: true,
			}),
			BodyLimit:    middleware.BodyLimitMiddleware(cfg.BodyLimitBytes),
			AccessLog:    middleware.AccessLogMiddleware(logger),
			Auth:         middleware.AuthMiddleware(tokenManager),
			OptionalAuth: middleware.OptionalAuthMiddleware(tokenManager),
			Admin:        middleware.AdminMiddleware(),
			GuestSession: middleware.GuestSessionMiddleware(
				guestSessionUsecase,
				middleware.GuestSessionCookieConfig{
					Name:     controller.GuestSessionCookieName,
					MaxAge:   cfg.GuestSessionTTL,
					Secure:   cfg.CookieSecure,
					SameSite: cookieSameSite,
				},
			),
			Actor:                   middleware.ActorMiddleware(),
			CSRF:                    middleware.CSRFMiddleware(),
			RateLimitPublicRead:     middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitPublicRead),
			RateLimitAuthIP:         middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitAuthIP),
			RateLimitRefresh:        middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitRefresh),
			RateLimitUser:           middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitUser),
			RateLimitSearch:         middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitSearch),
			RateLimitEvent:          middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitEvent),
			RateLimitRecommendation: middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitRecommendation),
			RateLimitModal:          middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitModal),
			RateLimitAdmin:          middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitAdmin),
			RateLimitAdminBatch:     middleware.RateLimitMiddleware(rateLimitUsecase, middleware.RateLimitAdminBatch),
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	startDailyRankingBatch(ctx, rankingBatchUsecase, log.Default())

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal(err)
		}
	}()

	<-ctx.Done()
	e.Logger.Info("shutdown server")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}

func parseSameSite(value string) http.SameSite {
	switch value {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
