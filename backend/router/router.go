package router

import (
	"net/http"

	"coffee-ranker/controller"
	"coffee-ranker/middleware"

	"github.com/labstack/echo/v4"
)

type Controllers struct {
	Health         *controller.HealthController
	CSRF           *controller.CSRFController
	GuestSession   *controller.GuestSessionController
	Auth           *controller.AuthController
	Content        *controller.ContentController
	Search         *controller.SearchController
	Event          *controller.EventController
	Saved          *controller.SavedItemController
	Rating         *controller.RatingController
	Ranking        *controller.RankingController
	Recommendation *controller.RecommendationController
	Modal          *controller.ModalController
	AdminBean      *controller.AdminBeanController
	AdminArticle   *controller.AdminArticleController
	AdminRelation  *controller.AdminRelationController
	AdminBatch     *controller.AdminBatchController
	AdminAudit     *controller.AdminAuditController
	Cleanup        *controller.CleanupController
	AdminRateLimit *controller.AdminRateLimitController
}

type Middlewares struct {
	Recover                 echo.MiddlewareFunc
	RequestID               echo.MiddlewareFunc
	ClientIPHash            echo.MiddlewareFunc
	SecureHeaders           echo.MiddlewareFunc
	CORS                    echo.MiddlewareFunc
	BodyLimit               echo.MiddlewareFunc
	AccessLog               echo.MiddlewareFunc
	Auth                    echo.MiddlewareFunc
	OptionalAuth            echo.MiddlewareFunc
	Admin                   echo.MiddlewareFunc
	GuestSession            echo.MiddlewareFunc
	Actor                   echo.MiddlewareFunc
	CSRF                    echo.MiddlewareFunc
	RateLimitPublicRead     echo.MiddlewareFunc
	RateLimitAuthIP         echo.MiddlewareFunc
	RateLimitRefresh        echo.MiddlewareFunc
	RateLimitUser           echo.MiddlewareFunc
	RateLimitSearch         echo.MiddlewareFunc
	RateLimitEvent          echo.MiddlewareFunc
	RateLimitRecommendation echo.MiddlewareFunc
	RateLimitModal          echo.MiddlewareFunc
	RateLimitAdmin          echo.MiddlewareFunc
	RateLimitAdminBatch     echo.MiddlewareFunc
}

func NewRouter(controllers Controllers, middlewares Middlewares) (*echo.Echo, error) {
	e := echo.New()
	if err := Register(e, controllers, middlewares); err != nil {
		return nil, err
	}
	return e, nil
}

func Register(e *echo.Echo, controllers Controllers, middlewares Middlewares) error {
	if e == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "echo is nil")
	}
	if err := validateDeps(controllers, middlewares); err != nil {
		return err
	}

	registerGlobalMiddlewares(e, middlewares)
	registerHealthRoutes(e, controllers)
	registerPublicRoutes(e, controllers, middlewares)
	registerAuthRoutes(e, controllers, middlewares)
	registerUserRoutes(e, controllers, middlewares)
	registerActorRoutes(e, controllers, middlewares)
	registerAdminRoutes(e, controllers, middlewares)
	return nil
}

func registerGlobalMiddlewares(e *echo.Echo, mw Middlewares) {
	e.Use(mw.Recover)
	e.Use(mw.RequestID)
	e.Use(mw.ClientIPHash)
	e.Use(syncControllerContext())
	e.Use(mw.SecureHeaders)
	e.Use(mw.CORS)
	e.Use(mw.BodyLimit)
	e.Use(mw.AccessLog)
}

func registerHealthRoutes(e *echo.Echo, c Controllers) {
	e.GET("/health", c.Health.Health)
	e.GET("/ready", c.Health.Ready)
}

func registerPublicRoutes(e *echo.Echo, c Controllers, mw Middlewares) {
	e.GET("/auth/csrf", c.CSRF.Issue, mw.RateLimitPublicRead)
	e.POST("/guest-session", c.GuestSession.GetOrCreate, mw.RateLimitPublicRead)

	e.GET("/beans", c.Content.ListBeans, mw.RateLimitPublicRead)
	e.GET("/beans/:id", c.Content.GetBean, mw.RateLimitPublicRead)
	e.GET("/beans/:id/articles", c.Content.RelatedArticles, mw.RateLimitPublicRead)

	e.GET("/articles", c.Content.ListArticles, mw.RateLimitPublicRead)
	e.GET("/rankings", c.Ranking.List, mw.RateLimitPublicRead)
	e.GET("/rankings/top", c.Ranking.Top, mw.RateLimitPublicRead)
}

func registerAuthRoutes(e *echo.Echo, c Controllers, mw Middlewares) {
	e.POST("/auth/signup", c.Auth.Signup, mw.RateLimitAuthIP)
	e.POST("/auth/login", c.Auth.Login, mw.RateLimitAuthIP)
	e.POST("/auth/refresh", c.Auth.Refresh, mw.RateLimitRefresh, mw.CSRF)
	e.POST("/auth/logout", c.Auth.Logout, mw.Auth, syncControllerContext(), mw.RateLimitUser, mw.CSRF)
	e.POST("/auth/logout-all", c.Auth.LogoutAllDevices, mw.Auth, syncControllerContext(), mw.RateLimitUser, mw.CSRF)
	e.GET("/auth/me", c.Auth.Me, mw.Auth, syncControllerContext(), mw.RateLimitUser)
}

func registerUserRoutes(e *echo.Echo, c Controllers, mw Middlewares) {
	user := []echo.MiddlewareFunc{mw.Auth, syncControllerContext(), mw.RateLimitUser}
	userWithCSRF := []echo.MiddlewareFunc{mw.Auth, syncControllerContext(), mw.RateLimitUser, mw.CSRF}

	e.GET("/articles/:slug", c.Content.GetArticleBySlug, user...)
	e.GET("/articles/id/:id", c.Content.GetArticleByID, user...)
	e.GET("/articles/id/:id/beans", c.Content.RelatedBeans, user...)

	e.GET("/saved", c.Saved.List, user...)
	e.POST("/saved", c.Saved.Save, userWithCSRF...)
	e.GET("/saved/:rank_target_id", c.Saved.Exists, user...)
	e.DELETE("/saved/:rank_target_id", c.Saved.Remove, userWithCSRF...)

	e.GET("/ratings", c.Rating.List, user...)
	e.POST("/ratings", c.Rating.Rate, userWithCSRF...)
	e.GET("/ratings/:rank_target_id", c.Rating.Get, user...)
	e.DELETE("/ratings/:rank_target_id", c.Rating.Delete, userWithCSRF...)
}

func registerActorRoutes(e *echo.Echo, c Controllers, mw Middlewares) {
	actor := []echo.MiddlewareFunc{mw.OptionalAuth, mw.GuestSession, mw.Actor, syncControllerContext()}

	e.GET("/search/beans", c.Search.SearchBeans, appendMiddlewares(actor, mw.RateLimitSearch)...)
	e.GET("/search/articles", c.Search.SearchArticles, appendMiddlewares(actor, mw.RateLimitSearch)...)
	e.GET("/recommendations", c.Recommendation.List, appendMiddlewares(actor, mw.RateLimitRecommendation)...)
	e.POST("/events", c.Event.Record, appendMiddlewares(actor, mw.RateLimitEvent, mw.CSRF)...)
	e.POST("/modals", c.Modal.Show, appendMiddlewares(actor, mw.RateLimitModal, mw.CSRF)...)
	e.POST("/modals/click", c.Modal.Click, appendMiddlewares(actor, mw.RateLimitModal, mw.CSRF)...)
	e.POST("/modals/close", c.Modal.Close, appendMiddlewares(actor, mw.RateLimitModal, mw.CSRF)...)
}

func registerAdminRoutes(e *echo.Echo, c Controllers, mw Middlewares) {
	admin := e.Group("/admin", mw.Auth, mw.Admin, syncControllerContext())
	adminWithCSRF := []echo.MiddlewareFunc{mw.RateLimitAdmin, mw.CSRF}
	adminBatchWithCSRF := []echo.MiddlewareFunc{mw.RateLimitAdminBatch, mw.CSRF}

	admin.POST("/beans", c.AdminBean.Create, adminWithCSRF...)
	admin.PUT("/beans/:id", c.AdminBean.Update, adminWithCSRF...)
	admin.PATCH("/beans/:id/publish", c.AdminBean.Publish, adminWithCSRF...)
	admin.PATCH("/beans/:id/unpublish", c.AdminBean.Unpublish, adminWithCSRF...)

	admin.POST("/articles", c.AdminArticle.Create, adminWithCSRF...)
	admin.PUT("/articles/:id", c.AdminArticle.Update, adminWithCSRF...)
	admin.PATCH("/articles/:id/publish", c.AdminArticle.Publish, adminWithCSRF...)
	admin.PATCH("/articles/:id/unpublish", c.AdminArticle.Unpublish, adminWithCSRF...)

	admin.POST("/bean-articles", c.AdminRelation.Create, adminWithCSRF...)
	admin.DELETE("/bean-articles/:bean_id/:article_id", c.AdminRelation.Delete, adminWithCSRF...)
	admin.PUT("/beans/:bean_id/articles", c.AdminRelation.ReplaceByBeanID, adminWithCSRF...)
	admin.PATCH("/beans/:bean_id/articles/order", c.AdminRelation.UpdateDisplayOrder, adminWithCSRF...)

	admin.POST("/batches/ranking", c.AdminBatch.RunRanking, adminBatchWithCSRF...)
	admin.POST("/batches/interest", c.AdminBatch.RunInterest, adminBatchWithCSRF...)
	admin.GET("/batches", c.AdminBatch.ListRuns, mw.RateLimitAdmin)
	admin.GET("/batches/latest", c.AdminBatch.Latest, mw.RateLimitAdmin)

	admin.GET("/audit-logs", c.AdminAudit.List, mw.RateLimitAdmin)
	admin.GET("/audit-logs/:id", c.AdminAudit.FindByID, mw.RateLimitAdmin)
	admin.GET("/audit-logs/request/:request_id", c.AdminAudit.ListByRequestID, mw.RateLimitAdmin)

	admin.POST("/cleanup/expired", c.Cleanup.DeleteExpired, adminBatchWithCSRF...)
	admin.POST("/rate-limits/reset", c.AdminRateLimit.Reset, adminWithCSRF...)
}

func syncControllerContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if ctx, ok := middleware.GetAppContext(c); ok {
				if ctx.AuthUserID != nil {
					c.Set(controller.ContextUserIDKey, *ctx.AuthUserID)
				}
				if ctx.GuestSessionID != nil {
					c.Set(controller.ContextGuestSessionIDKey, *ctx.GuestSessionID)
				}
				if ctx.RequestID != "" {
					c.Set(controller.ContextRequestIDKey, ctx.RequestID)
				}
				if ctx.ClientIPHash != "" {
					c.Set(controller.ContextIPHashKey, ctx.ClientIPHash)
				}
			}
			return next(c)
		}
	}
}

func appendMiddlewares(base []echo.MiddlewareFunc, items ...echo.MiddlewareFunc) []echo.MiddlewareFunc {
	middlewares := make([]echo.MiddlewareFunc, 0, len(base)+len(items))
	middlewares = append(middlewares, base...)
	middlewares = append(middlewares, items...)
	return middlewares
}

func validateDeps(c Controllers, mw Middlewares) error {
	if c.Health == nil || c.CSRF == nil || c.GuestSession == nil || c.Auth == nil || c.Content == nil || c.Search == nil || c.Event == nil || c.Saved == nil || c.Rating == nil || c.Ranking == nil || c.Recommendation == nil || c.Modal == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "controller dependency is required")
	}
	if c.AdminBean == nil || c.AdminArticle == nil || c.AdminRelation == nil || c.AdminBatch == nil || c.AdminAudit == nil || c.Cleanup == nil || c.AdminRateLimit == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "admin controller dependency is required")
	}

	required := []echo.MiddlewareFunc{
		mw.Recover,
		mw.RequestID,
		mw.ClientIPHash,
		mw.SecureHeaders,
		mw.CORS,
		mw.BodyLimit,
		mw.AccessLog,
		mw.Auth,
		mw.OptionalAuth,
		mw.Admin,
		mw.GuestSession,
		mw.Actor,
		mw.CSRF,
		mw.RateLimitPublicRead,
		mw.RateLimitAuthIP,
		mw.RateLimitRefresh,
		mw.RateLimitUser,
		mw.RateLimitSearch,
		mw.RateLimitEvent,
		mw.RateLimitRecommendation,
		mw.RateLimitModal,
		mw.RateLimitAdmin,
		mw.RateLimitAdminBatch,
	}
	for _, item := range required {
		if item == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "middleware dependency is required")
		}
	}
	return nil
}
