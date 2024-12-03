package handlers

import (
	"context"
	"os"
	"strconv"
	"time"

	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"github.com/piyawat001/user-auth-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
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
		Identifier string `json:"identifier"` // รับค่าเป็นทั้ง email หรือ username
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
		"id":        user.ID.Hex(),  // ส่ง ID ของผู้ใช้
		"username":  user.Username,  // ส่ง username
		"email":     user.Email,     // ส่ง email
		"password":  user.Password,  // ส่ง password (ถ้าจำเป็น แต่ควรปกป้องข้อมูล)
		"role":      user.Role,      // ส่ง role
		"status":    user.Status,    // ส่งสถานะ
		"package":   user.Package,   // ส่ง package
		"hospital":  user.Hospital,  // ส่งชื่อโรงพยาบาล
		"createdAt": user.CreatedAt, // ส่งวันที่สร้าง
		"updatedAt": user.UpdatedAt, // ส่งวันที่อัพเดตล่าสุด
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
		UserID     string `json:"user_id"`
		Package    string `json:"package"`
		Role       string `json:"role"`
		ExpiryDays int    `json:"expiry_days"` // Days until package expires
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

// ฟังก์ชันสำหรับสร้างคำถามใหม่
func (h *Handler) CreateQuestion(c *fiber.Ctx) error {
	var question models.Question
	if err := c.BodyParser(&question); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Validation
	if question.Title == "" || question.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title and content are required"})
	}

	// Set default values
	question.CreatedAt = time.Now()
	question.UpdatedAt = time.Now()
	question.Status = "pending"
	question.ReadStatus.User = true
	question.ReadStatus.Admin = false
	question.ReadStatus.NotificationBell = false

	// Optionally, set AdminID to null initially
	question.AdminID = primitive.NilObjectID

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, question)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot create question"})
	}

	question.ID = result.InsertedID.(primitive.ObjectID)

	// Create a notification (optional)
	go func() {
		notifCollection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("notifications")
		notification := models.Notification{
			ReceiverID:  primitive.NilObjectID, // Admin can be added later
			SenderID:    question.UserID,
			QuestionID:  question.ID,
			Type:        "new_question",
			Message:     fmt.Sprintf("New Question: %s", question.Title),
			IsRead:      false,
			CreatedAt:   time.Now(),
			RedirectURL: fmt.Sprintf("/questions/%s", question.ID.Hex()),
		}

		_, err := notifCollection.InsertOne(ctx, notification)
		if err != nil {
			fmt.Printf("Error creating notification: %v\n", err)
		}
	}()

	return c.Status(fiber.StatusCreated).JSON(question)
}

// GetUserQuestions ดึงประวัติคำถามของผู้ใช้
func (h *Handler) GetMyQuestions(c *fiber.Ctx) error {
	userID := c.Params("userId")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Parse query parameters for pagination
	page := c.Query("page", "1")
	pageSize := c.Query("page_size", "10")
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt <= 0 {
		pageInt = 1
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt <= 0 {
		pageSizeInt = 10
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSkip(int64((pageInt - 1) * pageSizeInt)).SetLimit(int64(pageSizeInt)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(ctx, bson.M{"user_id": objectID}, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch questions"})
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot decode questions"})
	}

	return c.JSON(fiber.Map{
		"total":     len(questions),
		"questions": questions,
		"page":      pageInt,
		"page_size": pageSizeInt,
	})
}

// GetQuestionDetail ดูรายละเอียดคำถามเฉพาะข้อ
func (h *Handler) GetQuestionDetail(c *fiber.Ctx) error {
	questionID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(questionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid question ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var question models.Question
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&question)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Question not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot fetch question"})
	}

	// อัพเดต read status
	if !question.ReadStatus.User {
		update := bson.M{
			"$set": bson.M{
				"read_status.user": true,
				"updated_at":       time.Now(),
			},
		}
		_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
		if err != nil {
			fmt.Printf("Error updating read status: %v\n", err)
		}
	}

	return c.JSON(question)
}

func (h *Handler) GetAllQuestions(c *fiber.Ctx) error {
	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
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

func (h *Handler) GetQuestionsByUser(c *fiber.Ctx) error {
	userID := c.Params("userId")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{"user_id": objectID})
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

func (h *Handler) UpdateNotificationBellStatus(c *fiber.Ctx) error {

	userID := c.Params("userId")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// อัพเดตทุกคำถามของผู้ใช้
	result, err := collection.UpdateMany(
		ctx,
		bson.M{"user_id": objectID},
		bson.M{
			"$set": bson.M{
				"read_status.notification_bell": true,
				"updated_at":                    time.Now(),
			},
		},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot update notification bell status",
		})
	}

	return c.JSON(fiber.Map{
		"message":        "Notification bell status updated successfully",
		"modified_count": result.ModifiedCount,
	})
}

// / DeleteQuestion ลบคำถาม
func (h *Handler) DeleteQuestion(c *fiber.Ctx) error {
	questionID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(questionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid question ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ลบคำถามตาม ID ที่ระบุ
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot delete question"})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Question not found"})
	}

	return c.JSON(fiber.Map{"message": "Question deleted successfully"})
}

// Notification Handlers
func (h *Handler) GetUserNotifications(c *fiber.Ctx) error {
	userID := c.Params("userId")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{"receiver_id": objectID})
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
			"is_read": true,
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

func (h *Handler) GetNotificationCount(c *fiber.Ctx) error {
	userID := c.Params("id")        // เปลี่ยนเป็น "id" แทน "userId"
	role := c.Query("role", "user") // Default to user, can specify admin

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var filter bson.M
	if role == "admin" {
		// Count notifications for admin (e.g., pending questions)
		filter = bson.M{
			"receiver_id": objectID,
			"type":        "new_question", // ตัวอย่างเช่น การแจ้งเตือนคำถามใหม่
			"is_read":     false,
		}
	} else {
		// Count unread notifications for user
		filter = bson.M{
			"receiver_id": objectID,
			"is_read":     false,
		}
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot count notifications"})
	}

	return c.JSON(fiber.Map{
		"unread_count": count,
		"role":         role,
	})
}

// ฟังก์ชันสำหรับดึงคำถามที่ admin ยังไม่ได้ตอบ
func (h *Handler) GetPendingQuestions(c *fiber.Ctx) error {
	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ค้นหาคำถามที่มีสถานะเป็น "pending" หรือ "inProgress"
	cursor, err := collection.Find(ctx, bson.M{
		"status": bson.M{"$in": []string{"pending", "inProgress"}},
	})
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

// UpdateQuestion อัพเดตคำถามหรือคำตอบ
func (h *Handler) UpdateQuestion(c *fiber.Ctx) error {
	questionID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(questionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid question ID"})
	}

	var updateData struct {
		Content string `json:"content,omitempty"`
		Answer  string `json:"answer,omitempty"`
		Status  string `json:"status,omitempty"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{"updated_at": time.Now()},
	}

	if updateData.Content != "" {
		update["$set"].(bson.M)["content"] = updateData.Content
		update["$set"].(bson.M)["is_edited"] = true
		update["$push"] = bson.M{
			"edit_history": bson.M{
				"content":   updateData.Content,
				"edited_at": time.Now(),
				"edited_by": "admin", // Replace with actual user ID
			},
		}
	}

	if updateData.Answer != "" {
		update["$set"].(bson.M)["answer"] = updateData.Answer
		update["$set"].(bson.M)["status"] = "answered"
		update["$set"].(bson.M)["read_status.admin"] = true
		update["$set"].(bson.M)["read_status.user"] = false
	}

	if updateData.Status != "" {
		update["$set"].(bson.M)["status"] = updateData.Status
	}

	if len(update["$set"].(bson.M)) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No valid fields to update"})
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot update question"})
	}

	if result.ModifiedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Question not found or no changes made"})
	}

	return c.JSON(fiber.Map{"message": "Question updated successfully"})
}
