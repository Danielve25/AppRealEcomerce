package main

import (
	"context"
	"log"
	"os"

	"app/db"
	"app/router"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno
	godotenv.Load()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	DATABASE_URL := os.Getenv("DATABASE_URL")

	// Conectar a la base de datos
	conn, err := pgx.Connect(context.Background(), DATABASE_URL)
	if err != nil {
		log.Fatalf("Error conectando a BD: %v", err)
	}
	defer conn.Close(context.Background())

	// Crear instancia de queries
	queries := db.New(conn)

	app := fiber.New()

	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} - duracion: ${duration}\n",
		TimeFormat: "02-Jan-2006 15:04:05",
		TimeZone:   "UTC-5",
	}))

	router.SetupRoutes(app, queries)

	log.Fatal(app.Listen(":8080"))
}
