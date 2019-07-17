package exec

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gocardless/draupnir/pkg/models"
	"github.com/gocardless/draupnir/pkg/server/api/middleware"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
)

type Executor interface {
	CreateBtrfsSubvolume(ctx context.Context, id int) error
	FinaliseImage(ctx context.Context, image models.Image) error
	CreateInstance(ctx context.Context, imageID int, instanceID int, port int) error
	RetrieveInstanceCredentials(ctx context.Context, id int) (map[string][]byte, error)
	DestroyImage(ctx context.Context, id int) error
	DestroyInstance(ctx context.Context, id int) error
}

type OSExecutor struct {
	DataPath string
}

func GetLogger(ctx context.Context) log.Logger {
	logger, ok := ctx.Value(middleware.LoggerKey).(*log.Logger)
	if !ok {
		// Only a programming bug should cause this scenario, so exit the program
		// if it occurs.
		log.Fatal("Unable to retrieve logger from context")
	}
	return *logger
}

func runCommandAndLog(logger log.Logger, message string, command *exec.Cmd) error {
	// Execute our command, which gives us stdout and an exit error
	outputBytes, err := command.Output()
	// Always log stdout
	logger = logger.With("stdout", string(outputBytes))

	if err != nil {
		// Even though the error string is passed back up the stack and printed
		// there, output it here anyway so that we have all the relevant fields in
		// a single log entry
		logger = logger.With("error", err.Error())

		// If we can get stderr, by casting to an exit error, then log that too
		if ee, ok := err.(*exec.ExitError); ok {
			logger = logger.With("stderr", string(ee.Stderr))
		}
	}
	logger.Info(message)

	return err
}

// CreateBtrfsSubvolume creates a BTRFS subvolume in $(DataPath)/image_uploads
// and sets its permissions to 775 so that 'upload' can write to it.
func (e OSExecutor) CreateBtrfsSubvolume(ctx context.Context, id int) error {
	name := fmt.Sprintf("%d", id)
	path := filepath.Join(e.DataPath, "image_uploads", name)
	logger := GetLogger(ctx).With("imageID", id).With("path", path)

	cmd := exec.CommandContext(ctx, "btrfs", "subvolume", "create", path)
	err := runCommandAndLog(logger, "Created btrfs subvolume", cmd)
	if err != nil {
		return err
	}

	perms := os.ModeDir | 0775
	err = os.Chmod(path, perms)
	if err != nil {
		return err
	}

	logger.Info("Set subvolume permissions")

	return nil
}

// FinaliseImage runs draupnir-finalise_image against the image
// This does the following things:
// - Gives ownership of the image directory to postgres
// - Sets the permissions to 700 so postgres will start
// - Removes postmaster.* files
// - Starts postgres
// - Runs anonymisation function
// - Stops postgres
// - Creates a snapshot of the image directory
// This snapshot is the finalised image
//
// draupnir-finalise-image is a separate script because it has to run with sudo.
func (e OSExecutor) FinaliseImage(ctx context.Context, image models.Image) error {
	anonFile, err := ioutil.TempFile("/tmp", "draupnir")
	if err != nil {
		return err
	}

	_, err = io.WriteString(anonFile, image.Anon)
	if err != nil {
		return err
	}

	err = anonFile.Sync()
	if err != nil {
		return err
	}

	logger := GetLogger(ctx).With("imageID", image.ID)

	cmd := exec.CommandContext(
		ctx,
		"sudo",
		"draupnir-finalise-image",
		e.DataPath,
		fmt.Sprintf("%d", image.ID),
		fmt.Sprintf("%d", 5432+image.ID),
		anonFile.Name(),
	)

	err = runCommandAndLog(logger, "Finalised image", cmd)
	if err != nil {
		return err
	}

	logger.With("file", anonFile.Name()).Info("Removing anonymisation file")
	return os.Remove(anonFile.Name())
}

func (e OSExecutor) CreateInstance(ctx context.Context, imageID int, instanceID int, port int) error {
	logger := GetLogger(ctx).With("imageID", imageID).With("instanceID", instanceID).With("port", port)

	cmd := exec.Command(
		"sudo",
		"draupnir-create-instance",
		e.DataPath,
		fmt.Sprintf("%d", imageID),
		fmt.Sprintf("%d", instanceID),
		fmt.Sprintf("%d", port),
	)

	return runCommandAndLog(logger, "Creating instance", cmd)
}

// RetrieveInstanceCredentials reads the certificate and key files from the
// instance directory and returns them in a map
func (e OSExecutor) RetrieveInstanceCredentials(ctx context.Context, id int) (map[string][]byte, error) {
	logger := GetLogger(ctx).With("imageID", id)

	basePath := filepath.Join(e.DataPath, "instances", fmt.Sprintf("%d", id))

	files := []string{"client.key", "client.crt", "ca.crt"}
	fileContents := make(map[string][]byte)

	for _, fileName := range files {
		bytes, err := ioutil.ReadFile(filepath.Join(basePath, fileName))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read credentials file %s", fileName)
		}

		fileContents[fileName] = bytes
	}

	logger.Info("Successfully retrieved instance credentials")
	return fileContents, nil
}

func (e OSExecutor) DestroyImage(ctx context.Context, id int) error {
	logger := GetLogger(ctx).With("imageID", id)

	cmd := exec.Command(
		"sudo",
		"draupnir-destroy-image",
		e.DataPath,
		fmt.Sprintf("%d", id),
	)

	return runCommandAndLog(logger, "Destroyed image", cmd)
}

func (e OSExecutor) DestroyInstance(ctx context.Context, id int) error {
	logger := GetLogger(ctx).With("instanceID", id)

	cmd := exec.CommandContext(
		ctx,
		"sudo",
		"draupnir-destroy-instance",
		e.DataPath,
		fmt.Sprintf("%d", id),
	)

	return runCommandAndLog(logger, "Destroyed instance", cmd)
}
