package filter

import (
	"strings"

	"gopkg.in/yaml.v2"
)

type ResourceFilter struct {
	kindMatcher   func(string) bool
	nameMatcher   func(string) bool
	KindHighliter func(string) (string, int)
	NameHighliter func(string) (string, int)
}

func NewResourceFilter(pattern string) *ResourceFilter {
	kind, name := splitPattern(pattern, "/")
	return &ResourceFilter{
		kindMatcher:   createMatcher(kind),
		nameMatcher:   createMatcher(name),
		KindHighliter: createHighliter(kind),
		NameHighliter: createHighliter(name),
	}
}

func (filter *ResourceFilter) Match(doc yaml.MapSlice) bool {
	return filter.kindMatcher(GetKind(doc)) && filter.nameMatcher(GetName(doc))
}

func splitPattern(pattern string, sep string) (string, string) {
	if i := strings.Index(pattern, "/"); i >= 0 {
		return pattern[:i], pattern[i+1:]
	}
	if len(pattern) >= 1 && isUpper(pattern[0]) || len(pattern) >= 2 && pattern[0] == '^' && isUpper(pattern[1]) {
		return pattern, ""
	} else {
		return "", pattern
	}
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func createMatcher(pattern string) func(string) bool {
	exactStart, exactEnd := false, false
	if len(pattern) >= 1 && pattern[0] == '^' {
		exactStart = true
		pattern = pattern[1:]
	}
	if len(pattern) >= 1 && pattern[len(pattern)-1] == '$' {
		exactEnd = true
		pattern = pattern[:len(pattern)-1]
	}
	pattern = strings.ToLower(pattern)
	if exactStart && exactEnd {
		return func(s string) bool {
			return strings.ToLower(s) == pattern
		}
	} else if exactStart {
		return func(s string) bool {
			return strings.HasPrefix(strings.ToLower(s), pattern)
		}
	} else if exactEnd {
		return func(s string) bool {
			return strings.HasSuffix(strings.ToLower(s), pattern)
		}
	} else {
		return func(s string) bool {
			return strings.Contains(strings.ToLower(s), pattern)
		}
	}
}

func createHighliter(pattern string) func(string) (string, int) {
	exactStart, exactEnd := false, false
	if len(pattern) >= 1 && pattern[0] == '^' {
		pattern = pattern[1:]
		exactStart = true
	}
	if len(pattern) >= 1 && pattern[len(pattern)-1] == '$' {
		pattern = pattern[:len(pattern)-1]
		exactEnd = true
	}
	pattern = strings.ToLower(pattern)
	return func(s string) (string, int) {
		if len(pattern) == 0 {
			return s, 0
		}
		i := strings.Index(strings.ToLower(s), pattern)
		for {
			if i < 0 {
				return s, 0
			} else if exactStart && i != 0 || exactEnd && i+len(pattern) != len(s) {
				j := strings.Index(strings.ToLower(s[i+1:]), pattern)
				if j >= 0 {
					i += j + 1
				} else {
					i = j
				}
			} else {
				// https://askubuntu.com/questions/528928/how-to-do-underline-bold-italic-strikethrough-color-background-and-size-i
				return s[:i] + "\x1b[1;4m" + s[i:i+len(pattern)] + "\x1b[22;24m" + s[i+len(pattern):], len(pattern)
			}
		}
	}
}

// access kind
func GetKind(doc yaml.MapSlice) string {
	for _, item := range doc {
		// check if item is a map
		if item.Value == nil {
			continue
		}
		// check if item is a kind
		if item.Key == "kind" {
			return item.Value.(string)
		}
	}
	return ""
}

// access name nested in metadata
func GetName(doc yaml.MapSlice) string {
	for _, item := range doc {
		// check if item is a map
		if item.Value == nil {
			continue
		}
		// check if item is a metadata
		if item.Key == "metadata" {
			// iterate over metadata
			for _, metadata := range item.Value.(yaml.MapSlice) {
				// check if item is a name
				if metadata.Key == "name" {
					return metadata.Value.(string)
				}
			}
		}
	}
	return ""
}
