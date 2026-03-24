package handler

import (
	"app/db"
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
)

type VariantInput struct {
	SKU   string `json:"sku"`
	Price string `json:"price"`
	Stock int32  `json:"stock"`
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

	// Validar que los SKU no existan ya
	for _, v := range input.Variants {
		_, err := queries.GetVariantBySKU(ctx, v.SKU)
		if err == nil {
			return c.Status(400).JSON(fiber.Map{"error": "El SKU '" + v.SKU + "' ya existe"})
		}
		// Si err != nil, asumimos que no existe (podría ser otro error, pero por simplicidad)
	}

	// 1. producto
	product, err := queries.CreateProduct(ctx, db.CreateProductParams{
		Name:        input.Name,
		Description: pgtype.Text{String: input.Description, Valid: true},
		CategoryID:  pgtype.Int4{Int32: input.CategoryID, Valid: true},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error creando producto: " + err.Error()})
	}

	var createdVariants []interface{}

	// 2. variantes + stock
	for _, v := range input.Variants {
		price := pgtype.Numeric{}
		if err := price.Scan(v.Price); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "precio inválido"})
		}

		variant, err := queries.CreateProductVariant(ctx, db.CreateProductVariantParams{
			ProductID:  pgtype.Int4{Int32: product.ID, Valid: true},
			Attributes: []byte(`{}`),
			Sku:        v.SKU,
			Price:      price,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Error creando variante: " + err.Error()})
		}

		_, err = queries.CreateInventory(ctx, db.CreateInventoryParams{
			VariantID: pgtype.Int4{Int32: variant.ID, Valid: true},

			Stock: v.Stock,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Error creando inventario: " + err.Error()})
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
