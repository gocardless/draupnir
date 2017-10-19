package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/routes"
	"github.com/gocardless/draupnir/version"
	"github.com/google/jsonapi"
)

// Client represents the client for a draupnir server
type Client struct {
	// The URL of the draupnir server
	// e.g. "https://draupnir-server.my-infra.com"
	URL string
	// OAuth Access Token
	Token oauth2.Token
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
	CreateAccessToken(string) (string, error)
}

func (c Client) AuthorizationHeader() string {
	return fmt.Sprintf("Bearer %s", c.Token.RefreshToken)
}

func (c Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.AuthorizationHeader())
	req.Header.Set("Draupnir-Version", version.Version)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return response, err
	}

	versionHeader := response.Header["Draupnir-Version"]
	var apiVersion string
	if len(versionHeader) == 0 {
		apiVersion = "0.0.0"
	} else {
		apiVersion = versionHeader[0]
	}
	if apiVersion != version.Version {
		return response, fmt.Errorf("the API version (%s) does not match your client's version (%s). You may need to update your client", apiVersion, version.Version)
	}

	return response, nil
}

func (c Client) post(url string, payload *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.AuthorizationHeader())
	req.Header.Set("Draupnir-Version", version.Version)

	return http.DefaultClient.Do(req)
}

func (c Client) delete(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.AuthorizationHeader())
	req.Header.Set("Draupnir-Version", version.Version)

	return http.DefaultClient.Do(req)
}

func (c Client) GetLatestImage() (models.Image, error) {
	var image models.Image
	images, err := c.ListImages()

	if err != nil {
		fmt.Printf("error: %s\n", err)
		return image, err
	}

	// Make sure they are all sorted by UpdatedAt
	// The most up to date should be the first of the slice
	sort.Slice(images, func(i, j int) bool {
		return images[i].UpdatedAt.After(images[j].UpdatedAt)
	})

	for _, image := range images {
		if image.Ready {
			return image, nil
		}
	}

	return image, errors.New("no images available")
}

func (c Client) GetImage(id string) (models.Image, error) {
	var image models.Image
	resp, err := c.get(c.URL + "/images/" + id)
	if err != nil {
		return image, err
	}

	if resp.StatusCode != http.StatusOK {
		return image, parseError(resp.Body)
	}

	err = jsonapi.UnmarshalPayload(resp.Body, &image)
	return image, err
}

func (c Client) GetInstance(id string) (models.Instance, error) {
	var instance models.Instance
	resp, err := c.get(c.URL + "/instances/" + id)
	if err != nil {
		return instance, err
	}

	if resp.StatusCode != http.StatusOK {
		return instance, parseError(resp.Body)
	}

	err = jsonapi.UnmarshalPayload(resp.Body, &instance)
	return instance, err
}

// ListImages returns a list of all images
func (c Client) ListImages() ([]models.Image, error) {
	var images []models.Image
	resp, err := c.get(c.URL + "/images")
	if err != nil {
		return images, err
	}

	if resp.StatusCode != http.StatusOK {
		return images, parseError(resp.Body)
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
	resp, err := c.get(c.URL + "/instances")
	if err != nil {
		return instances, err
	}

	if resp.StatusCode != http.StatusOK {
		return instances, parseError(resp.Body)
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

// CreateInstance creates a new instance
func (c Client) CreateInstance(image models.Image) (models.Instance, error) {
	var instance models.Instance
	request := createInstanceRequest{ImageID: strconv.Itoa(image.ID)}

	var payload bytes.Buffer
	err := jsonapi.MarshalOnePayloadWithoutIncluded(&payload, &request)
	if err != nil {
		return instance, err
	}

	resp, err := c.post(c.URL+"/instances", &payload)
	if err != nil {
		return instance, err
	}

	if resp.StatusCode != http.StatusCreated {
		return instance, parseError(resp.Body)
	}

	err = jsonapi.UnmarshalPayload(resp.Body, &instance)
	return instance, err
}

// DestroyInstance destroys an instance
func (c Client) DestroyInstance(instance models.Instance) error {
	url := c.URL + "/instances/" + strconv.Itoa(instance.ID)
	resp, err := c.delete(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp.Body)
	}

	return nil
}

// DestroyImage destroys an image
func (c Client) DestroyImage(image models.Image) error {
	url := c.URL + "/images/" + strconv.Itoa(image.ID)
	resp, err := c.delete(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp.Body)
	}

	return nil
}

type createAccessTokenRequest struct {
	State string `jsonapi:"attr,state"`
}

// CreateAccessToken creates an oauth access token
func (c Client) CreateAccessToken(state string) (oauth2.Token, error) {
	var token oauth2.Token
	url := c.URL + "/access_tokens"

	request := createAccessTokenRequest{State: state}

	var payload bytes.Buffer
	err := jsonapi.MarshalOnePayloadWithoutIncluded(&payload, &request)
	if err != nil {
		return token, err
	}

	resp, err := c.post(url, &payload)
	if err != nil {
		return token, err
	}

	if resp.StatusCode != http.StatusCreated {
		return token, parseError(resp.Body)
	}

	err = json.NewDecoder(resp.Body).Decode(&token)
	return token, err
}

// parseError takes an io.Reader containing an API error response
// and converts it to an error
func parseError(r io.Reader) error {
	var apiError routes.APIError
	err := json.NewDecoder(r).Decode(&apiError)
	if err != nil {
		return err
	}
	return fmt.Errorf("%s (%s)", apiError.Title, apiError.Detail)
}
