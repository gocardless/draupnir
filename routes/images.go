package routes

import (
	"encoding/json"
	"fmt"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/store"
	"net/http"
	"time"
)

type Images struct {
	Store store.ImageStore
}

func (i Images) List(w http.ResponseWriter, r *http.Request) {
	images, err := i.Store.List()
	if err != nil {
		http.Error(w, "routes error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(images)
	if err != nil {
		http.Error(w, "json encoding failed", http.StatusInternalServerError)
		return
	}
}

type createRequest struct {
	BackedUpAt time.Time `json:"backed_up_at"`
}

func (i Images) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("json decoding failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	image := models.NewImage(req.BackedUpAt)
	image, err = i.Store.Create(models.NewImage(req.BackedUpAt))
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating image: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(image)
	if err != nil {
		http.Error(w, "json encoding failed", http.StatusInternalServerError)
		return
	}
}
