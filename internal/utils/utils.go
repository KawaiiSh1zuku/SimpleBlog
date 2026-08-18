package utils

import "strings"

func MapLogLevel(level string) int {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return -4
	case "INFO":
		return 0
	case "WARN":
		return 4
	case "ERROR":
		return 8
	default:
		return 0 // Default to INFO level if unknown
	}
}
