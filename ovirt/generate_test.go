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
		name         string
		g            GenerateVars
		template     string
		wantErr      error
		wantStorages []string
		wantWarns    []string
		wantVarFile  string
	}{
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []StorageMap{
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/nfs_dom_dr/",
							PrimaryAddr:   "10.1.1.2",
							SecondaryType: "nfs",
							SecondaryPath: "/nfs_dom_dr2/",
							SecondaryAddr: "10.1.2.2",
						},
					},
				},
			},
			template:    "disaster_recovery_vars.yml.tpl",
			wantVarFile: "disaster_recovery_vars.yml",
			wantStorages: []string{
				`storage nfs_dom remapped with name nfs_dom as nfs://10.1.2.2:/nfs_dom_dr2`,
			},
			wantWarns: []string{},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []StorageMap{
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/nfs_dom_dr/",
							PrimaryAddr:   "10.1.1.2",
							SecondaryType: "nfs",
							SecondaryPath: "/nfs_dom_dr2/",
							SecondaryAddr: "10.1.2.2",
						},
					},
				},
				Rewrite: map[string]string{
					"dr_lun_mappings":                   "~", // delete
					"dr_role_mappings[0].primary_name":  "PRIMARY",
					"dr_role_mappings[].secondary_name": "SECONDARY",
				},
			},
			template:    "disaster_recovery_vars.yml.tpl",
			wantVarFile: "disaster_recovery_vars_remap.yml",
			wantStorages: []string{
				`storage nfs_dom remapped with name nfs_dom as nfs://10.1.2.2:/nfs_dom_dr2`,
			},
			wantWarns: []string{},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@ovirt@internal",
				StorageDomains: []StorageMap{
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/nfs_tst/",
							PrimaryAddr:   "192.168.1.210",
							SecondaryType: "nfs",
							SecondaryPath: "/nfs_tst2/",
							SecondaryAddr: "192.168.2.210",
						},
					},
				},
			},
			template:    "disaster_recovery_vars2.yml.tpl",
			wantVarFile: "disaster_recovery_vars2.yml",
			wantStorages: []string{
				`storage nfstst remapped with name nfstst as nfs://192.168.2.210:/nfs_tst2`,
			},
			wantWarns: []string{},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []StorageMap{
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/non_exist/",
							PrimaryAddr:   "10.1.1.2",
							SecondaryType: "nfs",
							SecondaryPath: "/non_exist2/",
							SecondaryAddr: "10.1.2.2",
						},
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
				StorageDomains: []StorageMap{
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/nfs_tst/",
							PrimaryAddr:   "192.168.1.210",
							SecondaryType: "nfs",
							SecondaryPath: "/nfs_tst2/",
							SecondaryAddr: "192.168.2.210",
						},
					},
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/non_exist/",
							PrimaryAddr:   "10.1.1.2",
							SecondaryType: "nfs",
							SecondaryPath: "/non_exist2/",
							SecondaryAddr: "10.1.2.2",
						},
					},
				},
			},
			template:    "disaster_recovery_vars2.yml.tpl",
			wantVarFile: "disaster_recovery_vars2.yml",
			wantStorages: []string{
				`storage nfstst remapped with name nfstst as nfs://192.168.2.210:/nfs_tst2`,
			},
			wantWarns: []string{
				`storage map nfs://10.1.1.2:/non_exist not used`,
			},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []StorageMap{
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/nfs_dom_dr/",
							PrimaryAddr:   "10.1.1.2",
							SecondaryType: "nfs",
							SecondaryPath: "/nfs_dom_dr2/",
							SecondaryAddr: "10.1.2.2",
						},
					},
					{
						StorageBase: StorageBase{
							PrimaryType:   "fcp",
							PrimaryId:     "0abc45defc",
							SecondaryType: "fcp",
							SecondaryId:   "0abc45defc",
						},
					},
				},
			},
			template:    "disaster_recovery_vars.yml.tpl",
			wantVarFile: "disaster_recovery_vars_with_fcp.yml",
			wantStorages: []string{
				`storage fc_tst remapped with name fc_tst as fcp://0abc45defc`,
				`storage nfs_dom remapped with name nfs_dom as nfs://10.1.2.2:/nfs_dom_dr2`,
			},
			wantWarns: []string{},
		},
		{
			g: GenerateVars{
				SecondaryUrl:      "https://saengine2.localdomain/ovirt-engine/api",
				SecondaryUsername: "admin@internal",
				StorageDomains: []StorageMap{
					{
						StorageBase: StorageBase{
							PrimaryType:   "nfs",
							PrimaryPath:   "/nfs_dom_dr/",
							PrimaryAddr:   "10.1.1.2",
							SecondaryType: "nfs",
							SecondaryPath: "/nfs_dom_dr2/",
							SecondaryAddr: "10.1.2.2",
						},
					},
					{
						StorageBase: StorageBase{
							PrimaryType:   "iscsi",
							PrimaryId:     "bcca8438-810f-4932-bf25-d874babd97b1",
							PrimaryAddr:   "192.168.1.101",
							PrimaryPort:   "3260",
							SecondaryType: "iscsi",
							SecondaryId:   "bcca8438-810f-4932-bf25-d874babd97b1",
							SecondaryAddr: "192.168.2.101",
							SecondaryPort: "3260",
						},
						Targets: map[string]string{
							"iqn.2006-01.com.openfiler:olvm-data1": "iqn.2006-02.com.openfiler:olvm-data1-2",
							"iqn.2006-01.com.openfiler:olvm-data3": "iqn.2006-02.com.openfiler:olvm-data3-2",
						},
					},
					{
						StorageBase: StorageBase{
							PrimaryType:   "iscsi",
							PrimaryId:     "bcca8438-810f-4932-bf25-d874babd97b1",
							PrimaryAddr:   "192.168.1.101",
							PrimaryPort:   "3260",
							SecondaryType: "iscsi",
							SecondaryId:   "bcca8438-810f-4932-bf25-d874babd97b1",
							SecondaryAddr: "192.168.2.101",
							SecondaryPort: "3260",
						},
						Targets: map[string]string{
							"iqn.2006-01.com.openfiler:olvm-iso": "iqn.2006-02.com.openfiler:olvm-iso",
						},
					},
				},
			},
			template:    "disaster_recovery_vars_with_iscsi.yml.tpl",
			wantVarFile: "disaster_recovery_vars_with_iscsi.yml",
			wantStorages: []string{
				`storage data remapped with name data as iscsi://bcca8438-810f-4932-bf25-d874babd97b1:iqn.2006-02.com.openfiler:olvm-data1-2,iqn.2006-02.com.openfiler:olvm-data3-2`,
				`storage nfs_dom remapped with name nfs_dom as nfs://10.1.2.2:/nfs_dom_dr2`,
				`storage iso remapped with name iso as iscsi://bcca8438-810f-4932-bf25-d874babd97b1:iqn.2006-02.com.openfiler:olvm-iso`,
			},
			wantWarns: []string{},
		},
	} {
		t.Run(test.name+"#"+test.wantVarFile, func(t *testing.T) {
			g := test.g

			f, err := os.CreateTemp("", "xrm-controller")
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

			if storages, warns, err := g.writeAnsibleVarsFile(template, varFile); err != test.wantErr {
				t.Fatalf("GenerateVars.writeAnsibleVarsFile() error = %v, want = %v", err, test.wantErr)
			} else if err == nil {
				if !reflect.DeepEqual(storages, test.wantStorages) {
					t.Errorf("GenerateVars.writeAnsibleVarsFile() storages\n%#q\nwant\n%#q", storages, test.wantStorages)
				}

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
