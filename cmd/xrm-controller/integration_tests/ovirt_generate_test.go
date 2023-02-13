//go:build test_all || test_integration
// +build test_all test_integration

package main

import (
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	xrm "github.com/xrm-tech/xrm-controller/app/xrm-controller"
	"github.com/xrm-tech/xrm-controller/ovirt"
	"github.com/xrm-tech/xrm-controller/pkg/tests"
)

func TestGenerate(t *testing.T) {
	var (
		err    error
		logger = zerolog.New(os.Stdout)
	)
	if xrm.Cfg.Listen, err = tests.GetFreeLocalAddr(); err != nil {
		t.Fatal(err)
	}
	if xrm.Cfg.OVirtStoreDir, err = os.MkdirTemp("", "xrm-controller"); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(xrm.Cfg.StoreDir)

	request := "http://" + xrm.Cfg.Listen + "/ovirt/generate/test"
	// fileName := path.Join(ovirtStoreDir, "test/disaster_recovery_vars.yml")

	// create and start *fiber.App instance
	xrm.Cfg.Users = map[string]string{"test1": "password1", "test2": "password2"}
	app := xrm.RouterInit(&logger)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		_ = app.Listen(xrm.Cfg.Listen)
	}()
	wg.Wait()
	defer func() { _ = app.Shutdown() }()
	time.Sleep(time.Millisecond * 10)

	var siteConfig ovirt.Site
	siteConfig.Url = os.Getenv("XRM_CONTROLLER_ITEST_OVIRT_URL")
	siteConfig.Username = os.Getenv("XRM_CONTROLLER_ITEST_OVIRT_USERNAME")
	siteConfig.Password = os.Getenv("XRM_CONTROLLER_ITEST_OVIRT_PASSWORD")
	siteConfig.Ca = os.Getenv("XRM_CONTROLLER_ITEST_OVIRT_CA")
	if os.Getenv("XRM_CONTROLLER_ITEST_OVIRT_INSECURE") == "1" {
		siteConfig.Insecure = true
	}
	if err := xrm.Validate.Struct(&siteConfig); err != nil {
		t.Fatal(xrm.ValidatorError(err.(validator.ValidationErrors)))
	}

	siteConfig.Insecure = true
	if err := tests.DoGenerate(request, &siteConfig, "test2", "password2", http.StatusOK, ""); err != nil {
		t.Fatal(err)
	}
}
