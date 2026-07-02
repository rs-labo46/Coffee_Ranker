package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"coffee-ranker/controller"
	"coffee-ranker/entity"
	"coffee-ranker/middleware"

	"github.com/labstack/echo/v4"
)

// ControllerやMiddlewareのDI漏れをRegister時点で検知できることを確認する。
// 起動後にnil panicを起こさず、納品前に配線漏れを落とすためのRouter単体テスト。
func TestRegisterRequiresDependencies(t *testing.T) {
	controllers := validRouterControllers()
	controllers.Auth = nil

	err := Register(echo.New(), controllers, validRouterMiddlewares())
	if err == nil {
		t.Fatal("expected dependency error")
	}

	middlewares := validRouterMiddlewares()
	middlewares.CSRF = nil
	err = Register(echo.New(), validRouterControllers(), middlewares)
	if err == nil {
		t.Fatal("expected middleware dependency error")
	}
}

// 主要APIのPathがRouterに登録されていることを確認する。
// Controller実装があってもRouter登録漏れなら外部から使えないため、案件レビューで落ちやすい箇所を固定する。
func TestRegisterRoutesCriticalEndpoints(t *testing.T) {
	e, err := NewRouter(validRouterControllers(), validRouterMiddlewares())
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	want := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodPost, "/auth/login"},
		{http.MethodPost, "/auth/refresh"},
		{http.MethodGet, "/search/beans"},
		{http.MethodPost, "/events"},
		{http.MethodPost, "/saved"},
		{http.MethodPost, "/admin/beans"},
		{http.MethodPost, "/admin/batches/ranking"},
	}

	for _, item := range want {
		if !routeExists(e, item.method, item.path) {
			t.Fatalf("route %s %s is not registered", item.method, item.path)
		}
	}
}

// MiddlewareのAppContextがController用Context keyへ同期されることを確認する。
// Controllerがbodyのuser_idを信用せず、認証Middleware由来のUserIDだけを使うための境界テスト。
func TestSyncControllerContextCopiesAppContext(t *testing.T) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthUser(c, 77, entity.UserRoleAdmin, 3)
			middleware.SetRequestID(c, "req-1")
			ctx := middleware.EnsureAppContext(c)
			ctx.ClientIPHash = "ip-hash-1"
			return next(c)
		}
	})
	e.Use(syncControllerContext())
	e.GET("/ctx", func(c echo.Context) error {
		if got := c.Get(controller.ContextUserIDKey); got != uint64(77) {
			t.Fatalf("user id context = %v", got)
		}
		if got := c.Get(controller.ContextRequestIDKey); got != "req-1" {
			t.Fatalf("request id context = %v", got)
		}
		if got := c.Get(controller.ContextIPHashKey); got != "ip-hash-1" {
			t.Fatalf("ip hash context = %v", got)
		}
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ctx", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d", res.Code)
	}
}

// Router登録だけを検証するための空Controller群を返す。
func validRouterControllers() Controllers {
	return Controllers{
		Health:         &controller.HealthController{},
		CSRF:           &controller.CSRFController{},
		GuestSession:   &controller.GuestSessionController{},
		Auth:           &controller.AuthController{},
		Content:        &controller.ContentController{},
		Search:         &controller.SearchController{},
		Event:          &controller.EventController{},
		Saved:          &controller.SavedItemController{},
		Rating:         &controller.RatingController{},
		Ranking:        &controller.RankingController{},
		Recommendation: &controller.RecommendationController{},
		Modal:          &controller.ModalController{},
		AdminBean:      &controller.AdminBeanController{},
		AdminArticle:   &controller.AdminArticleController{},
		AdminRelation:  &controller.AdminRelationController{},
		AdminBatch:     &controller.AdminBatchController{},
		AdminAudit:     &controller.AdminAuditController{},
		Cleanup:        &controller.CleanupController{},
		AdminRateLimit: &controller.AdminRateLimitController{},
	}
}

// Router登録だけを検証するためのno-op Middleware群を返す。
func validRouterMiddlewares() Middlewares {
	mw := noopRouterMiddleware()
	return Middlewares{
		Recover:                 mw,
		RequestID:               mw,
		ClientIPHash:            mw,
		SecureHeaders:           mw,
		CORS:                    mw,
		BodyLimit:               mw,
		AccessLog:               mw,
		Auth:                    mw,
		OptionalAuth:            mw,
		Admin:                   mw,
		GuestSession:            mw,
		Actor:                   mw,
		CSRF:                    mw,
		RateLimitPublicRead:     mw,
		RateLimitAuthIP:         mw,
		RateLimitRefresh:        mw,
		RateLimitUser:           mw,
		RateLimitSearch:         mw,
		RateLimitEvent:          mw,
		RateLimitRecommendation: mw,
		RateLimitModal:          mw,
		RateLimitAdmin:          mw,
		RateLimitAdminBatch:     mw,
	}
}

// Router登録テスト用に処理をそのまま次へ渡すMiddlewareを返す。
func noopRouterMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
}

// Echoに指定method/pathが登録されているかを確認する。
func routeExists(e *echo.Echo, method string, path string) bool {
	for _, route := range e.Routes() {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}
