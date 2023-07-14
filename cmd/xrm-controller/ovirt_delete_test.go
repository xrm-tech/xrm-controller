package main

import (
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	xrm "github.com/xrm-tech/xrm-controller/app/xrm-controller"
	"github.com/xrm-tech/xrm-controller/pkg/tests"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

func TestDelete(t *testing.T) {
	var err error

	if xrm.Cfg.Listen, err = tests.GetFreeLocalAddr(); err != nil {
		t.Fatal(err)
	}
	if xrm.Cfg.OVirtStoreDir, err = os.MkdirTemp("", "xrm-controller"); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(xrm.Cfg.StoreDir)

	request := "http://" + xrm.Cfg.Listen + "/ovirt/delete/test"
	fileName := path.Join(xrm.Cfg.OVirtStoreDir, "test/disaster_recovery_vars.yml")

	// create and start *fiber.App instance
	xrm.Cfg.Logger = zerolog.New(os.Stdout)
	xrm.Cfg.Users = map[string]string{"test1": "password1", "test2": "password2"}
	app := xrm.RouterInit()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		_ = app.Listen(xrm.Cfg.Listen)
	}()
	wg.Wait()
	defer func() { _ = app.Shutdown() }()
	time.Sleep(time.Millisecond * 10)

	// run first test (no disaster_recovery_vars.yml), but must success
	req, _ := http.NewRequest("GET", request, nil)
	req.SetBasicAuth("test1", "password1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("/ovirt/cleanup/test error = %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || err != nil {
		t.Fatalf("/ovirt/cleanup/test = %d (%s), error is %v", resp.StatusCode, string(body), err)
	}
	if utils.FileExists(fileName) {
		t.Fatalf(fileName + " not cleaned")
	}

	// run test (disaster_recovery_vars.yml exist), must success
	dir := path.Dir(fileName)
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err = utils.TouchFile(fileName); err != nil {
		t.Fatal(err)
	}
	if !utils.FileExists(fileName) {
		t.Fatalf(fileName + " not created")
	}

	req.SetBasicAuth("test2", "password2")
	if resp, err = http.DefaultClient.Do(req); err != nil {
		t.Fatalf("/ovirt/cleanup/test error = %v", err)
	}
	body, err = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || err != nil {
		t.Fatalf("/ovirt/cleanup/test = %d (%s), error is %v", resp.StatusCode, string(body), err)
	}
	if utils.FileExists(fileName) {
		t.Fatalf(fileName + " not cleaned")
	}
}
