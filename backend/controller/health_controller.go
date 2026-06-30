package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthController struct {
	database *gorm.DB
	redis    *redis.Client
}

type HealthResponse struct {
	Status string `json:"status"`
}

func NewHealthController(database *gorm.DB, redisClient *redis.Client) *HealthController {
	return &HealthController{database: database, redis: redisClient}
}

func (h *HealthController) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *HealthController) Ready(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
	defer cancel()

	if h.database == nil || h.redis == nil {
		return c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "ng"})
	}

	sqlDB, err := h.database.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "ng"})
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "ng"})
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "ng"})
	}

	return c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}
