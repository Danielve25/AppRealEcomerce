package handler

import (
	"app/db"
	"context"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateCategoryInput struct {
	Name string `json:"name"`
}

type CreateSubCategoryInput struct {
	Name     string `json:"name"`
	ParentID int32  `json:"parent_id"`
}

func CreateCategory(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)

	var input CreateCategoryInput

	if err := c.Bind().Body(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	name := input.Name
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "El nombre de la categoría es requerido"})
	}

	category, err := queries.CreateCategory(context.Background(), name)
	if err != nil {
		return c.Status(500).JSON(err)
	}

	return c.JSON(category)
}

func CreateSubCategory(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)

	var input CreateSubCategoryInput

	if err := c.Bind().Body(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}
	if input.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "El nombre de la subcategoría es requerido"})
	}
	if input.ParentID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "El ID de la categoría padre es requerido"})
	}

	subCategory, err := queries.CreateSubCategory(context.Background(), db.CreateSubCategoryParams{
		Name:     input.Name,
		ParentID: pgtype.Int4{Int32: input.ParentID, Valid: true},
	})
	if err != nil {
		return c.Status(500).JSON(err)
	}

	return c.JSON(subCategory)
}

func GetCategories(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)

	categories, err := queries.GetCategories(context.Background())
	if err != nil {
		return c.Status(500).JSON(err)
	}

	return c.JSON(categories)
}

func GetCategoryByID(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)

	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
	}

	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
	}

	category, err := queries.GetCategoryByID(context.Background(), int32(idInt))
	if err != nil {
		return c.Status(500).JSON(err)
	}

	return c.JSON(category)
}

func GetSubCategories(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)

	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
	}

	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
	}

	subCategories, err := queries.GetSubCategories(context.Background(), pgtype.Int4{Int32: int32(idInt), Valid: true})
	if err != nil {
		return c.Status(500).JSON(err)
	}

	return c.JSON(subCategories)
}
