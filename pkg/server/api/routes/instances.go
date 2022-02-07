package routes

import (
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/gocardless/draupnir/pkg/exec"
	"github.com/gocardless/draupnir/pkg/models"
	"github.com/gocardless/draupnir/pkg/server/api"
	"github.com/gocardless/draupnir/pkg/server/api/auth"
	"github.com/gocardless/draupnir/pkg/server/api/middleware"
	"github.com/gocardless/draupnir/pkg/store"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
)

type Instances struct {
	InstanceStore           store.InstanceStore
	ImageStore              store.ImageStore
	WhitelistedAddressStore store.WhitelistedAddressStore
	ApplyWhitelist          func(string)
	Executor                exec.Executor
	MinInstancePort         uint16
	MaxInstancePort         uint16
}

type CreateInstanceRequest struct {
	ImageID string `jsonapi:"attr,image_id"`
}

func (i Instances) Create(w http.ResponseWriter, r *http.Request) error {
	logger, err := middleware.GetLogger(r)
	if err != nil {
		return err
	}

	email, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		return err
	}

	req := CreateInstanceRequest{}
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		logger.Info(err.Error())
		api.InvalidJSONError.Render(w, http.StatusBadRequest)
		return nil
	}

	imageID, err := strconv.Atoi(req.ImageID)
	if err != nil {
		logger.Info(err.Error())
		api.BadImageIDError.Render(w, http.StatusBadRequest)
		return nil
	}

	image, err := i.ImageStore.Get(imageID)
	if err != nil {
		api.ImageNotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	if !image.Ready {
		api.UnreadyImageError.Render(w, http.StatusUnprocessableEntity)
		return nil
	}

	refreshToken, ok := r.Context().Value(middleware.RefreshTokenKey).(string)
	if !ok {
		log.Fatal("Access token key is missing from context")
	}

	instance := models.NewInstance(imageID, email, refreshToken)
	port, err := generateRandomFreePort(i.InstanceStore, i.MinInstancePort, i.MaxInstancePort)
	if err != nil {
		return err
	}
	instance.Port = port

	instance, err = i.InstanceStore.Create(instance)

	if err != nil {
		match, err := regexp.MatchString("instances_image_id_fkey", err.Error())
		if err == nil && match == true {
			logger.Info(err.Error())
			api.ImageNotFoundError.Render(w, http.StatusNotFound)
			return nil
		}

		return errors.Wrap(err, "failed to create instance")
	}

	ipaddr, err := middleware.GetUserIPAddress(r)
	if err != nil {
		return err
	}

	if err := i.Executor.CreateInstance(r.Context(), imageID, instance.ID, int(instance.Port)); err != nil {
		return errors.Wrap(err, "failed to create instance")
	}

	files, err := i.Executor.RetrieveInstanceCredentials(r.Context(), instance.ID)
	if err != nil {
		logger.With("instance", instance.ID).Info(
			errors.Wrap(err, "failed to retrieve instance credentials"),
		)
		api.InternalServerError.Render(w, http.StatusInternalServerError)
		return nil
	}

	creds := models.NewInstanceCredentials(
		instance.ID,
		string(files["ca.crt"]), string(files["client.crt"]), string(files["client.key"]),
	)
	instance.Credentials = &creds

	// Add the user's IP address to the whitelist
	address := models.NewWhitelistedAddress(ipaddr, &instance)
	address, err = i.WhitelistedAddressStore.Create(address)
	if err != nil {
		return errors.Wrap(err, "failed to record whitelisted IP address")
	}
	i.ApplyWhitelist("api")

	w.WriteHeader(http.StatusCreated)
	err = jsonapi.MarshalOnePayload(w, &instance)
	if err != nil {
		return errors.Wrap(err, "failed to marshal instance")
	}

	return nil
}

func (i Instances) List(w http.ResponseWriter, r *http.Request) error {
	email, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		return err
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

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		logger.With("instance", id).Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	if email != instance.UserEmail {
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	ipaddr, err := middleware.GetUserIPAddress(r)
	if err != nil {
		return err
	}

	files, err := i.Executor.RetrieveInstanceCredentials(r.Context(), instance.ID)
	if err != nil {
		logger.With("instance", id).Info(
			errors.Wrap(err, "failed to retrieve instance credentials"),
		)
		api.InternalServerError.Render(w, http.StatusInternalServerError)
		return nil
	}

	creds := models.NewInstanceCredentials(
		instance.ID,
		string(files["ca.crt"]), string(files["client.crt"]), string(files["client.key"]),
	)
	instance.Credentials = &creds

	// Add the user's IP address to the whitelist
	address := models.NewWhitelistedAddress(ipaddr, &instance)
	address, err = i.WhitelistedAddressStore.Create(address)
	if err != nil {
		return errors.Wrap(err, "failed to record whitelisted IP address")
	}
	i.ApplyWhitelist("api")

	return errors.Wrap(
		jsonapi.MarshalOnePayload(w, &instance),
		"failed to marshal instance",
	)
}

func (i Instances) Destroy(w http.ResponseWriter, r *http.Request) error {
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

	instance, err := i.InstanceStore.Get(id)
	if err != nil {
		logger.With("instance", id).Info(err.Error())
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	if email != auth.UPLOAD_USER_EMAIL && email != instance.UserEmail {
		api.NotFoundError.Render(w, http.StatusNotFound)
		return nil
	}

	logger.With("instance", id).Info("destroying instance")
	err = i.Executor.DestroyInstance(r.Context(), instance.ID)
	if err != nil {
		return errors.Wrap(err, "failed to destroy instance on disk")
	}

	err = i.InstanceStore.Destroy(instance)
	if err != nil {
		return errors.Wrap(err, "failed to remove instance from table")
	}

	// Destroying the instance will cascade and destroy any linked whitelisted
	// addresses. Trigger the whitelist reconciler in order to clean up the
	// obsolete rule.
	i.ApplyWhitelist("api")

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func generateRandomFreePort(store store.InstanceStore, minPort uint16, maxPort uint16) (uint16, error) {
	attempts := 0
	port := uint16(0)
	portAvailable := false

GetNewPort:
	for !portAvailable {
		attempts++
		if attempts >= 100 {
			return port, errors.Errorf("No free port found after %d attempts", attempts)
		}

		rand.Seed(time.Now().Unix() + int64(time.Now().Nanosecond()))
		port = minPort + uint16(rand.Intn(int(maxPort-minPort)))

		instances, err := store.List()
		if err != nil {
			return port, errors.Wrap(err, "failed to list instances to determine free port")
		}

		for _, instance := range instances {
			if instance.Port == port {
				goto GetNewPort
			}
		}
		portAvailable = true
	}

	return port, nil
}
