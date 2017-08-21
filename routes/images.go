package routes

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gocardless/draupnir/auth"
	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/store"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
)

type Images struct {
	ImageStore    store.ImageStore
	InstanceStore store.InstanceStore
	Executor      exec.Executor
	Authenticator auth.Authenticator
}

const mediaType = "application/json"

func (i Images) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mediaType)

	_, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	image, err := i.ImageStore.Get(id)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	err = jsonapi.MarshalOnePayload(w, &image)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Images) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mediaType)

	_, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	images, err := i.ImageStore.List()
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	// Build a slice of pointers to our images, because this is what jsonapi wants
	_images := make([]*models.Image, 0)
	for i := range images {
		_images = append(_images, &images[i])
	}

	err = jsonapi.MarshalManyPayload(w, _images)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, internalServerError)
		log.Print(err.Error())
		return
	}
}

type createImageRequest struct {
	BackedUpAt time.Time `jsonapi:"attr,backed_up_at,iso8601"`
	Anon       string    `jsonapi:"attr,anonymisation_script"`
}

func (i Images) Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mediaType)

	_, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	req := createImageRequest{}
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusBadRequest, invalidJSONError)
		return
	}

	image := models.NewImage(req.BackedUpAt, req.Anon)
	image, err = i.ImageStore.Create(image)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	if err := i.Executor.CreateBtrfsSubvolume(image.ID); err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := jsonapi.MarshalOnePayload(w, &image); err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Images) Done(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mediaType)

	_, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	image, err := i.ImageStore.Get(id)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	if !image.Ready {
		err = i.Executor.FinaliseImage(image)
		if err != nil {
			log.Print(err.Error())
			RenderError(w, http.StatusInternalServerError, internalServerError)
			return
		}

		image, err = i.ImageStore.MarkAsReady(image)
		if err != nil {
			log.Print(err.Error())
			RenderError(w, http.StatusInternalServerError, internalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	err = jsonapi.MarshalOnePayload(w, &image)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}
}

func (i Images) Destroy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mediaType)

	email, err := i.Authenticator.AuthenticateRequest(r)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusUnauthorized, unauthorizedError)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	image, err := i.ImageStore.Get(id)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusNotFound, notFoundError)
		return
	}

	if email == auth.UPLOAD_USER_EMAIL {
		// Destroy all instances of this image, if there are any
		instances, err := i.InstanceStore.List()
		for _, instance := range instances {
			if instance.ImageID != id {
				continue
			}
			err = i.InstanceStore.Destroy(instance)
			if err == nil {
				err = i.Executor.DestroyInstance(instance.ID)
			}
			if err != nil {
				log.Print(err.Error())
				RenderError(w, http.StatusInternalServerError, internalServerError)
				return
			}
		}
	}

	err = i.ImageStore.Destroy(image)
	if err != nil {
		log.Print(err.Error())

		match, err := regexp.MatchString("instances_image_id_fkey", err.Error())
		if err != nil {
			log.Print(err.Error())
		}

		if match == true {
			RenderError(
				w,
				http.StatusUnprocessableEntity,
				cannotDeleteImageWithInstancesError,
			)
			return
		}

		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	err = i.Executor.DestroyImage(id)
	if err != nil {
		log.Print(err.Error())
		RenderError(w, http.StatusInternalServerError, internalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
