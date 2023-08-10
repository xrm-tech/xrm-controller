package xrmcontroller

import (
	"reflect"
	"testing"

	"github.com/xrm-tech/xrm-controller/ovirt"
)

func Test_bodyPasswordCleanup(t *testing.T) {
	gotSitesConfig := ovirt.GenerateVars{
		PrimaryUrl:        "https://saengine.localdomain/ovirt-engine/api",
		PrimaryUsername:   "admin@internal",
		PrimaryPassword:   "_pwd_",
		SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
		SecondaryUsername: "admin@internal",
		SecondaryPassword: "_SECURE_",
		StorageDomains: []ovirt.StorageMap{
			{
				StorageBase: ovirt.StorageBase{
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
		},
	}
	wantSitesConfig := ovirt.GenerateVars{
		PrimaryUrl:        "https://saengine.localdomain/ovirt-engine/api",
		PrimaryUsername:   "admin@internal",
		PrimaryPassword:   "<STRIPPED>",
		SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
		SecondaryUsername: "admin@internal",
		SecondaryPassword: "<STRIPPED>",
		StorageDomains: []ovirt.StorageMap{
			{
				StorageBase: ovirt.StorageBase{
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
		},
	}

	body, err := Encode(gotSitesConfig)
	if err != nil {
		t.Fatal(err)
	}
	got := bodyPasswordCleanup(body)
	err = Decode(got, &gotSitesConfig)
	if err != nil {
		t.Fatalf("%v: %q", err, string(got))
	}
	if !reflect.DeepEqual(gotSitesConfig, wantSitesConfig) {
		t.Errorf("bodySecurityCleanup() got\n%#v\nwant\n%#v", gotSitesConfig, wantSitesConfig)
	}
}
