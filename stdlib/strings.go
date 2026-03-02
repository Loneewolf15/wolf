package stdlib

import (
	"strings"
	"unicode"
)

// WolfStrings implements the Wolf\Strings standard library.
// All functions are static and operate on string values.
type WolfStrings struct{}

// Contains checks if a string contains a substring.
func (WolfStrings) Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// StartsWith checks if a string starts with a prefix.
func (WolfStrings) StartsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// EndsWith checks if a string ends with a suffix.
func (WolfStrings) EndsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// Split divides a string by separator.
func (WolfStrings) Split(s, sep string) []string {
	return strings.Split(s, sep)
}

// Join concatenates elements with a separator.
func (WolfStrings) Join(parts []string, sep string) string {
	return strings.Join(parts, sep)
}

// Trim removes leading and trailing whitespace.
func (WolfStrings) Trim(s string) string {
	return strings.TrimSpace(s)
}

// TrimChars removes specific characters from both ends.
func (WolfStrings) TrimChars(s, chars string) string {
	return strings.Trim(s, chars)
}

// Upper converts to uppercase.
func (WolfStrings) Upper(s string) string {
	return strings.ToUpper(s)
}

// Lower converts to lowercase.
func (WolfStrings) Lower(s string) string {
	return strings.ToLower(s)
}

// Replace replaces all occurrences of old with new.
func (WolfStrings) Replace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// ReplaceFirst replaces only the first occurrence.
func (WolfStrings) ReplaceFirst(s, old, new string) string {
	return strings.Replace(s, old, new, 1)
}

// Length returns the string length.
func (WolfStrings) Length(s string) int {
	return len(s)
}

// Repeat repeats a string n times.
func (WolfStrings) Repeat(s string, count int) string {
	return strings.Repeat(s, count)
}

// Reverse reverses a string.
func (WolfStrings) Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Index returns the first index of substr, or -1.
func (WolfStrings) Index(s, substr string) int {
	return strings.Index(s, substr)
}

// Substring extracts a portion of a string.
func (WolfStrings) Substring(s string, start, end int) string {
	runes := []rune(s)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

// PadLeft pads a string on the left to the given length.
func (WolfStrings) PadLeft(s string, length int, pad string) string {
	for len(s) < length {
		s = pad + s
	}
	return s
}

// PadRight pads a string on the right to the given length.
func (WolfStrings) PadRight(s string, length int, pad string) string {
	for len(s) < length {
		s = s + pad
	}
	return s
}

// IsEmpty checks if a string is empty or whitespace-only.
func (WolfStrings) IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNumeric checks if a string contains only digits.
func (WolfStrings) IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsAlpha checks if a string contains only letters.
func (WolfStrings) IsAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// Title converts the first character of each word to uppercase.
func (WolfStrings) Title(s string) string {
	return strings.Title(s)
}
