package handler

import (
	"app/db"
	"context"
	"encoding/json"
	"strconv"

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
type VariantResponse struct {
	ID         int32                  `json:"id"`
	SKU        string                 `json:"sku"`
	Attributes map[string]interface{} `json:"attributes"`
	Price      float64                `json:"price"`
}
type ProductResponse struct {
	ID          int32             `json:"id"`
	Name        string            `json:"name"`
	Description pgtype.Text       `json:"description"`
	CreatedAt   pgtype.Timestamp  `json:"created_at"`
	MainImage   pgtype.Text       `json:"main_image"`
	Images      []string          `json:"images"`
	Variants    []VariantResponse `json:"variants"`
}

type UpdateProductInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	CategoryID  *int32  `json:"category_id"`
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

func GetProductByID(c fiber.Ctx) error {
	queries := c.Locals("queries").(*db.Queries)

	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
	}

	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID inválido"})
	}

	product, err := queries.GetProductFullByID(context.Background(), int32(idInt))

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error obteniendo producto: " + err.Error()})
	}

	// Parsear images
	var images []string
	if len(product.Images) > 0 {
		err = json.Unmarshal(product.Images, &images)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Error parseando imágenes"})
		}
	}

	// Parsear variants
	var variants []VariantResponse
	if len(product.Variants) > 0 {
		err = json.Unmarshal(product.Variants, &variants)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Error parseando variantes"})
		}
	}

	response := ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		CreatedAt:   product.CreatedAt,
		MainImage:   product.MainImage,
		Images:      images,
		Variants:    variants,
	}

	return c.JSON(response)
}

func GetAllProducts(c fiber.Ctx) error {

	var limitstr string = c.Query("limit", "10")
	var offsetstr string = c.Query("offset", "0")

	limit, err := strconv.ParseInt(limitstr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Parámetro 'limit' inválido"})
	}
	offset, err := strconv.ParseInt(offsetstr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Parámetro 'offset' inválido"})
	}

	queries := c.Locals("queries").(*db.Queries)

	products, err := queries.GetAllProducts(context.Background(), db.GetAllProductsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error obteniendo productos: " + err.Error()})
	}

	return c.JSON(products)
}

func GetProductsByCategory(c fiber.Ctx) error {
	categoryIDStr := c.Params("id")
	limitStr := c.Query("limit", "10")
	offsetStr := c.Query("offset", "0")

	ID, err := strconv.ParseInt(categoryIDStr, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID de categoría inválido"})
	}

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Parámetro 'limit' inválido"})
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Parámetro 'offset' inválido"})
	}

	queries := c.Locals("queries").(*db.Queries)

	products, err := queries.GetProductsByCategory(context.Background(), db.GetProductsByCategoryParams{
		CategoryID: pgtype.Int4{Int32: int32(ID), Valid: true},
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error obteniendo productos por categoría: " + err.Error()})
	}

	return c.JSON(products)
}
