package router

import (
	"app/db"
	"app/handler"
	"app/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func SetupRoutes(app *fiber.App, queries *db.Queries) {
	// Middleware que proporciona las queries a todos los handlers
	app.Use(func(c fiber.Ctx) error {
		c.Locals("queries", queries)
		return c.Next()
	})

	api := app.Group("/api", logger.New())

	api.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	protectedRoutes := api.Group("/", middleware.AuthRequired)

	products := protectedRoutes.Group("/products")

	products.Post("/create", middleware.IsRole(1), handler.CreateProduct)
	products.Post("/Category", middleware.IsRole(1), handler.CreateCategory)

	auth := api.Group("/auth")
	auth.Post("/login", handler.Login)
	auth.Post("/register", handler.Register)
	auth.Post("/logout", handler.Logout)
}
