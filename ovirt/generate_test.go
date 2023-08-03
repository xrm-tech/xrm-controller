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

func errStrs(errs []error) (out []string) {
	out = make([]string, len(errs))
	for i := 0; i < len(errs); i++ {
		out[i] = errs[i].Error()
	}
	return
}

func TestGenerateVars_writeAnsibleVarsFile(t *testing.T) {
	for _, test := range []struct {
		name        string
		g           GenerateVars
		template    string
		wantErr     error
		wantWarns   []string
		wantVarFile string
	}{
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []Storage{
					{
						PrimaryType:   "nfs",
						PrimaryPath:   "/nfs_dom_dr/",
						PrimaryAddr:   "10.1.1.2",
						SecondaryType: "nfs",
						SecondaryPath: "/nfs_dom_dr2/",
						SecondaryAddr: "10.1.2.2",
					},
				},
			},
			template:    "disaster_recovery_vars.yml.tpl",
			wantVarFile: "disaster_recovery_vars.yml",
			wantWarns: []string{
				`storage map for fc_tst not found`,
				`storage nfs_dom remapped with name nfs_dom as nfs://10.1.2.2:/nfs_dom_dr2`,
				`storage map for nfs_dom_2 not found`,
			},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []Storage{
					{
						PrimaryType:   "nfs",
						PrimaryPath:   "/nfs_dom_dr/",
						PrimaryAddr:   "10.1.1.2",
						SecondaryType: "nfs",
						SecondaryPath: "/nfs_dom_dr2/",
						SecondaryAddr: "10.1.2.2",
					},
					{
						PrimaryType:   "fcp",
						PrimaryPath:   "0abc45defc",
						SecondaryPath: "0abc45defc",
					},
				},
			},
			template:    "disaster_recovery_vars.yml.tpl",
			wantVarFile: "disaster_recovery_vars_with_fcp.yml",
			wantWarns: []string{
				`storage fc_tst remapped with name fc_tst as fcp://0abc45defc`,
				`storage nfs_dom remapped with name nfs_dom as nfs://10.1.2.2:/nfs_dom_dr2`,
				`storage map for nfs_dom_2 not found`,
			},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@ovirt@internal",
				StorageDomains: []Storage{
					{
						PrimaryType:   "nfs",
						PrimaryPath:   "/nfs_tst/",
						PrimaryAddr:   "192.168.1.210",
						SecondaryType: "nfs",
						SecondaryPath: "/nfs_tst2/",
						SecondaryAddr: "192.168.2.210",
					},
				},
			},
			template:    "disaster_recovery_vars2.yml.tpl",
			wantVarFile: "disaster_recovery_vars2.yml",
			wantWarns: []string{
				`storage map for nfs_d not found`,
				`storage nfstst remapped with name nfstst as nfs://192.168.2.210:/nfs_tst2`,
			},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []Storage{
					{
						PrimaryType:   "nfs",
						PrimaryPath:   "/non_exist/",
						PrimaryAddr:   "10.1.1.2",
						SecondaryType: "nfs",
						SecondaryPath: "/non_exist2/",
						SecondaryAddr: "10.1.2.2",
					},
				},
			},
			template: "disaster_recovery_vars.yml.tpl",
			wantErr:  ErrStorageRemapEmptyResult, // no storages in config remapped
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@ovirt@internal",
				StorageDomains: []Storage{
					{
						PrimaryType:   "nfs",
						PrimaryPath:   "/nfs_tst/",
						PrimaryAddr:   "192.168.1.210",
						SecondaryType: "nfs",
						SecondaryPath: "/nfs_tst2/",
						SecondaryAddr: "192.168.2.210",
					},
					{
						PrimaryType:   "nfs",
						PrimaryPath:   "/non_exist/",
						PrimaryAddr:   "10.1.1.2",
						SecondaryType: "nfs",
						SecondaryPath: "/non_exist2/",
						SecondaryAddr: "10.1.2.2",
					},
				},
			},
			template:    "disaster_recovery_vars2.yml.tpl",
			wantVarFile: "disaster_recovery_vars2.yml",
			wantWarns: []string{
				`storage map for nfs_d not found`,
				`storage nfstst remapped with name nfstst as nfs://192.168.2.210:/nfs_tst2`,
				`storage map nfs://10.1.1.2:/non_exist not used`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := test.g

			f, err := os.CreateTemp("", "")
			if err != nil {
				t.Fatal(err)
			}
			varFile := f.Name()
			os.Remove(varFile)

			_, filename, _, _ := runtime.Caller(0)
			testDir := path.Join(path.Dir(filename), "tests")
			template := path.Join(testDir, test.template)
			wantVarFile := path.Join(testDir, test.wantVarFile)

			defer os.Remove(varFile)

			if _, warns, err := g.writeAnsibleVarsFile(template, varFile); err != test.wantErr {
				t.Fatalf("GenerateVars.writeAnsibleVarsFile() error = %v, want = %v", err, test.wantErr)
			} else if err == nil {
				warnsStr := errStrs(warns)
				if !reflect.DeepEqual(warnsStr, test.wantWarns) {
					t.Errorf("GenerateVars.writeAnsibleVarsFile() warnings\n%#q\nwant\n%#q", warnsStr, test.wantWarns)
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
		})
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
