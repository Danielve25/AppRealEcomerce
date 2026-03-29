-- =======================================
-- USUARIOS
-- =======================================

-- ✅
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- ✅
-- name: CreateUser :one
insert into
    users (
        role_id,
        name,
        email,
        password_hash,
        is_active
    )
values ($1, $2, $3, $4, $5)
returning
    *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET
    name = $2,
    role_id = $3,
    is_active = $4
WHERE
    id = $1
RETURNING
    *;

-- =======================================
-- CATEGORÍAS
-- =======================================
-- ✅
-- name: CreateCategory :one
insert into categories (name) values ($1) returning *;

-- ✅
-- name: CreateSubCategory :one
insert into categories (name, parent_id) values ($1, $2) returning *;

-- ✅
-- name: GetCategories :many
select * from categories;

-- ✅
-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1;

-- ✅
-- name: GetSubCategories :many
SELECT * FROM categories WHERE parent_id = $1;

-- =======================================
-- PRODUCTOS
-- =======================================

-- ✅
-- name: CreateProduct :one
INSERT INTO
    products (
        category_id,
        name,
        description
    )
VALUES ($1, $2, $3)
RETURNING
    *;

-- ✅
-- name: GetProductFullByID :one
SELECT p.id, p.name, p.description, p.created_at,

-- Imagen principal
COALESCE(
    (
        SELECT image_url
        FROM product_images pi
        WHERE
            pi.product_id = p.id
            AND pi.is_primary = true
        LIMIT 1
    ),
    ''
) AS main_image,

-- Todas las imágenes
COALESCE(
    (
        SELECT json_agg(image_url)
        FROM product_images pi
        WHERE
            pi.product_id = p.id
    ),
    '[]'
) AS images,

-- Variantes
COALESCE(
    (
        SELECT json_agg(
                json_build_object(
                    'id', pv.id, 'sku', pv.sku, 'attributes', pv.attributes, 'price', pv.price
                )
            )
        FROM product_variants pv
        WHERE
            pv.product_id = p.id
    ),
    '[]'
) AS variants
FROM products p
WHERE
    p.id = $1
    AND p.is_active = true;

-- ✅
-- name: GetAllProducts :many
SELECT p.id, p.name, p.description, COALESCE(
        (
            SELECT image_url
            FROM product_images pi
            WHERE
                pi.product_id = p.id
                AND pi.is_primary = true
            LIMIT 1
        ), ''
    ) AS image, (
        SELECT MIN(price)
        FROM product_variants pv
        WHERE
            pv.product_id = p.id
    ) AS price_from
FROM products p
WHERE
    p.is_active = true
ORDER BY p.created_at DESC
LIMIT $1
OFFSET
    $2;

-- name: CountAllProducts :one
SELECT COUNT(*) FROM products WHERE is_active = true;

-- ✅
-- name: GetProductsByCategory :many
SELECT p.id, p.name, p.description,
    -- If no image is found, return an empty string instead of NULL
    COALESCE(
        (
            SELECT image_url
            FROM product_images pi
            WHERE
                pi.product_id = p.id
                AND pi.is_primary = true
            LIMIT 1
        ), ''
    ) AS image,
    -- If no price is found, return 0 (or a default numeric value)
    COALESCE(
        (
            SELECT MIN(price)
            FROM product_variants pv
            WHERE
                pv.product_id = p.id
        ), 0
    ) AS price_from
FROM products p
WHERE
    p.category_id = $1
    AND p.is_active = true
ORDER BY p.created_at DESC
LIMIT $2
OFFSET
    $3;

-- name: CountProductsByCategory :one
SELECT COUNT(*)
FROM products
WHERE
    category_id = $1
    AND is_active = true;

-- name: UpdateProduct :one
UPDATE products
SET
    category_id = COALESCE(
        sqlc.narg ('category_id'),
        category_id
    ),
    name = COALESCE(sqlc.narg ('name'), name),
    description = COALESCE(
        sqlc.narg ('description'),
        description
    )
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteProduct :exec
UPDATE products SET is_active = false WHERE id = $1;

-- =======================================
-- IMÁGENES DE PRODUCTOS
-- =======================================

-- name: CreateProductImage :one
INSERT INTO
    product_images (
        product_id,
        image_url,
        is_primary,
        display_order
    )
VALUES ($1, $2, $3, $4)
RETURNING
    *;

-- =======================================
-- VARIANTES DE PRODUCTOS
-- =======================================

-- ✅
-- name: CreateProductVariant :one
INSERT INTO
    product_variants (
        product_id,
        sku,
        attributes,
        price
    )
VALUES ($1, $2, $3, $4)
RETURNING
    *;

-- name: GetVariantByID :one
SELECT * FROM product_variants WHERE id = $1;

-- name: GetVariantBySKU :one
SELECT * FROM product_variants WHERE sku = $1;

-- name: UpdateVariant :one
UPDATE product_variants
SET
    price = $2,
    attributes = $3
WHERE
    id = $1
RETURNING
    *;

-- =======================================
-- INVENTARIO
-- =======================================

-- ✅
-- name: CreateInventory :one
INSERT INTO
    inventory (variant_id, stock)
VALUES ($1, $2)
ON CONFLICT (variant_id) DO
UPDATE
SET
    stock = inventory.stock + EXCLUDED.stock
RETURNING
    *;

-- name: GetInventoryByVariant :one
SELECT * FROM inventory WHERE variant_id = $1;

-- name: UpdateInventoryStock :one
UPDATE inventory
SET
    stock = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE
    variant_id = $1
RETURNING
    *;

-- name: ReserveStock :one
UPDATE inventory
SET
    reserved_stock = reserved_stock + $2
WHERE
    variant_id = $1
    AND stock - reserved_stock >= $2
RETURNING
    *;

-- name: ReleaseReservedStock :one
UPDATE inventory
SET
    reserved_stock = reserved_stock - $2
WHERE
    variant_id = $1
RETURNING
    *;

-- name: CheckStock :one
SELECT stock, reserved_stock FROM inventory WHERE variant_id = $1;

-- name: ConfirmStock :one
UPDATE inventory
SET
    stock = stock - $2,
    reserved_stock = reserved_stock - $2
WHERE
    variant_id = $1
RETURNING
    *;

-- =======================================
-- MOVIMIENTOS DE INVENTARIO
-- =======================================

-- name: CreateMovement :one
insert into
    inventory_movements (
        user_id,
        variant_id,
        movement_type,
        quantity,
        reference,
        reason
    )
values ($1, $2, $3, $4, $5, $6)
returning
    *;

-- name: GetMovementsByVariant :many
SELECT *
FROM inventory_movements
WHERE
    variant_id = $1
ORDER BY created_at DESC;

-- name: GetMovementsByUser :many
SELECT *
FROM inventory_movements
WHERE
    user_id = $1
ORDER BY created_at DESC;

-- =======================================
-- DIRECCIONES DE USUARIO
-- =======================================

-- name: GetUserAddresses :many
SELECT *
FROM user_addresses
WHERE
    user_id = $1
    AND is_active = true
ORDER BY is_default DESC, created_at DESC;

-- name: CreateUserAddress :one
INSERT INTO
    user_addresses (
        user_id,
        title,
        street_address,
        city,
        state_province,
        postal_code,
        country,
        phone_contact,
        is_default
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9
    )
RETURNING
    *;

-- name: ClearDefaultAddress :exec
UPDATE user_addresses
SET
    is_default = false
WHERE
    user_id = $1
    AND is_active = true;

-- name: SetDefaultAddress :one
UPDATE user_addresses
SET
    is_default = true
WHERE
    id = $1
    AND user_id = $2
    AND is_active = true
RETURNING
    *;

-- name: UpdateUserAddress :one
UPDATE user_addresses
SET
    title = $1,
    street_address = $2,
    city = $3,
    state_province = $4,
    postal_code = $5,
    country = $6,
    phone_contact = $7
WHERE
    id = $8
    AND user_id = $9
    AND is_active = true
RETURNING
    *;

-- name: DeleteUserAddress :exec
UPDATE user_addresses
SET
    is_active = false
WHERE
    id = $1
    AND user_id = $2;

-- =======================================
-- ÓRDENES
-- =======================================

-- name: CreateOrder :one
INSERT INTO
    orders (
        customer_id,
        shipping_address_id,
        total_amount,
        shipping_address_snapshot
    )
VALUES ($1, $2, $3, $4)
RETURNING
    *;

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1;

-- name: GetOrdersByUser :many
SELECT * FROM orders WHERE customer_id = $1 ORDER BY created_at DESC;

-- name: UpdateOrderStatus :one
UPDATE orders
SET
    status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $1
RETURNING
    *;

-- =======================================
-- ÍTEMS DE ÓRDENES
-- =======================================

-- name: CreateOrderItem :one
INSERT INTO
    order_items (
        order_id,
        variant_id,
        quantity,
        unit_price,
        subtotal
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING
    *;

-- =======================================
-- PAGOS
-- =======================================

-- name: CreatePayment :one
INSERT INTO
    payments (
        order_id,
        provider,
        transaction_ref,
        amount,
        status
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING
    *;

-- name: GetPaymentsByOrder :many
SELECT * FROM payments WHERE order_id = $1;

-- =======================================
-- PROMOCIONES
-- =======================================

-- name: GetPromotionByCode :one
SELECT *
FROM promotions
WHERE
    code = $1
    AND is_active = true
    AND (
        valid_from IS NULL
        OR valid_from <= NOW()
    )
    AND (
        valid_until IS NULL
        OR valid_until >= NOW()
    );

-- =======================================
-- DEVOLUCIONES
-- =======================================

-- name: CreateReturn :one
INSERT INTO
    returns (
        order_id,
        variant_id,
        quantity,
        reason,
        status
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING
    *;