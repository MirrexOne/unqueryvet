package analyzer

import "regexp"

// allowedPatternCache is local to a single analysis pass. Invalid patterns cache as nil.
type allowedPatternCache map[string]*regexp.Regexp

func (c allowedPatternCache) matchString(pattern, query string) bool {
	re, ok := c[pattern]
	if !ok {
		re, _ = regexp.Compile(pattern)
		if c != nil {
			c[pattern] = re
		}
	}
	return re != nil && re.MatchString(query)
}
