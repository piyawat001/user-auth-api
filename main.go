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
		AllowOrigins:     "*", // อนุญาตทุกแหล่งที่มา
		AllowMethods:     "GET,POST,PUT,DELETE",
		AllowHeaders:     "Content-Type,Authorization",
		AllowCredentials: false, // ไม่รองรับ cookies หรือ headers
	}))

	// Set up handlers with MongoDB client
	h := handlers.NewHandler(client)

	//create users
	app.Post("/register", h.Register) 
	app.Post("/login", h.Login) 

	//user
	app.Get("/users", h.GetAllUsers)       // ดึงข้อมูลผู้ใช้ทั้งหมด
	app.Delete("/users/:id", h.DeleteUser) // ลบผู้ใช้

	//Admin Routes
	app.Post("/admin/approve", h.ApproveUser)           // อนุมัติผู้ใช้
	app.Post("/admin/set-package", h.AdminSetPackage)   // ตั้งค่าชุดแพ็กเกจ
	app.Get("/pendingQuestions", h.GetPendingQuestions) // ดึงคำถามที่ยังไม่ได้ตอบ
	app.Post("/admin/approve", h.ApproveUser)


	//Patient Routes
	app.Post("/patients", h.CreatePatient)       // สร้างข้อมูลผู้ป่วยใหม่
	app.Put("/patients/:id", h.UpdatePatient)    // แก้ไขข้อมูลผู้ป่วย
	app.Delete("/patients/:id", h.DeletePatient) // ลบข้อมูลผู้ป่วย
	app.Get("/allpatients", h.GetAllPatients)    // ดึงข้อมูลผู้ป่วยทั้งหมด

	//Question Routes
	app.Post("/questions", h.CreateQuestion)                                        // สร้างคำถามใหม่
	app.Get("/questions/user/:userId", h.GetMyQuestions)                            // ดึงประวัติคำถามของผู้ใช้
	app.Get("/questions/:id", h.GetQuestionDetail)                                  // ดึงรายละเอียดคำถามเฉพาะข้อ
	app.Put("/questions/:id", h.UpdateQuestion)                                     // อัปเดตคำถาม (หรือการตอบคำถาม)
	app.Put("/questions/notification-bell/:userId", h.UpdateNotificationBellStatus) // อัปเดตสถานะแจ้งเตือน
	app.Delete("/questions/:id", h.DeleteQuestion)                                  // ลบคำถาม

	app.Get("/notifications/:id", h.GetNotificationCount)        // นับจำนวนการแจ้งเตือน
	app.Put("/notifications/:id/read", h.MarkNotificationAsRead) // ทำเครื่องหมายว่าแจ้งเตือนถูกอ่านแล้ว

	// Start server
	log.Fatal(app.Listen(":3000"))
}
