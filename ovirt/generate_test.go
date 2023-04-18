package ovirt

import (
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateVars_writeAnsibleVarsFile(t *testing.T) {
	g := GenerateVars{
		SecondaryUrl: "https://saengine2.localdomain/ovirt-engine/api",
		StorageDomains: []Storage{
			{
				StorageType:   "nfs",
				PrimaryName:   "nfs_dom",
				PrimaryPath:   "/nfs_dom_dr/",
				PrimaryAddr:   "10.1.1.2",
				SecondaryName: "nfs_dom",
				SecondaryPath: "/nfs_dom_dr2/",
				SecondaryAddr: "10.1.2.3",
			},
		},
	}
	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	varFile := f.Name()
	os.Remove(varFile)

	_, filename, _, _ := runtime.Caller(0)
	testDir := path.Join(path.Dir(filename), "tests")
	template := path.Join(testDir, "disaster_recovery_vars.yml.tpl")
	wantVarFile := path.Join(testDir, "disaster_recovery_vars.yml")

	defer os.Remove(varFile)

	if err := g.writeAnsibleVarsFile(template, varFile); err != nil {
		t.Fatalf("GenerateVars.writeAnsibleVarsFile() error = %v", err)
	}

	b, err := os.ReadFile(varFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(string(b), "\n")
	b, err = os.ReadFile(wantVarFile)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(string(b), "\n")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GenerateVars.writeAnsibleVarsFile() = %s", cmp.Diff(want, got))
	}
}

func TestGenerateVars_writeAnsibleFailbackFile(t *testing.T) {
	g := GenerateVars{
		SecondaryUrl: "https://saengine2.localdomain/ovirt-engine/api",
	}
	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	failbackFile := f.Name()
	os.Remove(failbackFile)

	_, filename, _, _ := runtime.Caller(0)
	testDir := path.Join(path.Dir(filename), "tests")
	failover := path.Join(testDir, "dr_failover.yml")
	failback := path.Join(testDir, "dr_failback.yml")

	defer os.Remove(failbackFile)

	if err := g.writeAnsibleFailbackFile(failover, failbackFile); err != nil {
		t.Fatalf("GenerateVars.writeAnsibleFailbackFile() error = %v", err)
	}

	b, err := os.ReadFile(failbackFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(string(b), "\n")
	b, err = os.ReadFile(failback)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(string(b), "\n")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GenerateVars.writeAnsibleFailbackFile() = %s", cmp.Diff(want, got))
	}
}
