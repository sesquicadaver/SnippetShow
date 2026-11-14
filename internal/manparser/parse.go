package manparser

import (
	"regexp"
	"strings"
)

var (
	sectionHeader = regexp.MustCompile(`(?m)^(?:OPTIONS|COMMANDS|FLAGS|DESCRIPTION)\b.*$`)
	// Matches indented option lines: "       -a, --all" or "       -f FILE"
	optionLine = regexp.MustCompile(`(?m)^[ \t]{2,}((?:-{1,2}[\w][\w\-]*|=?[A-Z_][\w\-]*|,?[ \t]+){1,}.+?)(?:[ \t]{2,}|\n)(.*)$`)
	flagToken  = regexp.MustCompile(`(?:^|[,\s])(-{1,2}[\w][\w\-]*)`)
)

// ParseOptionsFromText extracts flag/description pairs from cleaned man text.
func ParseOptionsFromText(text string) []Option {
	sections := extractOptionSections(text)
	if len(sections) == 0 {
		sections = []string{text}
	}
	seen := map[string]struct{}{}
	var out []Option
	for _, sec := range sections {
		out = append(out, parseSection(sec, seen)...)
	}
	return out
}

func extractOptionSections(text string) []string {
	lines := strings.Split(text, "\n")
	var sections []string
	var cur []string
	in := false
	flush := func() {
		if len(cur) > 0 {
			sections = append(sections, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if in {
				cur = append(cur, line)
			}
			continue
		}
		// Man section headers are short lines written in ALL CAPS in the source text.
		if isManSectionHeader(trimmed) {
			upper := strings.ToUpper(trimmed)
			if isTargetSection(upper) {
				flush()
				in = true
				continue
			}
			if in {
				flush()
				in = false
			}
			continue
		}
		if in {
			cur = append(cur, line)
		}
	}
	flush()
	_ = sectionHeader
	return sections
}

func isManSectionHeader(trimmed string) bool {
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return false
	}
	// Original text must already be all caps (not merely ToUpper of prose).
	if trimmed != strings.ToUpper(trimmed) {
		return false
	}
	if strings.ContainsAny(trimmed, "a-z") {
		return false
	}
	words := strings.Fields(trimmed)
	if len(words) == 0 || len(words) > 3 {
		return false
	}
	switch words[0] {
	case "NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "COMMANDS", "FLAGS",
		"ENVIRONMENT", "FILES", "SEE", "AUTHOR", "AUTHORS", "BUGS", "EXAMPLES",
		"NOTES", "HISTORY", "COPYRIGHT", "REPORTING", "EXIT", "STANDARDS",
		"USAGE", "ARGUMENTS", "OPERANDS":
		return true
	default:
		return false
	}
}

func isTargetSection(upper string) bool {
	for _, h := range []string{"OPTIONS", "COMMANDS", "FLAGS"} {
		if upper == h || strings.HasPrefix(upper, h+" ") || strings.HasPrefix(upper, h+"\t") {
			return true
		}
	}
	return false
}

func parseSection(sec string, seen map[string]struct{}) []Option {
	var out []Option
	lines := strings.Split(sec, "\n")
	var curFlag, curDesc string
	flush := func() {
		flag := NormalizeFlagList(curFlag)
		desc := strings.TrimSpace(curDesc)
		if flag == "" {
			curFlag, curDesc = "", ""
			return
		}
		// Combined man syntax "-a, --all" becomes runnable "-a --all" (no comma glued to argv).
		key := flag
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, Option{Flag: flag, Description: desc})
		}
		curFlag, curDesc = "", ""
	}

	for _, line := range lines {
		if isOptionStart(line) {
			flush()
			flag, rest := splitOptionLine(line)
			curFlag = flag
			curDesc = strings.TrimSpace(rest)
			continue
		}
		if curFlag != "" {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if curDesc == "" {
				curDesc = trimmed
			} else {
				curDesc += " " + trimmed
			}
		}
	}
	flush()
	_ = optionLine
	_ = flagToken
	return out
}

func isOptionStart(line string) bool {
	if len(line) < 3 {
		return false
	}
	// Must be indented and start with - or -- after indent.
	trimmedLeft := strings.TrimLeft(line, " \t")
	if trimmedLeft == line {
		return false // not indented
	}
	return strings.HasPrefix(trimmedLeft, "-")
}

// NormalizeFlagList turns man option syntax into space-separated tokens safe for
// shell argv. Trailing commas (e.g. "--audible,") are stripped so Run does not
// pass "--audible," as a single unrecognized argument (SS-20).
// Only commas that separate alternative flags (", --all") are removed; commas
// inside values that are not followed by another "-" flag are kept.
func NormalizeFlagList(flag string) string {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(flag))
	for i := 0; i < len(flag); {
		if flag[i] == ',' {
			j := i + 1
			for j < len(flag) && (flag[j] == ' ' || flag[j] == '\t') {
				j++
			}
			rest := flag[j:]
			if strings.HasPrefix(rest, "-") || strings.HasPrefix(rest, "−") {
				b.WriteByte(' ')
				i = j
				continue
			}
		}
		b.WriteByte(flag[i])
		i++
	}
	s := strings.TrimSpace(b.String())
	s = strings.TrimRight(s, ",，")
	fields := strings.Fields(s)
	for i, f := range fields {
		fields[i] = strings.TrimRight(f, ",，")
	}
	return strings.Join(fields, " ")
}

func splitOptionLine(line string) (flag, rest string) {
	trimmed := strings.TrimSpace(line)
	// Split on 2+ spaces or a tab between flag list and description.
	idx := -1
	for i := 0; i < len(trimmed)-1; i++ {
		if trimmed[i] == '\t' {
			idx = i
			break
		}
		if trimmed[i] == ' ' && trimmed[i+1] == ' ' {
			idx = i
			break
		}
	}
	if idx >= 0 {
		return strings.TrimSpace(trimmed[:idx]), strings.TrimSpace(trimmed[idx:])
	}
	// Single-space fallback: "--help display this help"
	if strings.HasPrefix(trimmed, "--") {
		if sp := strings.IndexByte(trimmed, ' '); sp > 0 {
			return trimmed[:sp], strings.TrimSpace(trimmed[sp+1:])
		}
	}
	return trimmed, ""
}
