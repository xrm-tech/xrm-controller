package utils

import (
	"context"
	"os/exec"
	"time"
)

func ExecCmd(timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	return string(out), err
}
