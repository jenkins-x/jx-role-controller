package util

import "strings"

// StringMatchesAny returns true if the given text matches the includes/excludes lists
func StringMatchesAny(text string, includes, excludes []string) bool {
	for _, x := range excludes {
		if StringMatchesPattern(text, x) {
			return false
		}
	}
	if len(includes) == 0 {
		return true
	}
	for _, inc := range includes {
		if StringMatchesPattern(text, inc) {
			return true
		}
	}
	return false
}

// StringMatchesPattern returns true if the given text matches the includes/excludes lists
func StringMatchesPattern(text, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(text, prefix)
	}
	return text == pattern
}

func EnvVarBoolean(value string) bool {
	return value == "true" || value == "yes"
}

// StringArrayIndex returns the index in the slice which equals the given value
func StringArrayIndex(array []string, value string) int {
	for i, v := range array {
		if v == value {
			return i
		}
	}
	return -1
}
