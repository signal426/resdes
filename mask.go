package resdes

import (
	"strings"
	"unicode"
)

func NormalizePath(path string) string {
	sb := strings.Builder{}
	var toUpper bool
	for _, c := range path {
		if c == '_' {
			toUpper = true
			continue
		}
		if toUpper {
			sb.WriteRune(unicode.ToUpper(c))
			toUpper = false
			continue
		}
		sb.WriteRune(c)
	}
	return sb.String()
}

func GetPathsFromMask(fieldMask ...string) map[string]struct{} {
	if fieldMask == nil || len(fieldMask) == 0 {
		return nil
	}
	paths := make(map[string]struct{})
	for _, f := range fieldMask {
		paths[NormalizePath(f)] = struct{}{}
	}
	return paths
}

func IsPathInMask(path string, paths map[string]struct{}) bool {
	if paths == nil {
		return false
	}
	_, inMask := paths[path]
	return inMask
}
