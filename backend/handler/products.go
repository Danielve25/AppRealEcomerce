package handler

import (
	"app/db"
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
)

type VariantInput struct {
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
	Stock int32   `json:"stock"`
}

type CreateProductInput struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	CategoryID  int32          `json:"category_id"`
	Variants    []VariantInput `json:"variants"`
}

func CreateProduct(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)
	var input CreateProductInput

	if err := c.Bind().Body(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	ctx := context.Background()

	// 1. producto
	product, err := queries.CreateProduct(ctx, db.CreateProductParams{
		Name:        input.Name,
		Description: pgtype.Text{String: input.Description, Valid: true},
		CategoryID:  pgtype.Int4{Int32: input.CategoryID, Valid: true},
	})
	if err != nil {
		return c.Status(500).JSON(err)
	}

	var createdVariants []interface{}

	// 2. variantes + stock
	for _, v := range input.Variants {
		price := pgtype.Numeric{}
		if err := price.Scan(v.Price); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "precio inválido"})
		}

		variant, err := queries.CreateProductVariant(ctx, db.CreateProductVariantParams{
			ProductID: pgtype.Int4{Int32: product.ID, Valid: true},
			Sku:       v.SKU,
			Price:     price,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "variante"})
		}

		_, err = queries.CreateInventory(ctx, db.CreateInventoryParams{
			VariantID: variant.ProductID,
			Stock:     v.Stock,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "stock"})
		}

		createdVariants = append(createdVariants, variant)
	}

	return c.JSON(fiber.Map{
		"product":  product,
		"variants": createdVariants,
	})
}

type CreateCategoryInput struct {
	Name string `json:"name"`
}

func CreateCategory(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)

	var input CreateCategoryInput

	name := input.Name
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "El nombre de la categoría es requerido"})
	}
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	category, err := queries.CreateCategory(context.Background(), name)
	if err != nil {
		return c.Status(500).JSON(err)
	}

	return c.JSON(category)
}
