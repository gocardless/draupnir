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

func (s DBInstanceStore) Create(instance models.Instance) (models.Instance, error) {
	row := s.DB.QueryRow(
		`INSERT INTO instances (image_id, port, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		instance.ImageID,
		instance.Port,
		instance.CreatedAt,
		instance.UpdatedAt,
	)

	err := row.Scan(&instance.ID)

	return instance, err
}
