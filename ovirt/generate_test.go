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
