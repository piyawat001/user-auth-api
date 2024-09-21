package handlers

import (
	"context"
	"os"
	"time"

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
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&loginUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	collection := h.client.Database(os.Getenv("DATABASE_NAME")).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": loginUser.Email}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginUser.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.ID
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot generate token"})
	}

	// Include the user's role in the response
	return c.JSON(fiber.Map{
		"token": t,
		"role":  user.Role, // Add this line to include the role
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
