package semver

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type comparator func(Version, Version) bool

var (
	compEQ comparator = func(v1 Version, v2 Version) bool {
		return v1.Compare(v2) == 0
	}
	compNE = func(v1 Version, v2 Version) bool {
		return v1.Compare(v2) != 0
	}
	compGT = func(v1 Version, v2 Version) bool {
		return v1.Compare(v2) == 1
	}
	compGE = func(v1 Version, v2 Version) bool {
		return v1.Compare(v2) >= 0
	}
	compLT = func(v1 Version, v2 Version) bool {
		return v1.Compare(v2) == -1
	}
	compLE = func(v1 Version, v2 Version) bool {
		return v1.Compare(v2) <= 0
	}
)

type versionRange struct {
	v Version
	c comparator
}

// rangeFunc creates a Range from the given versionRange.
func (vr *versionRange) rangeFunc() Range {
	return Range(func(v Version) bool {
		return vr.c(v, vr.v)
	})
}

// Range represents a range of versions.
// A Range can be used to check if a Version satisfies it:
//
//     range, err := semver.ParseRange(">1.0.0 <2.0.0")
//     range(semver.MustParse("1.1.1") // returns true
type Range func(Version) bool

// OR combines the existing Range with another Range using logical OR.
func (rf Range) OR(f Range) Range {
	return Range(func(v Version) bool {
		return rf(v) || f(v)
	})
}

// AND combines the existing Range with another Range using logical AND.
func (rf Range) AND(f Range) Range {
	return Range(func(v Version) bool {
		return rf(v) && f(v)
	})
}

// ParseRange parses a range and returns a Range.
// If the range could not be parsed an error is returned.
//
// Valid ranges are:
//   - "<1.0.0"
//   - "<=1.0.0"
//   - ">1.0.0"
//   - ">=1.0.0"
//   - "1.0.0", "=1.0.0", "==1.0.0"
//   - "!1.0.0", "!=1.0.0"
//
// A Range can consist of multiple ranges separated by space:
// Ranges can be linked by logical AND:
//   - ">1.0.0 <2.0.0" would match between both ranges, so "1.1.1" and "1.8.7" but not "1.0.0" or "2.0.0"
//   - ">1.0.0 <3.0.0 !2.0.3-beta.2" would match every version between 1.0.0 and 3.0.0 except 2.0.3-beta.2
//
// Ranges can also be linked by logical OR:
//   - "<2.0.0 || >=3.0.0" would match "1.x.x" and "3.x.x" but not "2.x.x"
//
// AND has a higher precedence than OR. It's not possible to use brackets.
//
// Ranges can be combined by both AND and OR
//
//  - `>1.0.0 <2.0.0 || >3.0.0 !4.2.1` would match `1.2.3`, `1.9.9`, `3.1.1`, but not `4.2.1`, `2.1.1`
func ParseRange(s string) (Range, error) {
	var expandedParts [][]string
	// split on boolean or ||
	orParts := regexp.MustCompile("\\s*\\|\\|\\s*").Split(s, -1)
	for _, part := range orParts {
		parsed := parseRange(part)
		if len(parsed) > 0 {
			expandedParts = append(expandedParts, parseRange(strings.TrimSpace(part)))
		}
	}

	var orFn Range
	for _, p := range expandedParts {
		var andFn Range
		for _, ap := range p {
			opStr, vStr, err := splitComparatorVersion(ap)
			if err != nil {
				return nil, err
			}
			vr, err := buildVersionRange(opStr, vStr)
			if err != nil {
				return nil, fmt.Errorf("Could not parse Range %q: %s", ap, err)
			}
			rf := vr.rangeFunc()

			// Set function
			if andFn == nil {
				andFn = rf
			} else { // Combine with existing function
				andFn = andFn.AND(rf)
			}
		}
		if orFn == nil {
			orFn = andFn
		} else {
			orFn = orFn.OR(andFn)
		}

	}
	return orFn, nil
}

// buildVersionRange takes a slice of 2: operator and version
// and builds a versionRange, otherwise an error.
func buildVersionRange(opStr, vStr string) (*versionRange, error) {
	c := parseComparator(opStr)
	if c == nil {
		return nil, fmt.Errorf("Could not parse comparator %q in %q", opStr, strings.Join([]string{opStr, vStr}, ""))
	}
	v, err := Parse(vStr)
	if err != nil {
		return nil, fmt.Errorf("Could not parse version %q in %q: %s", vStr, strings.Join([]string{opStr, vStr}, ""), err)
	}

	return &versionRange{
		v: v,
		c: c,
	}, nil

}

// inArray checks if a byte is contained in an array of bytes
func inArray(s byte, list []byte) bool {
	for _, el := range list {
		if el == s {
			return true
		}
	}
	return false
}

// splitComparatorVersion splits the comparator from the version.
// Input must be free of leading or trailing spaces.
func splitComparatorVersion(s string) (string, string, error) {
	// handle case where there is only a wildcard
	if s == "x" {
		return "", "x", nil
	}
	i := strings.IndexFunc(s, isVersionString)
	if i == -1 {
		return "", "", fmt.Errorf("Could not get version from string: %q", s)
	}
	return strings.TrimSpace(s[0:i]), s[i:], nil
}

func isVersionString(c rune) bool {
	return unicode.IsDigit(c) || c == 118 /* 118 == 'v' */
}

func parseComparator(s string) comparator {
	switch s {
	case "==":
		fallthrough
	case "":
		fallthrough
	case "=":
		return compEQ
	case ">":
		return compGT
	case ">=":
		return compGE
	case "<":
		return compLT
	case "<=":
		return compLE
	case "!":
		fallthrough
	case "!=":
		return compNE
	}

	return nil
}

// MustParseRange is like ParseRange but panics if the range cannot be parsed.
func MustParseRange(s string) Range {
	r, err := ParseRange(s)
	if err != nil {
		panic(`semver: ParseRange(` + s + `): ` + err.Error())
	}
	return r
}
