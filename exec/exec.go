package exec

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type Executor interface {
	CreateBtrfsSubvolume(id int) error
	FinaliseImage(id int) error
	CreateInstance(imageID int, instanceID int, port int) error
	DestroyImage(id int) error
	DestroyInstance(id int) error
}

type OSExecutor struct{}

// CreateBtrfsSubvolume creates a BTRFS subvolume in /var/btrfs/image_uploads
// and sets its permissions to 775 so that 'upload' can write to it.
func (e OSExecutor) CreateBtrfsSubvolume(id int) error {
	name := fmt.Sprintf("%d", id)
	path := filepath.Join("/var/btrfs/image_uploads", name)
	output, err := exec.Command("btrfs", "subvolume", "create", path).Output()
	if err != nil {
		return err
	}
	log.Printf("Created btrfs subvolume %s: %s", name, output)

	perms := os.ModeDir | 0775
	err = os.Chmod(path, perms)
	if err != nil {
		return err
	}
	log.Printf("Set permissions for %s to %s", path, perms)

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
func (e OSExecutor) FinaliseImage(id int) error {
	output, err := exec.Command(
		"sudo",
		"draupnir-finalise-image",
		fmt.Sprintf("%d", id),
		fmt.Sprintf("%d", 5432+id),
	).Output()

	log.Printf("%s", output)
	if err != nil {
		return err
	}

	log.Printf("Finalised image %d", id)
	return nil
}

func (e OSExecutor) CreateInstance(imageID int, instanceID int, port int) error {
	output, err := exec.Command(
		"sudo",
		"draupnir-create-instance",
		fmt.Sprintf("%d", imageID),
		fmt.Sprintf("%d", instanceID),
		fmt.Sprintf("%d", port),
	).Output()

	log.Printf("%s", output)
	if err != nil {
		return err
	}

	log.Printf("Created instance %d", instanceID)
	return nil
}

func (e OSExecutor) DestroyImage(id int) error {
	output, err := exec.Command(
		"sudo",
		"draupnir-destroy-image",
		fmt.Sprintf("%d", id),
	).Output()

	log.Printf("%s", output)
	if err != nil {
		return err
	}

	log.Printf("Destroyed image %d", id)
	return nil
}

func (e OSExecutor) DestroyInstance(id int) error {
	output, err := exec.Command(
		"sudo",
		"draupnir-destroy-instance",
		fmt.Sprintf("%d", id),
	).Output()

	log.Printf("%s", output)
	if err != nil {
		return err
	}

	log.Printf("Destroyed instance %d", id)
	return nil
}
