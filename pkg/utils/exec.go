package utils

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ExecCmd(outFile string, timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	outErr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if FileExists(outFile) {
		if err := os.Rename(outFile, outFile+".old"); err != nil {
			return "", err
		}
	}

	f, err := os.OpenFile(outFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, _ = f.Write([]byte(cmd.Path + " '" + strings.Join(cmd.Args, "' '") + "'\n"))

	if err = cmd.Start(); err != nil {
		return "", err
	}

	var outBuf bytes.Buffer

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		b := scanner.Bytes()
		outBuf.Write(b)
		outBuf.WriteByte('\n')
		_, _ = f.Write(b)
		_, _ = f.Write([]byte{'\n'})
	}

	scanner = bufio.NewScanner(outErr)
	for scanner.Scan() {
		b := scanner.Bytes()
		outBuf.Write(b)
		outBuf.WriteByte('\n')
		_, _ = f.Write(b)
		_, _ = f.Write([]byte{'\n'})
	}

	err = cmd.Wait()

	return outBuf.String(), err
}
