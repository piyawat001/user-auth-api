//backend/user-auth-api/models/models.go 
package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Username  string             `json:"username" bson:"username"`
	Email     string             `json:"email" bson:"email"`
	Password  string             `json:"password" bson:"password"`
	Role      string             `json:"role" bson:"role"` // Role can be Admin, Faculty of Dentistry, Affiliated Hospitals, Other
	Status    string             `json:"status" bson:"status"` // Active, Inactive, etc.
	Package   string             `json:"package" bson:"package"` // Basic, Plus, Premium
	Hospital  string             `json:"hospital" bson:"hospital"` // Hospital name
	CreatedAt time.Time          `json:"created_at" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updatedAt"`
}

type Package struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"` // Basic, Plus, Premium
	Description string             `json:"description" bson:"description"`
	Price       float64            `json:"price" bson:"price"`
	Features    []string           `json:"features" bson:"features"`
}


type Patient struct {
	ID               primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ImageName        string             `json:"image_name" bson:"image_name"`
	Confirmation     string             `json:"confirm" bson:"confirm"`                       // Agree or Disagree
	Age              int                `json:"age" bson:"age"`                               // Consider using int for age
	Gender           string             `json:"gender" bson:"gender"`                         // Male or Female
	DurationOfLesion string             `json:"duration_of_lesion" bson:"duration_of_lesion"` // weeks, months, years
	Expansion        string             `json:"expansion" bson:"expansion"`                   // Buccolingual, Anteroposterior
	Paresthesia      bool               `json:"paresthesia" bson:"paresthesia"`               // Yes or No
	NumberOfLesions  string             `json:"number_of_lesions" bson:"number_of_lesions"`   // Single lesion, Multiple lesions
	CreatedAt        time.Time          `json:"created_at" bson:"createdAt"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updatedAt"`
}

type Question struct {
	ID           primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	AdminID      primitive.ObjectID `json:"admin_id,omitempty" bson:"admin_id,omitempty"`
	Title        string             `json:"title" bson:"title"`
	Content      string             `json:"content" bson:"content"`
	Status       string             `json:"status" bson:"status"` // "pending", "inProgress", "answered", "closed", "deleted"
	Answer       string             `json:"answer,omitempty" bson:"answer,omitempty"`
	IsEdited     bool               `json:"is_edited" bson:"is_edited"`
	EditHistory  []EditEntry        `json:"edit_history,omitempty" bson:"edit_history,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
	ReadStatus   struct {
		User             bool `json:"user" bson:"user"`
		Admin            bool `json:"admin" bson:"admin"`
		NotificationBell bool `json:"notification_bell" bson:"notification_bell"`
	} `json:"read_status" bson:"read_status"`
}

type EditEntry struct {
	Content  string    `json:"content" bson:"content"`
	EditedAt time.Time `json:"edited_at" bson:"edited_at"`
	EditedBy string    `json:"edited_by" bson:"edited_by"`
}

type Notification struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ReceiverID  primitive.ObjectID `json:"receiver_id" bson:"receiver_id"`
	SenderID    primitive.ObjectID `json:"sender_id" bson:"sender_id"`
	QuestionID  primitive.ObjectID `json:"question_id" bson:"question_id"`
	Type        string             `json:"type" bson:"type"`
	Message     string             `json:"message" bson:"message"`
	IsRead      bool               `json:"is_read" bson:"is_read"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	RedirectURL string             `json:"redirect_url" bson:"redirect_url"`
}

