package handlers

import (
	"context"
	"os"
	"time"
	"log"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/piyawat001/user-auth-api/models"
)

type Handler struct {
	client *mongo.Client
}

func NewHandler(client *mongo.Client) *Handler {
	return &Handler{client: client}
}
func (h *Handler) GetAllUsers(c *fiber.Ctx) error {
    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("users")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := collection.Find(ctx, bson.M{})
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch users"})
    }
    defer cursor.Close(ctx)

    var users []bson.M
    if err = cursor.All(ctx, &users); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot decode users"})
    }

    // Filter out sensitive information
    for i := range users {
        delete(users[i], "password")
    }

    return c.JSON(users)
}
func (h *Handler) Register(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot hash password"})
	}

	user.Password = string(hashedPassword)
	user.Role = "user"
	user.Status = "pending"
	user.Package = "free"
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// ตรวจสอบให้แน่ใจว่าได้ส่งค่าชื่อโรงพยาบาล
	if user.Hospital == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Hospital is required"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot insert user"})
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	user.Password = "" // Don't send password back

	return c.Status(fiber.StatusCreated).JSON(user)
}



func (h *Handler) Login(c *fiber.Ctx) error {
	var loginUser struct {
		Identifier string `json:"identifier"`  // รับค่าเป็นทั้ง email หรือ username
		Password   string `json:"password"`
	}

	// ตรวจสอบการส่งข้อมูลใน request
	if err := c.BodyParser(&loginUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// เชื่อมต่อกับ collection "users"
	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// สร้าง query เพื่อค้นหาผู้ใช้ด้วย email หรือ username
	var user models.User
	filter := bson.M{
		"$or": []bson.M{
			{"email": loginUser.Identifier},
			{"username": loginUser.Identifier},
		},
	}

	// ดึงข้อมูลผู้ใช้ที่ตรงกับ email หรือ username
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email/username or password"})
	}

	// ตรวจสอบความถูกต้องของ password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginUser.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email/username or password"})
	}

	// สร้าง JWT token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.ID
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot generate token"})
	}

	// ส่ง response กลับไปพร้อมกับข้อมูลผู้ใช้
	return c.JSON(fiber.Map{
		"token":     t,
		"id":        user.ID.Hex(),       // ส่ง ID ของผู้ใช้
		"username":  user.Username,       // ส่ง username
		"email":     user.Email,          // ส่ง email
		"password":  user.Password,       // ส่ง password (ถ้าจำเป็น แต่ควรปกป้องข้อมูล)
		"role":      user.Role,           // ส่ง role
		"status":    user.Status,         // ส่งสถานะ
		"package":   user.Package,        // ส่ง package
		"hospital":  user.Hospital,       // ส่งชื่อโรงพยาบาล
		"createdAt": user.CreatedAt,      // ส่งวันที่สร้าง
		"updatedAt": user.UpdatedAt,      // ส่งวันที่อัพเดตล่าสุด
	})
}





func (h *Handler) ApproveUser(c *fiber.Ctx) error {
	var approveRequest struct {
		UserID string `json:"user_id"`
	}

	if err := c.BodyParser(&approveRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	objectID, err := primitive.ObjectIDFromHex(approveRequest.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":    "approved",
			"updatedAt": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot update user"})
	}

	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(fiber.Map{"message": "User approved successfully"})
}

func (h *Handler) GetPackages(c *fiber.Ctx) error {
	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("packages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch packages"})
	}
	defer cursor.Close(ctx)

	var packages []models.Package
	if err = cursor.All(ctx, &packages); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot decode packages"})
	}

	return c.JSON(packages)
}

func (h *Handler) DeleteUser(c *fiber.Ctx) error {
    userID := c.Params("id")

    objectID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
    }

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("users")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot delete user"})
    }

    if result.DeletedCount == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
    }

    return c.JSON(fiber.Map{"message": "User deleted successfully"})
}

func (h *Handler) AdminSetPackage(c *fiber.Ctx) error {
    var setPackageRequest struct {
        UserID      string `json:"user_id"`
        Package     string `json:"package"`
        Role        string `json:"role"`
        ExpiryDays  int    `json:"expiry_days"`  // Days until package expires
    }

    if err := c.BodyParser(&setPackageRequest); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
    }

    objectID, err := primitive.ObjectIDFromHex(setPackageRequest.UserID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
    }

    var expiryDate *time.Time
    if setPackageRequest.Package == "plus" {
        // Set expiry based on the number of days specified by the admin
        expiry := time.Now().Add(time.Hour * 24 * time.Duration(setPackageRequest.ExpiryDays))
        expiryDate = &expiry
    } else if setPackageRequest.Package == "premium" {
        // No expiry for "premium"
        expiryDate = nil
    }

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("users")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    update := bson.M{
        "$set": bson.M{
            "package":   setPackageRequest.Package,
            "role":      setPackageRequest.Role,
            "expiry":    expiryDate,
            "updatedAt": time.Now(),
        },
    }

    result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot update user"})
    }

    if result.ModifiedCount == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
    }

    return c.JSON(fiber.Map{"message": "User package and role updated successfully"})
}


func (h *Handler) CreatePatient(c *fiber.Ctx) error {
	var patient models.Patient
	if err := c.BodyParser(&patient); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	patient.CreatedAt = time.Now()
	patient.UpdatedAt = time.Now()

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("patients")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, patient)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot insert patient"})
	}

	patient.ID = result.InsertedID.(primitive.ObjectID)

	return c.Status(fiber.StatusCreated).JSON(patient)
}

func (h *Handler) UpdatePatient(c *fiber.Ctx) error {
	patientID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(patientID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid patient ID"})
	}

	var patient models.Patient
	if err := c.BodyParser(&patient); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	patient.UpdatedAt = time.Now()

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("patients")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": patient,
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot update patient"})
	}

	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Patient not found"})
	}

	return c.JSON(fiber.Map{"message": "Patient updated successfully"})
}
func (h *Handler) DeletePatient(c *fiber.Ctx) error {
	patientID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(patientID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid patient ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("patients")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot delete patient"})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Patient not found"})
	}

	return c.JSON(fiber.Map{"message": "Patient deleted successfully"})
}

func (h *Handler) GetAllPatients(c *fiber.Ctx) error {
    // เชื่อมต่อกับ collection "patients"
    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("patients")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // ค้นหาข้อมูลผู้ป่วยทั้งหมด
    cursor, err := collection.Find(ctx, bson.M{})
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch patients"})
    }
    defer cursor.Close(ctx)

    var patients []models.Patient
    // ดึงข้อมูลทั้งหมดจาก cursor และเก็บใน slice patients
    if err = cursor.All(ctx, &patients); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot decode patients"})
    }

    return c.JSON(patients)
}

// เพิ่มฟังก์ชันสำหรับ Question

func (h *Handler) CreateQuestion(c *fiber.Ctx) error {
    var question models.Question
    if err := c.BodyParser(&question); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
    }

    question.CreatedAt = time.Now()
    question.UpdatedAt = time.Now()
    question.Status = "pending"
    question.ReadStatus = false

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := collection.InsertOne(ctx, question)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot insert question"})
    }

    question.ID = result.InsertedID.(primitive.ObjectID)

    // สร้าง Notification สำหรับ admin
    notification := models.Notification{
        UserID:     question.AdminID, // ถ้ามีการกำหนด AdminID ไว้
        QuestionID: question.ID,
        Message:    "New question received",
        IsRead:     false,
        CreatedAt:  time.Now(),
    }
    h.CreateNotification(notification)

    return c.Status(fiber.StatusCreated).JSON(question)
}

func (h *Handler) GetQuestionByID(c *fiber.Ctx) error {
    questionID := c.Params("id")
    log.Printf("Received question ID: %s", questionID) // เพิ่ม logging

    objectID, err := primitive.ObjectIDFromHex(questionID)
    if err != nil {
        log.Printf("Invalid ObjectID: %v", err) // เพิ่ม logging
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid question ID"})
    }

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var question models.Question
    err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&question)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            log.Printf("Question not found for ID: %s", questionID) // เพิ่ม logging
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Question not found"})
        }
        log.Printf("Database error: %v", err) // เพิ่ม logging
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
    }

    log.Printf("Question found: %+v", question) // เพิ่ม logging
    return c.JSON(question)
}

func (h *Handler) UpdateQuestionStatus(c *fiber.Ctx) error {
    questionID := c.Params("id")
    objectID, err := primitive.ObjectIDFromHex(questionID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid question ID"})
    }

    var updateData struct {
        Status string `json:"status"`
        Answer string `json:"answer,omitempty"`
    }
    if err := c.BodyParser(&updateData); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
    }

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    update := bson.M{
        "$set": bson.M{
            "status":    updateData.Status,
            "answer":    updateData.Answer,
            "updatedAt": time.Now(),
        },
    }

    result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot update question"})
    }

    if result.ModifiedCount == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Question not found"})
    }

    // สร้าง Notification สำหรับ user
    var question models.Question
    err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&question)
    if err == nil {
        notification := models.Notification{
            UserID:     question.UserID,
            QuestionID: question.ID,
            Message:    "Your question has been answered",
            IsRead:     false,
            CreatedAt:  time.Now(),
        }
        h.CreateNotification(notification)
    }

    return c.JSON(fiber.Map{"message": "Question updated successfully"})
}

func (h *Handler) ListQuestionsByUserID(c *fiber.Ctx) error {
    userID := c.Params("userId")
    objectID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
    }

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := collection.Find(ctx, bson.M{"userId": objectID})
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch questions"})
    }
    defer cursor.Close(ctx)

    var questions []models.Question
    if err = cursor.All(ctx, &questions); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot decode questions"})
    }

    return c.JSON(questions)
}

func (h *Handler) ListPendingQuestions(c *fiber.Ctx) error {
    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := collection.Find(ctx, bson.M{"status": "pending"})
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch pending questions"})
    }
    defer cursor.Close(ctx)

    var questions []models.Question
    if err = cursor.All(ctx, &questions); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot decode questions"})
    }

    return c.JSON(questions)
}

// เพิ่มฟังก์ชันสำหรับ Notification

func (h *Handler) CreateNotification(notification models.Notification) error {
    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("notifications")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := collection.InsertOne(ctx, notification)
    return err
}

func (h *Handler) GetNotificationsByUserID(c *fiber.Ctx) error {
    userID := c.Params("userId")
    objectID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
    }

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("notifications")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := collection.Find(ctx, bson.M{"userId": objectID})
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch notifications"})
    }
    defer cursor.Close(ctx)

    var notifications []models.Notification
    if err = cursor.All(ctx, &notifications); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot decode notifications"})
    }

    return c.JSON(notifications)
}

func (h *Handler) MarkNotificationAsRead(c *fiber.Ctx) error {
    notificationID := c.Params("id")
    objectID, err := primitive.ObjectIDFromHex(notificationID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid notification ID"})
    }

    collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("notifications")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    update := bson.M{
        "$set": bson.M{
            "isRead": true,
        },
    }

    result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot update notification"})
    }

    if result.ModifiedCount == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Notification not found"})
    }

    return c.JSON(fiber.Map{"message": "Notification marked as read"})
}