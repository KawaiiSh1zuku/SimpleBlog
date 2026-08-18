package dto

import (
	"net/url"
	"strings"
	"unicode"
)

// safePlainText is an output-side defense for legacy rows created before the
// database plain-text constraints existed.
func safePlainText(value string) string {
	value = strings.NewReplacer("<", "＜", ">", "＞", "\r\n", "\n", "\r", "\n").Replace(value)
	return strings.Map(func(char rune) rune {
		if char == '\n' || char == '\t' || !unicode.IsControl(char) {
			return char
		}
		return -1
	}, value)
}

func safeHTTPURL(value string) string {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return value
}
