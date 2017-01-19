package store

import (
	"database/sql"
	"github.com/gocardless/draupnir/models"
	_ "github.com/lib/pq" // used to setup the PG driver
)

type InstanceStore interface {
	Create(models.Instance) (models.Instance, error)
}

type DBInstanceStore struct {
	DB *sql.DB
}

func (s DBInstanceStore) Create(image models.Instance) (models.Instance, error) {
	row := s.DB.QueryRow(
		"INSERT INTO instances (image_id, created_at, updated_at) VALUES ($1, $2, $3) RETURNING *",
		image.ImageID,
		image.CreatedAt,
		image.UpdatedAt,
	)

	err := row.Scan(
		&image.ID,
		&image.ImageID,
		&image.CreatedAt,
		&image.UpdatedAt,
	)

	return image, err
}
