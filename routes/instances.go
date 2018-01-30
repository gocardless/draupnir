package routes

import (
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"

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
}

type CreateInstanceRequest struct {
	ImageID string `jsonapi:"attr,image_id"`
}

func (i Instances) Create(w http.ResponseWriter, r *http.Request) error {
	logger, err := GetLogger(r)
	if err != nil {
		return err
	}

	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return nil
	}

	req := CreateInstanceRequest{}
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusBadRequest, invalidJSONError)
		return nil
	}

	imageID, err := strconv.Atoi(req.ImageID)
	if err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusBadRequest, badImageIDError)
		return nil
	}

	image, err := i.ImageStore.Get(imageID)
	if err != nil {
		RenderError(w, http.StatusNotFound, imageNotFoundError)
		return nil
	}

	if !image.Ready {
		RenderError(w, http.StatusUnprocessableEntity, unreadyImageError)
		return nil
	}

	instance := models.NewInstance(imageID, email)
	instance.Port = generateRandomPort()
	instance, err = i.InstanceStore.Create(instance)
	if err != nil {

		match, err := regexp.MatchString("instances_image_id_fkey", err.Error())
		if err == nil && match == true {
			logger.Info(err.Error())
			RenderError(w, http.StatusNotFound, imageNotFoundError)
			return nil
		}

		return errors.Wrap(err, "failed to create instance")
	}

	if err := i.Executor.CreateInstance(imageID, instance.ID, instance.Port); err != nil {
		return errors.Wrap(err, "failed to create instance")
	}

	w.WriteHeader(http.StatusCreated)
	err = jsonapi.MarshalOnePayload(w, &instance)
	if err != nil {
		return errors.Wrap(err, "failed to marshal instance")
	}

	return nil
}

func (i Instances) List(w http.ResponseWriter, r *http.Request) error {
	logger, err := GetLogger(r)
	if err != nil {
		return err
	}

	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return nil
	}

	instances, err := i.InstanceStore.List()
	if err != nil {
		return errors.Wrap(err, "failed to get instances")
	}

	// Build a slice of pointers to our images, because this is what jsonapi wants
	// At the same time, filter out instances that don't belong to this user
	_instances := make([]*models.Instance, 0)
	for idx, instance := range instances {
		if instance.UserEmail == email {
			_instances = append(_instances, &instances[idx])
		}
	}

	return errors.Wrap(
		jsonapi.MarshalManyPayload(w, _instances),
		"failed to marshal instances",
	)
}

func (i Instances) Get(w http.ResponseWriter, r *http.Request) error {
	logger, err := GetLogger(r)
	if err != nil {
		return err
	}

	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return nil
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return nil
	}

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		logger.With("instance", id).Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return nil
	}

	if email != instance.UserEmail {
		RenderError(w, http.StatusNotFound, notFoundError)
		return nil
	}

	return errors.Wrap(
		jsonapi.MarshalOnePayload(w, &instance),
		"failed to marshal instance",
	)
}

func (i Instances) Destroy(w http.ResponseWriter, r *http.Request) error {
	logger, err := GetLogger(r)
	if err != nil {
		return err
	}

	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return nil
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		logger.Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return nil
	}

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		logger.With("instance", id).Info(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return nil
	}

	if email != auth.UPLOAD_USER_EMAIL && email != instance.UserEmail {
		RenderError(w, http.StatusNotFound, notFoundError)
		return nil
	}

	logger.With("instance", id).Info("destroying instance")
	err = i.InstanceStore.Destroy(instance)
	if err != nil {
		return errors.Wrap(err, "failed to destroy instance")
	}

	err = i.Executor.DestroyInstance(instance.ID)
	if err != nil {
		return errors.Wrap(err, "failed to destroy instance")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func generateRandomPort() int {
	const minPort = 5433
	const maxPort = 6000

	rand.Seed(time.Now().Unix())
	return minPort + rand.Intn(maxPort-minPort)
}
