package client

import (
	"bytes"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gocardless/draupnir/models"
	"github.com/google/jsonapi"
)

// Client represents the client for a draupnir server
type Client struct {
	// The URL of the draupnir server
	// e.g. "https://draupnir-server.my-infra.io"
	URL string
}

// DraupnirClient defines the API that a draupnir client conforms to
type DraupnirClient interface {
	GetImage(id string) (models.Image, error)
	GetInstance(id string) (models.Instance, error)
	ListImages() ([]models.Image, error)
	ListInstances() ([]models.Instance, error)
	CreateInstance(image models.Image) (models.Instance, error)
	DestroyInstance(instance models.Instance) error
	DestroyImage(image models.Image) error
}

func (c Client) GetImage(id string) (models.Image, error) {
	var image models.Image
	resp, err := http.Get(c.URL + "/images/" + id)
	if err != nil {
		return image, err
	}

	if resp.StatusCode != http.StatusOK {
		return image, ErrorFromReader(resp.Body)
	}

	err = jsonapi.UnmarshalPayload(resp.Body, &image)
	return image, err
}

func (c Client) GetInstance(id string) (models.Instance, error) {
	var instance models.Instance
	resp, err := http.Get(c.URL + "/instances/" + id)
	if err != nil {
		return instance, err
	}

	if resp.StatusCode != http.StatusOK {
		return instance, ErrorFromReader(resp.Body)
	}

	err = jsonapi.UnmarshalPayload(resp.Body, &instance)
	return instance, err
}

// ListImages returns a list of all images
func (c Client) ListImages() ([]models.Image, error) {
	var images []models.Image
	resp, err := http.Get(c.URL + "/images")
	if err != nil {
		return images, err
	}

	maybeImages, err := jsonapi.UnmarshalManyPayload(resp.Body, reflect.TypeOf(images))
	if err != nil {
		return nil, err
	}

	// Convert from []interface{} to []Image
	images = make([]models.Image, 0)
	for _, image := range maybeImages {
		i := image.(*models.Image)
		images = append(images, *i)
	}

	return images, nil
}

// ListInstances returns a list of all instances
func (c Client) ListInstances() ([]models.Instance, error) {
	var instances []models.Instance
	resp, err := http.Get(c.URL + "/instances")
	if err != nil {
		return instances, err
	}

	maybeInstances, err := jsonapi.UnmarshalManyPayload(resp.Body, reflect.TypeOf(instances))
	if err != nil {
		return nil, err
	}

	// Convert from []interface{} to []Instance
	instances = make([]models.Instance, 0)
	for _, instance := range maybeInstances {
		i := instance.(*models.Instance)
		instances = append(instances, *i)
	}

	return instances, nil
}

type createInstanceRequest struct {
	ImageID string `jsonapi:"attr,image_id"`
}

func (c Client) CreateInstance(image models.Image) (models.Instance, error) {
	var instance models.Instance
	request := createInstanceRequest{ImageID: strconv.Itoa(image.ID)}

	var payload bytes.Buffer
	err := jsonapi.MarshalOnePayloadWithoutIncluded(&payload, &request)
	if err != nil {
		return instance, err
	}

	resp, err := http.Post(c.URL+"/instances", "application/json", &payload)
	if err != nil {
		return instance, err
	}

	// If we don't get a 201 back, return the response as an error
	if resp.StatusCode != http.StatusCreated {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return instance, errors.New(buf.String())
	}

	err = jsonapi.UnmarshalPayload(resp.Body, &instance)
	return instance, err
}

func (c Client) DestroyInstance(instance models.Instance) error {
	url := c.URL + "/instances/" + strconv.Itoa(instance.ID)
	req, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(""))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// If we don't get a 204 back, return the response as an error
	if resp.StatusCode != http.StatusNoContent {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return errors.New(buf.String())
	}

	return nil
}

func (c Client) DestroyImage(image models.Image) error {
	url := c.URL + "/images/" + strconv.Itoa(image.ID)
	req, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(""))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// If we don't get a 204 back, return the response as an error
	if resp.StatusCode != http.StatusNoContent {
		return ErrorFromReader(resp.Body)
	}

	return nil
}
