package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/piyawat001/user-auth-api/handlers"
	"github.com/piyawat001/user-auth-api/middleware"
)

var client *mongo.Client

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Initialize Fiber app
	app := fiber.New()

	// เพิ่ม CORS Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:8080", // แก้ไขเป็นที่อยู่ของ Frontend
		AllowMethods:     "GET,POST,PUT,DELETE",
		AllowHeaders:     "Content-Type,Authorization",
		AllowCredentials: true, // หากคุณใช้คุกกี้หรือ Authorization headers
	}))

	// Set up handlers with MongoDB client
	h := handlers.NewHandler(client)

	// Public routes
	app.Post("/register", h.Register)
	app.Post("/login", h.Login)
	app.Get("/packages", h.GetPackages)

	// Apply authentication middleware
	app.Use(middleware.Auth)

	// Protected routes
	app.Get("/users", h.GetAllUsers)
	app.Post("/admin/approve", h.ApproveUser)
	app.Post("/admin/set-package", h.AdminSetPackage)
	// Start server
	log.Fatal(app.Listen(":3000"))
}
