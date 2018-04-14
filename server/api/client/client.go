package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/gocardless/draupnir/models"
	apiErrors "github.com/gocardless/draupnir/server/api/errors"
	"github.com/gocardless/draupnir/server/api/routes"
	"github.com/gocardless/draupnir/version"
	"github.com/google/jsonapi"
)

// Client represents the client for a draupnir server
type Client struct {
	// The URL of the draupnir server
	// e.g. "https://draupnir-server.my-infra.com"
	url string
	// OAuth Access Token
	token  oauth2.Token
	client *http.Client
}

// NewClient constructs a new draupnir client, pointing at the given endpoint
func NewClient(url string, token oauth2.Token, insecure bool) Client {
	client := &http.Client{}

	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return Client{url, token, client}
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
	resp, err := c.get("/images/" + id)
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
	resp, err := c.get("/instances/" + id)
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
	resp, err := c.get("/images")
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
	resp, err := c.get("/instances")
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

// CreateInstance creates a new instance
func (c Client) CreateInstance(image models.Image) (models.Instance, error) {
	var instance models.Instance
	request := routes.CreateInstanceRequest{ImageID: strconv.Itoa(image.ID)}

	var payload bytes.Buffer
	err := jsonapi.MarshalOnePayloadWithoutIncluded(&payload, &request)
	if err != nil {
		return instance, err
	}

	resp, err := c.post("/instances", &payload)
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
	url := fmt.Sprintf("/instances/%d", instance.ID)
	resp, err := c.delete(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp.Body)
	}

	return nil
}

// CreateImage creates a new image. This does not complete the process of preparing an
// image, subsequent upload and finalisation steps are required.
func (c Client) CreateImage(backedUpAt time.Time, anon []byte) (models.Image, error) {
	var image models.Image
	request := routes.CreateImageRequest{BackedUpAt: backedUpAt, Anon: string(anon)}

	var payload bytes.Buffer
	err := jsonapi.MarshalOnePayloadWithoutIncluded(&payload, &request)
	if err != nil {
		return image, err
	}

	resp, err := c.post("/images", &payload)
	if err != nil {
		return image, err
	}

	if resp.StatusCode != http.StatusCreated {
		return image, parseError(resp.Body)
	}

	err = jsonapi.UnmarshalPayload(resp.Body, &image)
	return image, err
}

// FinaliseImage posts to images/id/done, causing draupnir to run the finalisation process
// to anonymise and prepare the image for usage.
func (c Client) FinaliseImage(imageID int) (models.Image, error) {
	var image models.Image
	var emptyPayload bytes.Buffer

	resp, err := c.post(fmt.Sprintf("/images/%d/done", imageID), &emptyPayload)
	if err != nil {
		err = jsonapi.UnmarshalPayload(resp.Body, &image)
	}

	return image, err
}

// DestroyImage destroys an image
func (c Client) DestroyImage(image models.Image) error {
	url := fmt.Sprintf("/images/%d", image.ID)
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
	request := createAccessTokenRequest{State: state}

	var payload bytes.Buffer
	err := jsonapi.MarshalOnePayloadWithoutIncluded(&payload, &request)
	if err != nil {
		return token, err
	}

	resp, err := c.post("/access_tokens", &payload)
	if err != nil {
		return token, err
	}

	if resp.StatusCode != http.StatusCreated {
		return token, parseError(resp.Body)
	}

	err = json.NewDecoder(resp.Body).Decode(&token)
	return token, err
}

func (c Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authorizationHeader())
	req.Header.Set("Draupnir-Version", version.Version)

	return c.client.Do(req)
}

func (c Client) get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.url+path, strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c Client) post(path string, payload *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.url+path, payload)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c Client) delete(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, c.url+path, strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c Client) authorizationHeader() string {
	return fmt.Sprintf("Bearer %s", c.token.RefreshToken)
}

// parseError takes an io.Reader containing an API error response
// and converts it to an error
func parseError(r io.Reader) error {
	var apiError apiErrors.Error
	err := json.NewDecoder(r).Decode(&apiError)
	if err != nil {
		return err
	}
	return fmt.Errorf("%s (%s)", apiError.Title, apiError.Detail)
}
