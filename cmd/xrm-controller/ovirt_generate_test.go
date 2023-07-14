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
)

func TestGenerateValidate(t *testing.T) {
	var err error

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

	if err := tests.DoGenerate(request, nil, "test1", "password2", http.StatusUnauthorized, ""); err != nil {
		t.Fatal(err)
	}

	// run without parameters
	if err := tests.DoGenerate(request, nil, "test1", "password1", http.StatusBadRequest, ""); err != nil {
		t.Fatal(err)
	}

	siteConfig := ovirt.GenerateVars{
		SecondaryUrl: "https://saengine2.localdomain/ovirt-engine/api",
		StorageDomains: []ovirt.Storage{
			{
				PrimaryType:   "nfs",
				PrimaryName:   "nfs_dom",
				PrimaryPath:   "/nfs_dom_dr/",
				PrimaryAddr:   "10.1.1.2",
				SecondaryType: "nfs",
				SecondaryName: "nfs_dom",
				SecondaryPath: "/nfs_dom_dr2/",
				SecondaryAddr: "10.1.2.3",
			},
		},
	}
	// run without parameters
	if err := tests.DoGenerate(request, &siteConfig, "test2", "password2", http.StatusBadRequest, "site_primary_url is empty\nsite_primary_username is empty\nsite_primary_password is empty\nsite_secondary_username is empty\nsite_secondary_password is empty\n"); err != nil {
		t.Fatal(err)
	}

	// siteConfig.Url = "https://127.0.0.1/ovirt-engine/api"
	// siteConfig.Username = "admin@internal"
	// siteConfig.Password = "123456"
	// siteConfig.Insecure = true
	// if err := doGenerate(request, &siteConfig, "test2", "password2", http.StatusOK, ""); err != nil {
	// 	t.Fatal(err)
	// }
}

func TestGenerateFiltering(t *testing.T) {
	var err error

	if xrm.Cfg.Listen, err = tests.GetFreeLocalAddr(); err != nil {
		t.Fatal(err)
	}
	if xrm.Cfg.OVirtStoreDir, err = os.MkdirTemp("", "xrm-controller"); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(xrm.Cfg.StoreDir)

	request := "http://" + xrm.Cfg.Listen + "/ovirt/generate/test%2F..%2F..%2F..%2Fetc"
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
		PrimaryUrl:        "https://saengine2.localdomain/ovirt-engine/api",
		PrimaryUsername:   "admin@ovirt@internal",
		PrimaryPassword:   "password",
		SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
		SecondaryUsername: "admin@ovirt@internal",
		SecondaryPassword: "password",
		StorageDomains: []ovirt.Storage{
			{
				PrimaryType:   "nfs",
				PrimaryName:   "nfs_dom",
				PrimaryPath:   "/nfs_dom_dr/",
				PrimaryAddr:   "10.1.1.2",
				SecondaryType: "nfs",
				SecondaryName: "nfs_dom",
				SecondaryPath: "/nfs_dom_dr2/",
				SecondaryAddr: "10.1.2.3",
			},
		},
	}
	// run without parameters
	if err := tests.DoGenerate(request, &siteConfig, "test2", "password2", http.StatusBadRequest, "name is invalid\n"); err == nil {
		t.Fatal("mnust fail")
	}
}
