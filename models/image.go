package models

import (
  "time"
)

type Image struct {
  ID int `json:"id"`
  BackedUpAt time.Time `json:"backed_up_at"`
  Ready bool `json:"ready"`
}
