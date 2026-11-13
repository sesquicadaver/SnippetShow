package scanner

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

// Runner executes external package-manager / PATH helper commands (injectable in tests).
type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args ...string) (stdout string, err error)
}

// OSRunner uses the real process environment.
type OSRunner struct{}

// LookPath delegates to exec.LookPath.
func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Run executes name with args and returns trimmed stdout.
func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

// defaultTimeout bounds external PM queries.
const defaultTimeout = 60 * time.Second

func withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, defaultTimeout)
}
