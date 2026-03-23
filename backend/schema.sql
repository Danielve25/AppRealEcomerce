-- Habilitar la extensión para UUIDs si no está activa
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE EXTENSION IF NOT EXISTS citext;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ==========================================
-- 1. USUARIOS Y ROLES (Sin cambios)
-- ==========================================
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL, -- Ej: 'ADMIN', 'DISPATCHER', 'CUSTOMER'
    permissions JSONB -- Ej: {"can_edit_products": true, "can_dispatch": true}
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    role_id INT REFERENCES roles (id),
    name VARCHAR(100) NOT NULL,
    email CITEXT UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_role_id ON users (role_id);

-- ++++++++++++++++++++++++++++++++++++++++++
-- [NUEVO] 1.1 DIRECCIONES DE USUARIO
-- ++++++++++++++++++++++++++++++++++++++++++
-- Un usuario puede tener múltiples direcciones (Casa, Oficina, etc.)
CREATE TABLE user_addresses (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users (id) ON DELETE CASCADE,
    title VARCHAR(100) NOT NULL, -- Ej: 'Mi Casa', 'Oficina'
    street_address VARCHAR(255) NOT NULL, -- Calle, Carrera, Número, Apto
    city VARCHAR(100) NOT NULL,
    state_province VARCHAR(100) NOT NULL, -- Departamento/Estado
    postal_code VARCHAR(20),
    country VARCHAR(100) NOT NULL,
    phone_contact VARCHAR(20), -- Teléfono específico para esta dirección
    is_default BOOLEAN DEFAULT FALSE, -- Dirección de envío predeterminada
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_addresses_user_id ON user_addresses (user_id);

CREATE UNIQUE INDEX idx_user_default_address ON user_addresses (user_id)
WHERE
    is_default = true;
-- ==========================================
-- 2. CATÁLOGO DE PRODUCTOS E INVENTARIO
-- ==========================================
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id INT REFERENCES categories (id) -- Permite subcategorías
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    category_id INT REFERENCES categories (id),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_category_id ON products (category_id);

-- ++++++++++++++++++++++++++++++++++++++++++
-- [NUEVO] 2.1 IMÁGENES DE PRODUCTO
-- ++++++++++++++++++++++++++++++++++++++++++
-- Un producto puede tener múltiples imágenes. Se guarda la URL (ej. de AWS S3 o Cloudinary)
CREATE TABLE product_images (
    id SERIAL PRIMARY KEY,
    product_id INT REFERENCES products (id) ON DELETE CASCADE,
    image_url VARCHAR(255) NOT NULL, -- La URL de la imagen en tu almacenamiento
    is_primary BOOLEAN DEFAULT FALSE, -- ¿Es la imagen principal que se muestra en el catálogo?
    display_order INT DEFAULT 0, -- Orden para mostrar las imágenes en la galería
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_product_images_product_id ON product_images (product_id);

CREATE TABLE product_variants (
    id SERIAL PRIMARY KEY,
    product_id INT REFERENCES products (id) ON DELETE CASCADE,
    sku VARCHAR(100) UNIQUE NOT NULL,
    attributes JSONB, -- Ej: {"color": "rojo", "talla": "M"}
    price DECIMAL(10, 2) NOT NULL,
    min_stock_alert INT NOT NULL DEFAULT 5
);

CREATE INDEX idx_product_variants_product_id ON product_variants (product_id);

-- ==========================================
-- 3. KARDEX / MOVIMIENTOS DE INVENTARIO (Sin cambios)
-- ==========================================
CREATE TABLE inventory_movements (
    id SERIAL PRIMARY KEY,
    variant_id INT REFERENCES product_variants (id),
    user_id UUID REFERENCES users (id), -- Empleado/Admin que registró el movimiento
    movement_type VARCHAR(20) CHECK (
        movement_type IN (
            'IN',
            'OUT',
            'ADJUSTMENT',
            'TRANSFER'
        )
    ),
    quantity INT NOT NULL,
    reference VARCHAR(255), -- Ej: ID del pedido, número de factura de proveedor
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_inventory_variant_id ON inventory_movements (variant_id);

CREATE INDEX idx_inventory_user_id ON inventory_movements (user_id);

CREATE INDEX idx_inventory_created_at ON inventory_movements (created_at);

-- ==========================================
-- 4. PEDIDOS Y VENTAS (Actualizado)
-- ==========================================
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    customer_id UUID REFERENCES users (id),
    -- [CAMBIO]: Ahora apuntamos a una dirección registrada por el usuario
    shipping_address_id INT REFERENCES user_addresses (id),
    status VARCHAR(30) CHECK (
        status IN (
            'PENDING',
            'PREPARING',
            'DISPATCHED',
            'DELIVERED',
            'CANCELLED'
        )
    ) DEFAULT 'PENDING',
    total_amount DECIMAL(10, 2) NOT NULL,
    -- [CAMBIO]: Aunque apuntamos al ID, guardamos una copia textual de la dirección
    -- para que, si el usuario la edita en el futuro, el pedido histórico no cambie.
    shipping_address_snapshot TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_customer_id ON orders (customer_id);

CREATE INDEX idx_orders_status ON orders (status);

CREATE INDEX idx_orders_created_at ON orders (created_at);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id UUID REFERENCES orders (id) ON DELETE CASCADE,
    variant_id INT REFERENCES product_variants (id),
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    subtotal DECIMAL(10, 2) NOT NULL
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);

CREATE INDEX idx_order_items_variant_id ON order_items (variant_id);

-- ==========================================
-- 5. PAGOS (Sin cambios)
-- ==========================================
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    order_id UUID REFERENCES orders (id),
    provider VARCHAR(50), -- Ej: 'Stripe', 'MercadoPago', 'Cash'
    transaction_ref VARCHAR(255) UNIQUE,
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) CHECK (
        status IN (
            'PENDING',
            'COMPLETED',
            'FAILED',
            'REFUNDED'
        )
    ) DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payments_order_id ON payments (order_id);

CREATE INDEX idx_payments_status ON payments (status);

-- ==========================================
-- 6. OPCIONAL: PROMOCIONES Y DEVOLUCIONES (Sin cambios)
-- ==========================================
CREATE TABLE promotions (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    discount_type VARCHAR(20) CHECK (
        discount_type IN ('PERCENTAGE', 'FIXED_AMOUNT')
    ),
    discount_value DECIMAL(10, 2) NOT NULL,
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE returns (
    id SERIAL PRIMARY KEY,
    order_id UUID REFERENCES orders (id),
    variant_id INT REFERENCES product_variants (id),
    quantity INT NOT NULL,
    reason TEXT,
    status VARCHAR(20) CHECK (
        status IN (
            'PENDING',
            'APPROVED',
            'REJECTED'
        )
    ),
    refund_amount DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_returns_order_id ON returns (order_id);

CREATE INDEX idx_returns_variant_id ON returns (variant_id);

CREATE TABLE inventory (
    id SERIAL PRIMARY KEY,
    variant_id INT REFERENCES product_variants (id) ON DELETE CASCADE,
    stock INT NOT NULL DEFAULT 0,
    reserved_stock INT NOT NULL DEFAULT 0, -- pedidos pendientes
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (variant_id)
);

CREATE INDEX idx_inventory_variant ON inventory (variant_id);