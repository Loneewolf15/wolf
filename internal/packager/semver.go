package packager

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

type Version struct {
	Major int
	Minor int
	Patch int
	Pre   string
}

func ParseVersion(v string) (*Version, error) {
	v = strings.TrimPrefix(v, "v")
	matches := versionRegex.FindStringSubmatch(v)
	if matches == nil {
		return nil, fmt.Errorf("invalid semantic version: %s", v)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
		Pre:   matches[4],
	}, nil
}

func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}
	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}
	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}
	// Simple pre-release comparison: pre-release < no pre-release
	if v.Pre == "" && other.Pre != "" {
		return 1
	}
	if v.Pre != "" && other.Pre == "" {
		return -1
	}
	if v.Pre != other.Pre {
		if v.Pre > other.Pre {
			return 1
		}
		return -1
	}
	return 0
}

func (v *Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// Satisfies checks if a version satisfies a given constraint (e.g., ^1.2.3, ~1.2.3, >=1.2.3, 1.2.3)
func Satisfies(version, constraint string) bool {
	v, err := ParseVersion(version)
	if err != nil {
		return false // Non-semver tags don't satisfy semver constraints
	}

	constraint = strings.TrimSpace(constraint)

	if constraint == "latest" || constraint == "*" {
		return v.Pre == "" // latest ignores pre-releases
	}

	// Exact match
	if cv, err := ParseVersion(constraint); err == nil {
		return v.Compare(cv) == 0
	}

	if cv, err := ParseVersion(strings.TrimPrefix(constraint, "^")); err == nil {
		if v.Pre != "" && cv.Pre == "" {
			return false
		}
		if v.Major != cv.Major {
			return false
		}
		if v.Major == 0 {
			if v.Minor != cv.Minor {
				return false
			}
			return v.Compare(cv) >= 0
		}
		return v.Compare(cv) >= 0
	}

	if strings.HasPrefix(constraint, "~") {
		cv, err := ParseVersion(strings.TrimPrefix(constraint, "~"))
		if err != nil {
			return false
		}
		if v.Major != cv.Major || v.Minor != cv.Minor {
			return false
		}
		return v.Compare(cv) >= 0
	}

	if strings.HasPrefix(constraint, ">=") {
		cv, err := ParseVersion(strings.TrimPrefix(constraint, ">="))
		if err != nil {
			return false
		}
		return v.Compare(cv) >= 0
	}

	if strings.HasPrefix(constraint, "<=") {
		cv, err := ParseVersion(strings.TrimPrefix(constraint, "<="))
		if err != nil {
			return false
		}
		return v.Compare(cv) <= 0
	}

	if strings.HasPrefix(constraint, ">") {
		cv, err := ParseVersion(strings.TrimPrefix(constraint, ">"))
		if err != nil {
			return false
		}
		return v.Compare(cv) > 0
	}

	if strings.HasPrefix(constraint, "<") {
		cv, err := ParseVersion(strings.TrimPrefix(constraint, "<"))
		if err != nil {
			return false
		}
		return v.Compare(cv) < 0
	}

	return false
}

// HighestCompatible takes a constraint and a list of available tags and returns the best match.
func HighestCompatible(constraint string, tags []string) string {
	var best *Version
	var bestStr string

	// If exact match doesn't have semver semantics (like a branch name), just return it if it exists.
	// But we only evaluate SemVer tags here.

	for _, tag := range tags {
		if Satisfies(tag, constraint) {
			v, _ := ParseVersion(tag)
			if best == nil || v.Compare(best) > 0 {
				best = v
				bestStr = tag
			}
		}
	}

	// Fallback for non-semver branch or exact tag if nothing found mathematically
	if bestStr == "" {
		for _, tag := range tags {
			if tag == constraint {
				return tag
			}
		}
		// If tags is empty or it didn't match, but it's a simple branch/commit name,
		// return it so git clone can attempt to fetch it directly.
		if !strings.ContainsAny(constraint, "^~<>=") {
			return constraint
		}
	}

	return bestStr
}
