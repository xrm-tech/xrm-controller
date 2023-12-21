package ovirt

import (
	"testing"
)

func TestViewAll(t *testing.T) {
	tests := []struct {
		dir     string
		wantOut string
		wantErr bool
	}{
		{
			dir:     "tests/store",
			wantOut: "test1\ntest2 (INCOMPLETE)\n",
		},
		{
			dir: "tests/NON_EXIST",
		},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			gotOut, err := ViewAll(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ViewAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOut != tt.wantOut {
				t.Errorf("ViewAll() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestView(t *testing.T) {
	tests := []struct {
		dir     string
		name    string
		wantOut string
		wantErr error
	}{
		{
			dir:  "tests/store",
			name: "test1",
			wantOut: `---
dr_sites_primary_url: https://saengine.localdomain/ovirt-engine/api
dr_sites_primary_username: admin@internal
`,
		},
		{
			dir:     "tests/store",
			name:    "test2",
			wantErr: ErrAnsibleDrVarsFile,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, err := View(tt.name, tt.dir)
			if err != tt.wantErr {
				t.Errorf("View() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOut != tt.wantOut {
				t.Errorf("View() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}
