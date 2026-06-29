package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	appdb "coffee-ranker/db"
	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
	"coffee-ranker/usecase"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newTxManagerIntegrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	if os.Getenv("SKIP_REPOSITORY_INTEGRATION_TESTS") == "1" {
		t.Skip("SKIP_REPOSITORY_INTEGRATION_TESTS=1")
	}

	_ = godotenv.Load("../.env", ".env")

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Tokyo",
			txManagerTestEnv("POSTGRES_HOST", "127.0.0.1"),
			txManagerTestEnv("POSTGRES_USER", "coffee"),
			txManagerTestEnv("POSTGRES_PASSWORD", "coffeepassword"),
			txManagerTestEnv("POSTGRES_DB", "mydb"),
			txManagerTestEnv("POSTGRES_PORT", "5435"),
		)
	}

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	schema := fmt.Sprintf("tx_test_%d", time.Now().UnixNano())

	if err := database.Exec("CREATE SCHEMA " + quoteTxIdent(schema)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	if err := database.Exec("SET search_path TO " + quoteTxIdent(schema)).Error; err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	t.Cleanup(func() {
		_ = database.Exec("DROP SCHEMA IF EXISTS " + quoteTxIdent(schema) + " CASCADE").Error
		_ = sqlDB.Close()
	})

	if err := database.AutoMigrate(
		&model.User{},
		&model.GuestSession{},
		&model.Bean{},
		&model.Article{},
		&model.RankTarget{},
		&model.RefreshToken{},
		&model.BeanArticle{},
		&model.ModalDisplayLog{},
		&model.ModalBlockLog{},
		&model.ActionEvent{},
		&model.SavedItem{},
		&model.Rating{},
		&model.ContentMetric{},
		&model.InterestProfile{},
		&model.BatchRun{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("auto migrate test database: %v", err)
	}

	return database
}

func txManagerTestEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	if key == "POSTGRES_HOST" && value == "db" {
		return "127.0.0.1"
	}

	if key == "POSTGRES_PORT" && value == "5432" {
		return "5435"
	}

	return value
}

func quoteTxIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// TxManagerでエラーが返されたとき、DB更新がrollbackされることを確認する。
// rollback時に返るエラーが、TxManager側のエラーではなく元の業務エラーのまま返ることも確認する。
func TestTxManager_RollbackKeepsOriginalBusinessError(t *testing.T) {
	database := newTxManagerIntegrationTestDB(t)
	ctx := context.Background()

	txManager := appdb.NewTxManager(database)

	err := txManager.WithinTx(ctx, func(ctx context.Context, tx usecase.ITxRepos) error {
		user := model.User{
			Name:         "rollback-user",
			Email:        "rollback@example.com",
			PasswordHash: "hashed-password",
			Role:         entity.UserRoleUser,
			Status:       entity.UserStatusActive,
		}

		if err := tx.User().Create(ctx, &user); err != nil {
			return err
		}

		return entity.ErrRefreshTokenExpired
	})

	if !errors.Is(err, entity.ErrRefreshTokenExpired) {
		t.Fatalf("tx error = %v, want ErrRefreshTokenExpired", err)
	}

	exists, findErr := repository.NewUserRepository(database).ExistsByEmail(ctx, "rollback@example.com")
	if findErr != nil {
		t.Fatalf("exists after rollback: %v", findErr)
	}

	if exists {
		t.Fatal("user created inside rolled back transaction still exists")
	}
}
