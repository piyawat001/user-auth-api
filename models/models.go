package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Username   string             `json:"username" bson:"username"`
	Email      string             `json:"email" bson:"email"`
	Password   string             `json:"password" bson:"password"`
	Role       string             `json:"role" bson:"role"`
	Status     string             `json:"status" bson:"status"`
	Package    string             `json:"package" bson:"package"`
	Hospital   string             `json:"hospital" bson:"hospital"`  // ใช้ชื่อโรงพยาบาลแทน ID
	CreatedAt  time.Time          `json:"created_at" bson:"createdAt"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updatedAt"`
}

type Package struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Price       float64            `json:"price" bson:"price"`
	Features    []string           `json:"features" bson:"features"`
}

type Patient struct {
	ID                  primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ImageName           string             `json:"image_name" bson:"image_name"`
	Confirmation        string             `json:"confirm" bson:"confirm"` // Agree or Disagree
	Age                 int                `json:"age" bson:"age"`         // Consider using int for age
	Gender              string             `json:"gender" bson:"gender"`   // Male or Female
	DurationOfLesion    string             `json:"duration_of_lesion" bson:"duration_of_lesion"` // weeks, months, years
	Expansion           string             `json:"expansion" bson:"expansion"` // Buccolingual, Anteroposterior
	Paresthesia         bool               `json:"paresthesia" bson:"paresthesia"` // Yes or No
	NumberOfLesions     string             `json:"number_of_lesions" bson:"number_of_lesions"` // Single lesion, Multiple lesions
	CreatedAt           time.Time          `json:"created_at" bson:"createdAt"`
	UpdatedAt           time.Time          `json:"updated_at" bson:"updatedAt"`
}

type Question struct {
    ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
    UserID     primitive.ObjectID `json:"user_id" bson:"user_id"`
    AdminID    primitive.ObjectID `json:"admin_id,omitempty" bson:"admin_id,omitempty"`
    Title      string             `json:"title" bson:"title"`
    Content    string             `json:"content" bson:"content"`
    Status     string             `json:"status" bson:"status"` // "pending", "answered", "closed"
    Answer     string             `json:"answer,omitempty" bson:"answer,omitempty"`
    CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
    UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at"`
    ReadStatus bool               `json:"read_status" bson:"read_status"`
}

type Notification struct {
    ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
    UserID     primitive.ObjectID `json:"user_id" bson:"user_id"`
    QuestionID primitive.ObjectID `json:"question_id" bson:"question_id"`
    Message    string             `json:"message" bson:"message"`
    IsRead     bool               `json:"is_read" bson:"is_read"`
    CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
}