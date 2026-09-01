package models

type Tag struct {
	BaseModel
	Name        string `gorm:"not null;size:255;uniqueIndex" json:"name"`
	Description string `json:"description"`
}
