package main

import (
	"log"
	"net/http"

	"coffee-ranker/db"

	"github.com/labstack/echo/v4"
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

	e := echo.New()

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
