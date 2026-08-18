package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func normalizePlainText(value string, minimum, maximum int, multiline bool) (string, bool) {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.TrimSpace(value)
	length := utf8.RuneCountInString(value)
	if length < minimum || length > maximum || strings.ContainsAny(value, "<>") {
		return "", false
	}
	for _, char := range value {
		if char == '\n' || char == '\t' {
			if multiline {
				continue
			}
			return "", false
		}
		if unicode.IsControl(char) {
			return "", false
		}
	}
	return value, true
}
