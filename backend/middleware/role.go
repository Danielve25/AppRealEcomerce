// middleware/role.go
package middleware

import (
	"github.com/gofiber/fiber/v3"
)

// IsRole recibe una lista de IDs de roles permitidos (ej: 1 para Admin, 2 para User)
func IsRole(allowedRoles ...int32) fiber.Handler {
	return func(c fiber.Ctx) error {
		// 1. Obtener el role_id que sembró el middleware de Auth
		roleID, ok := c.Locals("role_id").(int32)

		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No se pudo determinar el rol del usuario",
			})
		}

		// 2. Verificar si el rol del usuario está en la lista de permitidos
		isAllowed := false
		for _, role := range allowedRoles {
			if roleID == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No tienes permisos para acceder a este recurso",
			})
		}

		// 3. Si todo está bien, continuar al siguiente handler
		return c.Next()
	}
}
