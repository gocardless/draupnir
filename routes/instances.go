package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/store"
)

type Instances struct {
	Store    store.InstanceStore
	Executor exec.Executor
}

type createInstanceRequest struct {
	ImageID int `json:"id"`
}

func (i Instances) Create(w http.ResponseWriter, r *http.Request) {
	var req createInstanceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("json decoding failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	instance := models.NewInstance(req.ImageID)
	instance, err = i.Store.Create(instance)
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating instance: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Do some actual shit to create the instance

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(instance)
	if err != nil {
		http.Error(w, "json encoding failed", http.StatusInternalServerError)
		return
	}
}
