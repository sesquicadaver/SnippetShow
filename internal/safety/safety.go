// Package safety detects dangerous shell constructs before Run (Phase 3 / P3-9…P3-10).
package safety

import (
	"regexp"
	"strings"
)

// Result describes whether a command needs user confirmation.
type Result struct {
	Dangerous bool     `json:"dangerous"`
	Reasons   []string `json:"reasons"`
}

// Command-word detection: optional wrapper prefixes (env/command/nice/nohup) and
// optional path directory prefix so /usr/bin/sudo and env sudo still confirm (H2).
const (
	cmdSep  = `(?:^|[;&|]|\|\||&&)`
	cmdWrap = `(?:(?:env|command|nice|nohup)\s+)*`
	cmdPath = `(?:\S*/)?`
)

var (
	// (?m) so shebang-prefixed multi-line scripts still match line-leading commands.
	reSudo = regexp.MustCompile(`(?im)` + cmdSep + `\s*` + cmdWrap + cmdPath + `sudo\b`)
	reDd   = regexp.MustCompile(`(?im)` + cmdSep + `\s*` + cmdWrap + cmdPath + `dd\s+`)
	reMkfs = regexp.MustCompile(`(?im)` + cmdSep + `\s*` + cmdWrap + cmdPath + `mkfs(?:\.[a-z0-9]+)?\b`)
	// rm may appear after sudo / whitespace / env prefixes — match as a command word.
	reRm = regexp.MustCompile(`(?im)(?:^|[;&|\s]|\|\||&&)` + cmdWrap + cmdPath + `rm\b([^;&|]*)`)
)

// Analyze inspects a command for rm -rf / dd / mkfs / sudo patterns (and chained variants).
func Analyze(command string) Result {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return Result{}
	}
	var reasons []string
	if reSudo.MatchString(cmd) {
		reasons = append(reasons, "sudo")
	}
	if reDd.MatchString(cmd) {
		reasons = append(reasons, "dd")
	}
	if reMkfs.MatchString(cmd) {
		reasons = append(reasons, "mkfs")
	}
	if hasRmRF(cmd) {
		reasons = append(reasons, "rm -rf")
	}
	return Result{Dangerous: len(reasons) > 0, Reasons: reasons}
}

func hasRmRF(cmd string) bool {
	matches := reRm.FindAllStringSubmatch(cmd, -1)
	for _, g := range matches {
		args := " " + strings.ToLower(g[1]) + " "
		combined := regexp.MustCompile(`\s-[a-z]*r[a-z]*f|\s-[a-z]*f[a-z]*r`).MatchString(args)
		if combined {
			return true
		}
		hasR := strings.Contains(args, "--recursive") || hasShortFlag(args, 'r')
		hasF := strings.Contains(args, "--force") || hasShortFlag(args, 'f')
		if hasR && hasF {
			return true
		}
	}
	return false
}

func hasShortFlag(args string, letter byte) bool {
	re := regexp.MustCompile(`\s-[a-z]*` + string(letter) + `[a-z]*(\s|$)`)
	return re.MatchString(args)
}
