package ovirt

import (
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/juju/fslock"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	drCleanTag    = "clean_engine"
	drFailoverTag = "fail_over"
	drFailbackTag = "fail_back"

	lock sync.Mutex
	wg   sync.WaitGroup
)

// Failover initiate failover for {dir}/{name}
func Failover(name, dir string) (out string, err error) {
	ansiblePath, err := exec.LookPath("ansible-playbook")
	if err != nil {
		err = ErrAnsibleNotFound
		return
	}

	if !validateName(name) {
		return "", ErrNameInvalid
	}

	dir = path.Join(dir, name)
	if !utils.DirExists(dir) {
		return "", ErrDirNotExist
	}

	if !lock.TryLock() {
		return "", ErrInProgress
	}

	flock := fslock.New(dir + ".lock")
	if err = flock.TryLock(); err != nil {
		lock.Unlock()
		return
	}

	playbook := path.Join(dir, ansibleFailoverPlaybook)

	wg.Add(1)

	go func() {
		defer func() {
			lock.Unlock()
			flock.Unlock()
			wg.Done()
		}()
		// TODO: reduce verbose ?
		out, err = utils.ExecCmd(dir+"/failover.log", time.Minute*10, ansiblePath, playbook, "-t", drFailoverTag, "-vvvvv")
	}()

	wg.Wait()

	return out, err
}

// Failback initiate failback for {dir}/{name}
func Failback(name, dir string) (out string, err error) {
	if !validateName(name) {
		return "", ErrNameInvalid
	}

	dir = path.Join(dir, name)
	if !utils.DirExists(dir) {
		return "", ErrDirNotExist
	}

	if !lock.TryLock() {
		return "", ErrInProgress
	}

	flock := fslock.New(dir + ".lock")
	if err = flock.TryLock(); err != nil {
		lock.Unlock()
		return
	}

	playbook := path.Join(dir, ansibleFailbackPlaybook)

	wg.Add(1)

	go func() {
		defer func() {
			lock.Unlock()
			flock.Unlock()
			wg.Done()
		}()
		// TODO: reduce verbose ?
		out, err = utils.ExecCmd(dir+"/failback.log", time.Minute*10, "ansible-playbook", playbook, "-t", drFailbackTag, "-vvvvv")
	}()

	wg.Wait()

	return out, err
}

// Cleanup cleanup for {dir}/{name}
func Cleanup(name, dir string) (out string, err error) {
	if !validateName(name) {
		return "", ErrNameInvalid
	}

	dir = path.Join(dir, name)
	if !utils.DirExists(dir) {
		return "", ErrDirNotExist
	}

	if !lock.TryLock() {
		return "", ErrInProgress
	}

	flock := fslock.New(dir + ".lock")
	if err = flock.TryLock(); err != nil {
		lock.Unlock()
		return
	}

	playbook := path.Join(dir, ansibleFailoverPlaybook)

	wg.Add(1)

	go func() {
		defer func() {
			lock.Unlock()
			flock.Unlock()
			wg.Done()
		}()
		// TODO: reduce verbose ?
		out, err = utils.ExecCmd(dir+"/cleanup.log", time.Minute*10, "ansible-playbook", playbook, "-t", drCleanTag, "-vvvvv")
	}()

	wg.Wait()

	return out, err
}
