package models

import (
	"time"
)

type Instance struct {
	ID        int       `json:"id"`
	ImageID   int       `json:"image_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewInstance(imageID int) Instance {
	return Instance{
		ImageID:   imageID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
