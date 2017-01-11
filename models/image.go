package models

import (
	"time"
)

type Image struct {
	ID         int       `json:"id"`
	BackedUpAt time.Time `json:"backed_up_at"`
	Ready      bool      `json:"ready"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewImage(backedUpAt time.Time) Image {
	return Image{
		BackedUpAt: backedUpAt,
		Ready:      false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}
