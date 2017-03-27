package models

import (
	"time"
)

type Image struct {
	ID         int       `jsonapi:"primary,images"`
	BackedUpAt time.Time `jsonapi:"attr,backed_up_at,iso8601"`
	Ready      bool      `jsonapi:"attr,ready"`
	Anon       string
	CreatedAt  time.Time `jsonapi:"attr,created_at,iso8601"`
	UpdatedAt  time.Time `jsonapi:"attr,updated_at,iso8601"`
}

func NewImage(backedUpAt time.Time, anon string) Image {
	return Image{
		BackedUpAt: backedUpAt,
		Ready:      false,
		Anon:       anon,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}
