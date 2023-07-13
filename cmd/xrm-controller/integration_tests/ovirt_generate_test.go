//go:build test_all || test_integration
// +build test_all test_integration

package main

import (
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	xrm "github.com/xrm-tech/xrm-controller/app/xrm-controller"
	"github.com/xrm-tech/xrm-controller/ovirt"
	"github.com/xrm-tech/xrm-controller/pkg/tests"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

func TestGenerate(t *testing.T) {
	var (
		err error
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

	siteConfig := ovirt.GenerateVars{
		PrimaryUrl:        os.Getenv("XRM_PRIMARY_URL"),
		PrimaryUsername:   os.Getenv("XRM_PRIMARY_USERNAME"),
		PrimaryPassword:   os.Getenv("XRM_PRIMARY_PASSWORD"),
		SecondaryUrl:      os.Getenv("XRM_SECONDARY_URL"),
		SecondaryUsername: os.Getenv("XRM_SECONDARY_USERNAME"),
		SecondaryPassword: os.Getenv("XRM_SECONDARY_PASSWORD"),
	}
	if err := siteConfig.Validate(); err != nil {
		t.Fatal(err)
	}

	if err := tests.DoGenerate(request, &siteConfig, "test2", "password2", http.StatusOK, ""); err != nil {
		t.Fatal(err)
	}
}
