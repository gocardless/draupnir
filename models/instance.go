package models

import (
	"time"
)

type Instance struct {
	ID        int `jsonapi:"primary,instances"`
	ImageID   int `jsonapi:"attr,image_id"`
	UserEmail string
	CreatedAt time.Time `jsonapi:"attr,created_at,iso8601"`
	UpdatedAt time.Time `jsonapi:"attr,updated_at,iso8601"`
	Port      int       `jsonapi:"attr,port"`
}

func NewInstance(imageID int, email string) Instance {
	return Instance{
		ImageID:   imageID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserEmail: email,
	}
}
