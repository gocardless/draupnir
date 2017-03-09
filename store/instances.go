package store

import (
	"database/sql"
	"github.com/gocardless/draupnir/models"
	_ "github.com/lib/pq" // used to setup the PG driver
)

type InstanceStore interface {
	Create(models.Instance) (models.Instance, error)
	List() ([]models.Instance, error)
	Get(id int) (models.Instance, error)
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

func (s DBInstanceStore) List() ([]models.Instance, error) {
	instances := make([]models.Instance, 0)

	rows, err := s.DB.Query(`SELECT id, image_id, port, created_at, updated_at FROM instances`)
	if err != nil {
		return instances, err
	}

	defer rows.Close()

	var instance models.Instance
	for rows.Next() {
		err = rows.Scan(
			&instance.ID,
			&instance.ImageID,
			&instance.Port,
			&instance.CreatedAt,
			&instance.UpdatedAt,
		)

		if err != nil {
			return instances, err
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

func (s DBInstanceStore) Get(id int) (models.Instance, error) {
	instance := models.Instance{}

	row := s.DB.QueryRow("SELECT id, image_id, port, created_at, updated_at FROM instances WHERE id = $1", id)
	err := row.Scan(
		&instance.ID,
		&instance.ImageID,
		&instance.Port,
		&instance.CreatedAt,
		&instance.UpdatedAt,
	)
	if err != nil {
		return instance, err
	}

	return instance, nil
}
