package utils

import (
	"strings"
)

func BoolToUInt8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func Dedup(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, e := range in {
		e = strings.TrimRight(e, "/")
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}
