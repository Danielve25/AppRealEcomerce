package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func main() {
	app := fiber.New()

	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path}\n",
		TimeFormat: "02-Jan-2006 15:04:05",
		TimeZone:   "UTC-5",
	}))

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))
}
