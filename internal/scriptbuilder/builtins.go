package scriptbuilder

import (
	"path/filepath"
	"strings"
)

// builtinBinaryShadows maps shell builtins that shadow coreutils binaries used
// by man pages. Bash `[` requires a closing `]`; `/usr/bin/[ --help` does not (SS-25).
var builtinBinaryShadows = map[string]string{
	"[":    "/usr/bin/[",
	"test": "/usr/bin/test",
}

// ResolveBuiltinShadows rewrites a single command line so man-driven Run hits
// the real binary instead of a bash builtin (SS-25).
func ResolveBuiltinShadows(command string) string {
	if command == "" {
		return command
	}
	indentLen := len(command) - len(strings.TrimLeft(command, " \t"))
	indent := command[:indentLen]
	rest := command[indentLen:]
	if rest == "" || strings.HasPrefix(rest, "#") {
		return command
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return command
	}
	head := fields[0]
	// Already a path to the binary — leave alone.
	if strings.Contains(head, "/") {
		base := filepath.Base(head)
		if _, ok := builtinBinaryShadows[base]; ok {
			return command
		}
	}
	repl, ok := builtinBinaryShadows[head]
	if !ok {
		return command
	}
	fields[0] = repl
	return indent + strings.Join(fields, " ")
}

// ResolveBuiltinShadowsBody rewrites every non-shebang line of a script body (SS-25).
func ResolveBuiltinShadowsBody(body string) string {
	if body == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if i == 0 && strings.HasPrefix(trim, "#!") {
			continue
		}
		lines[i] = ResolveBuiltinShadows(line)
	}
	return strings.Join(lines, "\n")
}
