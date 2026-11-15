// Package manparser extracts summaries and OPTIONS/FLAGS from man pages.
// Phase 1: P1-9…P1-12. Policy D6: man parse is for installed programs in MVP.
package manparser

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Option is a parsed flag/command pair from a man page.
type Option struct {
	Flag        string `json:"flag"`
	Description string `json:"description"`
}

// Runner executes man / whatis / col (injectable in tests).
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout string, err error)
	RunPipeline(ctx context.Context, env []string, leftName string, leftArgs []string, rightName string, rightArgs []string) (string, error)
}

// OSRunner uses the real process environment.
type OSRunner struct{}

// Run executes a single command.
func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

// RunPipeline runs left | right (e.g. man | col).
func (OSRunner) RunPipeline(ctx context.Context, env []string, leftName string, leftArgs []string, rightName string, rightArgs []string) (string, error) {
	left := exec.CommandContext(ctx, leftName, leftArgs...)
	right := exec.CommandContext(ctx, rightName, rightArgs...)
	if len(env) > 0 {
		left.Env = env
	} else {
		left.Env = os.Environ()
	}
	right.Env = os.Environ()
	var out, errBuf bytes.Buffer
	pipe, err := left.StdoutPipe()
	if err != nil {
		return "", err
	}
	right.Stdin = pipe
	right.Stdout = &out
	right.Stderr = &errBuf
	left.Stderr = &bytes.Buffer{}
	if err := left.Start(); err != nil {
		return "", err
	}
	if err := right.Start(); err != nil {
		_ = left.Process.Kill()
		return "", err
	}
	_ = left.Wait()
	if err := right.Wait(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

// Parser reads man pages via Runner.
type Parser struct {
	Runner Runner
}

// New returns a Parser with the OS runner.
func New() *Parser {
	return &Parser{Runner: OSRunner{}}
}

func (p *Parser) runner() Runner {
	if p == nil || p.Runner == nil {
		return OSRunner{}
	}
	return p.Runner
}

// Summary returns a short description via whatis / man -f (P1-9).
func Summary(cmd string) (string, error) {
	return New().Summary(context.Background(), cmd)
}

// Summary returns a short description for cmd.
func (p *Parser) Summary(ctx context.Context, cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	r := p.runner()
	out, err := r.Run(ctx, "whatis", "-s", "1,8", cmd)
	if err != nil || strings.TrimSpace(out) == "" {
		out, err = r.Run(ctx, "man", "-f", cmd)
		if err != nil {
			return "", fmt.Errorf("manparser: summary %q: %w", cmd, err)
		}
	}
	return parseWhatis(out, cmd), nil
}

func parseWhatis(out, cmd string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// "ls (1) - list directory contents"
		if i := strings.Index(line, " - "); i >= 0 {
			return strings.TrimSpace(line[i+3:])
		}
		if i := strings.Index(line, " – "); i >= 0 { // en-dash
			return strings.TrimSpace(line[i+len(" – "):])
		}
		_ = cmd
	}
	return strings.TrimSpace(out)
}

// RawMan returns cleaned man text without ANSI/overstrike (P1-10).
func RawMan(cmd string) (string, error) {
	return New().RawMan(context.Background(), cmd)
}

// RawMan fetches MANWIDTH=1000 man -P cat | col -bx.
func (p *Parser) RawMan(ctx context.Context, cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	env := append(os.Environ(), "MANWIDTH=1000")
	out, err := p.runner().RunPipeline(ctx, env, "man", []string{"-P", "cat", cmd}, "col", []string{"-bx"})
	if err != nil {
		// Fallback without col.
		raw, err2 := p.runner().Run(ctx, "man", "-P", "cat", cmd)
		if err2 != nil {
			return "", fmt.Errorf("manparser: raw man %q: %w", cmd, err)
		}
		return stripOverstrike(raw), nil
	}
	return stripOverstrike(out), nil
}

// stripOverstrike removes classic man overstrike (c\bc) and backspaces.
func stripOverstrike(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\b' {
			continue
		}
		if i+2 < len(runes) && runes[i+1] == '\b' {
			// Keep the second character of overstrike pair.
			b.WriteRune(runes[i+2])
			i += 2
			continue
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

// ParseOptions extracts OPTIONS/COMMANDS/FLAGS for a live command (P1-11).
func ParseOptions(cmd string) ([]Option, error) {
	return New().ParseOptions(context.Background(), cmd)
}

// ParseOptions loads raw man text then runs the regex section parser.
func (p *Parser) ParseOptions(ctx context.Context, cmd string) ([]Option, error) {
	raw, err := p.RawMan(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return ParseOptionsFromText(raw), nil
}
