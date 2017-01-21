package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/store"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
)

type Images struct {
	Store    store.ImageStore
	Executor exec.Executor
}

func (i Images) List(w http.ResponseWriter, r *http.Request) {
	images, err := i.Store.List()
	if err != nil {
		RenderError(w, 500, internalServerError)
		return
	}

	// Build a slice of pointers to our images, because this is what jsonapi wants
	var _images []*models.Image
	for _, i := range images {
		_images = append(_images, &i)
	}
	err = jsonapi.MarshalManyPayload(w, _images)
	if err != nil {
		RenderError(w, 500, internalServerError)
		return
	}
}

type createImageRequest struct {
	BackedUpAt time.Time `json:"backed_up_at"`
}

func (i Images) Create(w http.ResponseWriter, r *http.Request) {
	var req createImageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		RenderError(w, 500, internalServerError)
		return
	}

	image := models.NewImage(req.BackedUpAt)
	image, err = i.Store.Create(image)
	if err != nil {
		RenderError(w, 500, internalServerError)
		return
	}

	err = i.Executor.CreateBtrfsSubvolume(image.ID)
	if err != nil {
		RenderError(w, 500, internalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = jsonapi.MarshalOnePayload(w, &image)
	if err != nil {
		RenderError(w, 500, internalServerError)
		return
	}
}

func (i Images) Done(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		RenderError(w, 404, notFoundError)
		return
	}

	image, err := i.Store.Get(id)
	if err != nil {
		RenderError(w, 404, notFoundError)
		return
	}

	if !image.Ready {
		err = i.Executor.FinaliseImage(image.ID)
		if err != nil {
			log.Print(err.Error())
			RenderError(w, 500, internalServerError)
			return
		}

		image, err = i.Store.MarkAsReady(image)
		if err != nil {
			log.Print(err.Error())
			RenderError(w, 500, internalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	err = jsonapi.MarshalOnePayload(w, &image)
	if err != nil {
		RenderError(w, 500, internalServerError)
		return
	}
}
