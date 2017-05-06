package routes

import (
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/store"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
)

type Instances struct {
	InstanceStore store.InstanceStore
	ImageStore    store.ImageStore
	Executor      exec.Executor
}

type createInstanceRequest struct {
	ImageID string `jsonapi:"attr,image_id"`
}

func (i Instances) Create(w http.ResponseWriter, r *http.Request) {
	req := createInstanceRequest{}
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusBadRequest, invalidJSONError)
		return
	}

	imageID, err := strconv.Atoi(req.ImageID)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusBadRequest, badImageIDError)
		return
	}

	image, err := i.ImageStore.Get(imageID)
	if err != nil {
		RenderError(w, http.StatusNotFound, imageNotFoundError)
		return
	}

	if !image.Ready {
		RenderError(w, http.StatusUnprocessableEntity, unreadyImageError)
		return
	}

	instance := models.NewInstance(imageID)
	instance.Port = generateRandomPort()
	instance, err = i.InstanceStore.Create(instance)
	if err != nil {
		log.Print(err.Error())

		match, err := regexp.MatchString("instances_image_id_fkey", err.Error())
		if err == nil && match == true {
			RenderError(w, http.StatusNotFound, imageNotFoundError)
			return
		}

		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	if err := i.Executor.CreateInstance(imageID, instance.ID, instance.Port); err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = jsonapi.MarshalOnePayload(w, &instance)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Instances) List(w http.ResponseWriter, r *http.Request) {
	instances, err := i.InstanceStore.List()
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	// Build a slice of pointers to our images, because this is what jsonapi wants
	_instances := make([]*models.Instance, 0)
	for i := range instances {
		_instances = append(_instances, &instances[i])
	}

	err = jsonapi.MarshalManyPayload(w, _instances)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, internalServerError)
		log.Print(err.Error())
		return
	}
}

func (i Instances) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	err = jsonapi.MarshalOnePayload(w, &instance)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Instances) Destroy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	err = i.InstanceStore.Destroy(instance)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	err = i.Executor.DestroyInstance(instance.ID)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func generateRandomPort() int {
	const minPort = 5433
	const maxPort = 6000

	rand.Seed(time.Now().Unix())
	return minPort + rand.Intn(maxPort-minPort)
}
