// handler/auth.go
package handler

import (
	"app/db"
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret string

func init() {
	godotenv.Load()
	jwtSecret = os.Getenv("JWT_SECRET")
	log.Printf("jwt secret: %s", jwtSecret)
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET no está configurado en .env")
	}
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(c fiber.Ctx) error {
	var input LoginInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	email := input.Email
	password := input.Password

	if email == "" || password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email y contraseña requeridos"})
	}

	// Obtener queries desde el contexto
	queries := c.Locals("queries").(*db.Queries)

	user, err := queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Usuario no encontrado"})
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Contraseña incorrecta"})
	}

	// Extraer el ID de UUID y el valor int32 de RoleID
	userIDStr := user.ID.String()
	roleIDVal := int32(0)
	if user.RoleID.Valid {
		roleIDVal = user.RoleID.Int32
	}

	claims := jwt.MapClaims{
		"user_id": userIDStr,
		"role_id": roleIDVal,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		log.Printf("error firmando JWT: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No se pudo generar el token"})
	}
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    signedToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	return c.JSON(fiber.Map{
		"token": signedToken,
		"user": fiber.Map{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.RoleID,
		},
	})

}

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Register(c fiber.Ctx) error {
	var input RegisterInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "JSON inválido"})
	}

	name := input.Name
	email := input.Email
	password := input.Password
	if name == "" || email == "" || password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nombre, email y contraseña requeridos"})
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No se pudo procesar la contraseña"})
	}

	// Obtener queries desde el contexto
	queries := c.Locals("queries").(*db.Queries)

	if _, err := queries.GetUserByEmail(context.Background(), email); err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "El email ya está registrado"})
	}

	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		RoleID:       pgtype.Int4{Int32: 2, Valid: true},
		IsActive:     pgtype.Bool{Bool: true, Valid: true}, // Asignar rol de usuario por defecto
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"id":        user.ID,
			"name":      user.Name,
			"email":     user.Email,
			"role":      user.RoleID,
			"is_active": user.IsActive,
		},
	})
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func Logout(c fiber.Ctx) error {
	// 1. "Matar" la cookie para el navegador
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",                         // Vaciar el contenido
		Expires:  time.Now().Add(-time.Hour), // Expira hace una hora
		HTTPOnly: true,
		Secure:   false, // Cambiar a true en producción (HTTPS)
		SameSite: "Lax",
		Path:     "/",
	})

	// 2. Responder al cliente
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sesión cerrada correctamente",
	})
}
