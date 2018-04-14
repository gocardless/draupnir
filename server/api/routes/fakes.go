package routes

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/common/log"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/server/api/chain"
)

func NewFakeLogger() (log.Logger, *bytes.Buffer) {
	var buffer bytes.Buffer
	return log.NewLogger(&buffer), &buffer
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
	_CreateBtrfsSubvolume func(ctx context.Context, id int) error
	_FinaliseImage        func(ctx context.Context, image models.Image) error
	_CreateInstance       func(ctx context.Context, imageID int, instanceID int, port int) error
	_DestroyImage         func(ctx context.Context, id int) error
	_DestroyInstance      func(ctx context.Context, id int) error
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

func (e FakeExecutor) DestroyImage(ctx context.Context, id int) error {
	return e._DestroyImage(ctx, id)
}

func (e FakeExecutor) DestroyInstance(ctx context.Context, id int) error {
	return e._DestroyInstance(ctx, id)
}

type FakeAuthenticator struct {
	_AuthenticateRequest func(r *http.Request) (string, error)
}

func (f FakeAuthenticator) AuthenticateRequest(r *http.Request) (string, error) {
	return f._AuthenticateRequest(r)
}

type FakeOAuthClient struct {
	_AuthCodeURL func(string, ...oauth2.AuthCodeOption) string
	_Exchange    func(context.Context, string) (*oauth2.Token, error)
}

func (c *FakeOAuthClient) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return c._AuthCodeURL(state, opts...)
}

func (c *FakeOAuthClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return c._Exchange(ctx, code)
}

func fakeOauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "the-client-id",
		ClientSecret: "the-client-secret",
		Scopes:       []string{"the-scope"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.org/auth",
			TokenURL: "https://example.org/token",
		},
		RedirectURL: "https://draupnir.org/redirect",
	}
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
	req = req.WithContext(context.WithValue(req.Context(), loggerKey, &logger))
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, "test@draupnir"))
	return req, recorder, output
}
