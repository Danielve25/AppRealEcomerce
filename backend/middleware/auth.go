package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(c fiber.Ctx) error {
	var tokenString string

	// 1. Intentar obtener el token de la Cookie (Prioridad Web)
	tokenString = c.Cookies("jwt")

	// 2. Si no hay cookie, intentar obtener del Header (Prioridad App)
	if tokenString == "" {
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// 3. Validar si encontramos algo
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No hay una sesión activa o token faltante",
		})
	}

	// 4. Parsear y Validar el JWT
	jwtSecret := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validar que el método de firma sea HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(jwtSecret), nil
	})

	// 5. Manejar errores de validación
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token inválido o expirado",
		})
	}

	// 6. Extraer claims y guardarlos en Locals para que los Handlers los usen
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		// user_id es un string (UUID)
		userID := claims["user_id"].(string)
		c.Locals("user_id", userID)

		// role_id es un float64 (convertir a int32)
		roleID := int32(claims["role_id"].(float64))
		c.Locals("role_id", roleID)
	}

	return c.Next()
}
