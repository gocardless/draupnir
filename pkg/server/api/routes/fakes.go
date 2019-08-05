package routes

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/common/log"
	"golang.org/x/net/context"

	"github.com/gocardless/draupnir/pkg/models"
	"github.com/gocardless/draupnir/pkg/server/api/chain"
	"github.com/gocardless/draupnir/pkg/server/api/middleware"
)

func NewFakeLogger() (log.Logger, *bytes.Buffer) {
	var buffer bytes.Buffer
	writer := io.MultiWriter(&buffer, os.Stdout)
	return log.NewLogger(writer), &buffer
}

type FakeImageStore struct {
	_List        func() ([]models.Image, error)
	_Get         func(int) (models.Image, error)
	_Create      func(models.Image) (models.Image, error)
	_Destroy     func(models.Image) error
	_MarkAsReady func(models.Image) (models.Image, error)
}

func (s FakeImageStore) List() ([]models.Image, error) {
	return s._List()
}

func (s FakeImageStore) Get(id int) (models.Image, error) {
	return s._Get(id)
}

func (s FakeImageStore) Create(image models.Image) (models.Image, error) {
	return s._Create(image)
}

func (s FakeImageStore) Destroy(image models.Image) error {
	return s._Destroy(image)
}

func (s FakeImageStore) MarkAsReady(image models.Image) (models.Image, error) {
	return s._MarkAsReady(image)
}

type FakeInstanceStore struct {
	_Create  func(models.Instance) (models.Instance, error)
	_List    func() ([]models.Instance, error)
	_Get     func(int) (models.Instance, error)
	_Destroy func(instance models.Instance) error
}

func (s FakeInstanceStore) Create(image models.Instance) (models.Instance, error) {
	return s._Create(image)
}

func (s FakeInstanceStore) List() ([]models.Instance, error) {
	return s._List()
}

func (s FakeInstanceStore) Get(id int) (models.Instance, error) {
	return s._Get(id)
}

func (s FakeInstanceStore) Destroy(instance models.Instance) error {
	return s._Destroy(instance)
}

type FakeExecutor struct {
	_CreateBtrfsSubvolume        func(ctx context.Context, id int) error
	_FinaliseImage               func(ctx context.Context, image models.Image) error
	_CreateInstance              func(ctx context.Context, imageID int, instanceID int, port int) error
	_RetrieveInstanceCredentials func(ctx context.Context, id int) (map[string][]byte, error)
	_DestroyImage                func(ctx context.Context, id int) error
	_DestroyInstance             func(ctx context.Context, id int) error
}

func (e FakeExecutor) CreateBtrfsSubvolume(ctx context.Context, id int) error {
	return e._CreateBtrfsSubvolume(ctx, id)
}

func (e FakeExecutor) FinaliseImage(ctx context.Context, image models.Image) error {
	return e._FinaliseImage(ctx, image)
}

func (e FakeExecutor) CreateInstance(ctx context.Context, imageID int, instanceID int, port int) error {
	return e._CreateInstance(ctx, imageID, instanceID, port)
}

func (e FakeExecutor) RetrieveInstanceCredentials(ctx context.Context, id int) (map[string][]byte, error) {
	return e._RetrieveInstanceCredentials(ctx, id)
}

func (e FakeExecutor) DestroyImage(ctx context.Context, id int) error {
	return e._DestroyImage(ctx, id)
}

func (e FakeExecutor) DestroyInstance(ctx context.Context, id int) error {
	return e._DestroyInstance(ctx, id)
}

type FakeErrorHandler struct {
	Error error
}

func (f *FakeErrorHandler) Handle(h chain.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		f.Error = err
	}
}

// This function is used in tests to construct an HTTP request, response
// recorder and a fake logger
func createRequest(t *testing.T, method string, path string, body io.Reader) (*http.Request, *httptest.ResponseRecorder, *bytes.Buffer) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal(err)
	}

	logger, output := NewFakeLogger()
	req = req.WithContext(context.WithValue(req.Context(), middleware.LoggerKey, &logger))
	req = req.WithContext(context.WithValue(req.Context(), middleware.AuthUserKey, "test@draupnir"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.RefreshTokenKey, "refresh-token"))

	return req, recorder, output
}
