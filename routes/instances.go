package routes

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/store"
	"github.com/google/jsonapi"
)

type Instances struct {
	Store    store.InstanceStore
	Executor exec.Executor
}

type createInstanceRequest struct {
	ImageID string `jsonapi:"attr,image_id"`
}

func (i Instances) Create(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != mediaType {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	w.Header().Set("Content-Type", mediaType)

	req := createInstanceRequest{}
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusBadRequest, badImageIDError)
		return
	}

	imageID, err := strconv.Atoi(req.ImageID)
	if err != nil {
		RenderError(w, http.StatusBadRequest, badImageIDError)
		return
	}

	instance := models.NewInstance(imageID)
	instance, err = i.Store.Create(instance)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	// Do some actual shit to create the instance

	w.WriteHeader(http.StatusCreated)
	err = jsonapi.MarshalOnePayload(w, &instance)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}
