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
		RenderError(w, http.StatusBadRequest, invalidJSONError)
		return
	}

	imageID, err := strconv.Atoi(req.ImageID)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusBadRequest, badImageIDError)
		return
	}

	// TODO: check if the image id corresponds to a real image,
	//       and that image is ready

	instance := models.NewInstance(imageID)
	instance.Port = generateRandomPort()
	instance, err = i.Store.Create(instance)
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

func generateRandomPort() int {
	const minPort = 1025
	const maxPort = 49152

	rand.Seed(time.Now().Unix())
	return minPort + rand.Intn(maxPort-minPort)
}
