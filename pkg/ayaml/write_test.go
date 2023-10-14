package ayaml

import (
	"bufio"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		inPath  string
		outPath string
	}{
		{inPath: "dict.yml.tpl", outPath: "dict.yml"},
		{inPath: "dict_empty.yml.tpl", outPath: "dict_empty.yml"},
		{inPath: "list.yml.tpl", outPath: "list.yml"},
		{inPath: "disaster_recovery_vars.yml.tpl", outPath: "disaster_recovery_vars.yml"},
	}

	_, filename, _, _ := runtime.Caller(0)
	testDir := path.Join(path.Dir(filename), "tests")

	for _, tt := range tests {
		t.Run(tt.inPath, func(t *testing.T) {
			inPath := path.Join(testDir, tt.inPath)
			f, err := os.Open(inPath)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			nodes, err := Decode(f)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}

			of, err := os.CreateTemp("", "xrm-controller-"+tt.outPath)
			if err != nil {
				t.Error(err)
				return
			}
			defer func() {
				of.Close()
				os.Remove(of.Name())
			}()

			w := bufio.NewWriter(of)
			if err = nodes.Write(w); err != nil {
				t.Error(err)
				return
			}
			of.Close()

			outPath := path.Join(testDir, tt.outPath)

			if output, err := exec.Command("diff", "-u", outPath, of.Name()).CombinedOutput(); err != nil {
				t.Errorf("diff %v:\n%s", err, string(output))
			}
		})
	}
}
