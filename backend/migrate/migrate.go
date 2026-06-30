package migrate

import (
	"coffee-ranker/model"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.GuestSession{},
		&model.Bean{},
		&model.Article{},
		&model.BeanArticle{},
		&model.RankTarget{},
		&model.ActionEvent{},
		&model.ModalDisplayLog{},
		&model.ModalBlockLog{},
		&model.SavedItem{},
		&model.Rating{},
		&model.ContentMetric{},
		&model.InterestProfile{},
		&model.BatchRun{},
		&model.AuditLog{},
	)
}
