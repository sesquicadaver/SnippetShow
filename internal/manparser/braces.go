package manparser

import (
	"strings"
)

// ExpandBraceAlternatives turns man syntax like "-ao{a|s|t|u}" into separate
// runnable options with annotated descriptions (SS-26).
// Flags without `{alt|…}` are returned unchanged as a single Option.
// Only the first brace group on the flag string is expanded (typical 7z style).
func ExpandBraceAlternatives(flag, desc string) []Option {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return nil
	}
	start := strings.IndexByte(flag, '{')
	end := strings.IndexByte(flag, '}')
	if start < 0 || end <= start+1 {
		return []Option{{Flag: flag, Description: strings.TrimSpace(desc)}}
	}
	inner := flag[start+1 : end]
	if !strings.Contains(inner, "|") {
		return []Option{{Flag: flag, Description: strings.TrimSpace(desc)}}
	}
	prefix := flag[:start]
	suffix := flag[end+1:]
	parts := strings.Split(inner, "|")
	out := make([]Option, 0, len(parts))
	baseDesc := strings.TrimSpace(desc)
	for _, part := range parts {
		alt := strings.TrimSpace(part)
		if alt == "" {
			continue
		}
		expanded := NormalizeFlagList(prefix + alt + suffix)
		if expanded == "" {
			continue
		}
		ann := baseDesc
		if ann != "" {
			ann += " "
		}
		ann += "(варіант: " + alt + ")"
		out = append(out, Option{Flag: expanded, Description: ann})
	}
	if len(out) == 0 {
		return []Option{{Flag: flag, Description: baseDesc}}
	}
	return out
}
