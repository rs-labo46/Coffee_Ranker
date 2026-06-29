package main

import (
	"log"

	"coffee-ranker/db"
	"coffee-ranker/model"
)

func main() {
	database, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := db.CloseDB(database); err != nil {
			log.Fatal(err)
		}
	}()

	if err := database.AutoMigrate(
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
	); err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully migrated")
}
