package routes

import (
	"net/http"

	"github.com/gocardless/draupnir/models"
)

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
	_CreateBtrfsSubvolume func(id int) error
	_FinaliseImage        func(image models.Image) error
	_CreateInstance       func(imageID int, instanceID int, port int) error
	_DestroyImage         func(id int) error
	_DestroyInstance      func(id int) error
}

func (e FakeExecutor) CreateBtrfsSubvolume(id int) error {
	return e._CreateBtrfsSubvolume(id)
}

func (e FakeExecutor) FinaliseImage(image models.Image) error {
	return e._FinaliseImage(image)
}

func (e FakeExecutor) CreateInstance(imageID int, instanceID int, port int) error {
	return e._CreateInstance(imageID, instanceID, port)
}

func (e FakeExecutor) DestroyImage(id int) error {
	return e._DestroyImage(id)
}

func (e FakeExecutor) DestroyInstance(id int) error {
	return e._DestroyInstance(id)
}

type FakeAuthenticator struct {
	_AuthenticateRequest func(r *http.Request) (string, error)
}

func (f FakeAuthenticator) AuthenticateRequest(r *http.Request) (string, error) {
	return f._AuthenticateRequest(r)
}

type AllowAll struct{}

func (a AllowAll) AuthenticateRequest(r *http.Request) (string, error) {
	return "hmac@gocardless.com", nil
}