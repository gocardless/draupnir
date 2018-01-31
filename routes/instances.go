package routes

import (
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/common/log"

	"github.com/gocardless/draupnir/auth"
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
	Authenticator auth.Authenticator
	Logger        log.Logger
}

type createInstanceRequest struct {
	ImageID string `jsonapi:"attr,image_id"`
}

func (i Instances) Create(w http.ResponseWriter, r *http.Request) {
	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		i.Logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	req := createInstanceRequest{}
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		i.Logger.Info(err.Error())
		RenderError(w, http.StatusBadRequest, invalidJSONError)
		return
	}

	imageID, err := strconv.Atoi(req.ImageID)
	if err != nil {
		i.Logger.Info(err.Error())
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

	instance := models.NewInstance(imageID, email)
	instance.Port = generateRandomPort()
	instance, err = i.InstanceStore.Create(instance)
	if err != nil {

		match, err := regexp.MatchString("instances_image_id_fkey", err.Error())
		if err == nil && match == true {
			i.Logger.Info(err.Error())
			RenderError(w, http.StatusNotFound, imageNotFoundError)
			return
		}

		i.Logger.With("error", err.Error()).With("http_request", r).Error("failed to create instance")
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	if err := i.Executor.CreateInstance(imageID, instance.ID, instance.Port); err != nil {
		i.Logger.With("error", err.Error()).With("http_request", r).Error("failed to create instance")
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = jsonapi.MarshalOnePayload(w, &instance)
	if err != nil {
		i.Logger.With("error", err.Error()).With("http_request", r).Error("failed to marshal instance")
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Instances) List(w http.ResponseWriter, r *http.Request) {
	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		i.Logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	instances, err := i.InstanceStore.List()
	if err != nil {
		i.Logger.With("error", err.Error()).With("http_request", r).Error("failed to list instances")
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	// Build a slice of pointers to our images, because this is what jsonapi wants
	// At the same time, filter out instances that don't belong to this user
	_instances := make([]*models.Instance, 0)
	for idx, instance := range instances {
		if instance.UserEmail == email {
			_instances = append(_instances, &instances[idx])
		}
	}

	err = jsonapi.MarshalManyPayload(w, _instances)
	if err != nil {
		i.Logger.With("error", err.Error()).With("http_request", r).Error("failed to marshal instances")
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Instances) Get(w http.ResponseWriter, r *http.Request) {
	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		i.Logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		i.Logger.Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		i.Logger.With("instance", id).Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	if email != instance.UserEmail {
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	err = jsonapi.MarshalOnePayload(w, &instance)
	if err != nil {
		i.Logger.With("error", err.Error()).With("http_request", r).With("instance", id).
			Error("failed to marshal instance")
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Instances) Destroy(w http.ResponseWriter, r *http.Request) {
	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		i.Logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		i.Logger.Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		i.Logger.With("instance", id).Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	if email != auth.UPLOAD_USER_EMAIL && email != instance.UserEmail {
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	i.Logger.With("instance", id).Info("destroying instance")
	err = i.InstanceStore.Destroy(instance)
	if err != nil {
		i.Logger.With("error", err.Error()).With("http_request", r).With("instance", id).
			Info("failed to destroy instance")
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	err = i.Executor.DestroyInstance(instance.ID)
	if err != nil {
		i.Logger.With("error", err.Error()).With("http_request", r).With("instance", id).
			Info("failed to destroy instance")
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
