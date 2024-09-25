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
