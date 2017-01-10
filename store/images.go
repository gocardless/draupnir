package store

import (
  "database/sql"
  _ "github.com/lib/pq" // used to setup the PG driver
  "github.com/gocardless/draupnir/models"
)

type ImageStore interface {
  List() ([]models.Image, error)
}

type DBImageStore struct {
  DB *sql.DB
}

func (s DBImageStore) List() ([]models.Image, error) {
  images := make([]models.Image, 0)

  rows, err := s.DB.Query("SELECT * from images")
  defer rows.Close()
  if err != nil {
    return images, err
  }

  var image models.Image
  for rows.Next() {
    err = rows.Scan(
      &image.ID,
      &image.BackedUpAt,
      &image.Ready,
    )

    if err != nil {
      return images, err
    }

    images = append(images, image)
  }

  return images, nil
}
