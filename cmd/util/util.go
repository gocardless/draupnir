package util

import (
	"log"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/willdonnelly/passwd"
)

const minPort = 1025
const maxPort = 49152

func init() {
	// seed math.rand with the current time
	// we don't need crypto-secure randomness,
	// just something vaguely random to pick a port for postgres
	rand.Seed(time.Now().Unix())
}

func RandomPort() int {
	return minPort + rand.Intn(maxPort-minPort)
}

func Execute(uid uint32, prog string, args ...string) ([]byte, error) {
	log.Printf("%s %s", prog, strings.Join(args, " "))
	cred := syscall.Credential{Uid: uid}
	attr := syscall.SysProcAttr{Credential: &cred}
	cmd := exec.Command(prog, args...)
	cmd.SysProcAttr = &attr
	return cmd.Output()
}

func GetUID(username string) (uint32, error) {
	return getUIDFromPasswd(username)
}

// disgusting bullshit parsing /etc/password -- no cgo
func getUIDFromPasswd(username string) (uint32, error) {
	users, err := passwd.Parse()
	uid := users[username].Uid

	uidInt, err := strconv.Atoi(uid)
	if err != nil {
		return 0, err
	}

	return uint32(uidInt), nil
}
