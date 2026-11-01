// Package text holds the small text helpers the harness shares.
package text

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var word = regexp.MustCompile(`[a-z0-9]+`)

// Tokens is the lower-case words and numbers of s: what two texts share however each is punctuated or
// laid out.
func Tokens(s string) []string {
	return word.FindAllString(strings.ToLower(s), -1)
}

// Mean is the average of xs, or 0 for none.
func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// Head is the first n characters of s, never cutting a character in half.
func Head(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// Tail is the last n characters of s.
func Tail(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

// Len is the length of s in characters.
func Len(s string) int {
	return utf8.RuneCountInString(s)
}
