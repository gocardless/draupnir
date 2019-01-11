package routes

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/server/api"
	"github.com/gocardless/draupnir/server/api/auth"
	"github.com/gocardless/draupnir/server/api/middleware"
	"github.com/gocardless/draupnir/store"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
)

type Images struct {
	ImageStore    store.ImageStore
	InstanceStore store.InstanceStore
	Executor      exec.Executor
}

func (i Images) Get(w http.ResponseWriter, r *http.Request) error {
	logger, err := middleware.GetLogger(r)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		logger.Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	image, err := i.ImageStore.Get(id)
	if err != nil {
		logger.Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	err = jsonapi.MarshalOnePayload(w, &image)
	if err != nil {
		return errors.Wrap(err, "failed to marshal payload")
	}

	return nil
}

func (i Images) List(w http.ResponseWriter, r *http.Request) error {
	images, err := i.ImageStore.List()
	if err != nil {
		return errors.Wrap(err, "failed to get images")
	}

	// Build a slice of pointers to our images, because this is what jsonapi wants
	_images := make([]*models.Image, 0)
	for i := range images {
		_images = append(_images, &images[i])
	}

	return errors.Wrap(
		jsonapi.MarshalManyPayload(w, _images),
		"failed to marshal images",
	)
}

type CreateImageRequest struct {
	BackedUpAt time.Time `jsonapi:"attr,backed_up_at,iso8601"`
	Anon       string    `jsonapi:"attr,anonymisation_script"`
}

func (i Images) Create(w http.ResponseWriter, r *http.Request) error {
	logger, err := middleware.GetLogger(r)
	if err != nil {
		return err
	}

	req := CreateImageRequest{}
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		logger.Info(err.Error())
		api.InvalidJSONError.Render(w, http.StatusBadRequest)
		return nil
	}

	image := models.NewImage(req.BackedUpAt, req.Anon)
	image, err = i.ImageStore.Create(image)
	if err != nil {
		return errors.Wrap(err, "failed to create new image")
	}

	if err := i.Executor.CreateBtrfsSubvolume(r.Context(), image.ID); err != nil {
		return errors.Wrap(err, "failed to create btrfs subvolume")
	}

	w.WriteHeader(http.StatusCreated)
	if err := jsonapi.MarshalOnePayload(w, &image); err != nil {
		return errors.Wrap(err, "failed to marshal image")
	}

	return nil
}

func (i Images) Done(w http.ResponseWriter, r *http.Request) error {
	logger, err := middleware.GetLogger(r)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		logger.Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	image, err := i.ImageStore.Get(id)
	if err != nil {
		logger.Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	if !image.Ready {
		err = i.Executor.FinaliseImage(r.Context(), image)
		if err != nil {
			return errors.Wrap(err, "failed to finalise image")
		}

		image, err = i.ImageStore.MarkAsReady(image)
		if err != nil {
			return errors.Wrap(err, "failed to mark image as ready")
		}
	}

	w.WriteHeader(http.StatusOK)

	return errors.Wrap(
		jsonapi.MarshalOnePayload(w, &image),
		"failed to marshal image",
	)
}

func (i Images) Destroy(w http.ResponseWriter, r *http.Request) error {
	logger, err := middleware.GetLogger(r)
	if err != nil {
		return err
	}

	email, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		logger.Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	image, err := i.ImageStore.Get(id)
	if err != nil {
		logger.Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	if email == auth.UPLOAD_USER_EMAIL {
		// Destroy all instances of this image, if there are any
		instances, err := i.InstanceStore.List()
		for _, instance := range instances {
			if instance.ImageID != id {
				continue
			}
			logger.With("instance", instance.ID).Info("destroying instance")
			err = i.InstanceStore.Destroy(instance)
			if err == nil {
				err = i.Executor.DestroyInstance(r.Context(), instance.ID)
			}
			if err != nil {
				return errors.Wrap(err, "failed to destroy instance")
			}
		}
	}

	logger.With("image", id).Info("destroying image")
	err = i.ImageStore.Destroy(image)
	if err != nil {
		match, err := regexp.MatchString("instances_image_id_fkey", err.Error())
		if err == nil && match == true {
			logger.With("image", id).Info("cannot destroy image with instances")
			api.CannotDeleteImageWithInstancesError.Render(w, http.StatusUnprocessableEntity)
			return nil
		}

		return errors.Wrap(err, "failed to destroy image")
	}

	err = i.Executor.DestroyImage(r.Context(), id)
	if err != nil {
		return errors.Wrap(err, "failed to destroy image")
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}
