package service

import (
	stdhtml "html"
	"io"
	"regexp"
	"strings"

	nethtml "golang.org/x/net/html"
)

var inlineMarkdownImagePattern = regexp.MustCompile(`!\[[^\]\r\n]*\]\(\s*(?:<([^>\r\n]+)>|([^\s)]+))(?:\s+(?:"[^"\r\n]*"|'[^'\r\n]*'|\([^\r\n)]*\)))?\s*\)`)

func firstPostImageURL(content string) *string {
	masked := maskMarkdownCode(content)
	bestOffset := len(masked) + 1
	bestURL := ""
	consider := func(offset int, candidate string) {
		candidate, ok := validExternalURL(candidate, false)
		if ok && offset < bestOffset {
			bestOffset = offset
			bestURL = candidate
		}
	}

	for _, match := range inlineMarkdownImagePattern.FindAllStringSubmatchIndex(masked, -1) {
		start, end := match[2], match[3]
		if start < 0 {
			start, end = match[4], match[5]
		}
		consider(match[0], stdhtml.UnescapeString(masked[start:end]))
	}

	tokenizer := nethtml.NewTokenizer(strings.NewReader(masked))
	offset := 0
	for {
		tokenType := tokenizer.Next()
		raw := tokenizer.Raw()
		tokenOffset := offset
		offset += len(raw)
		if tokenType == nethtml.ErrorToken {
			if tokenizer.Err() != io.EOF {
				return nil
			}
			break
		}
		if tokenType != nethtml.StartTagToken && tokenType != nethtml.SelfClosingTagToken {
			continue
		}
		token := tokenizer.Token()
		if !strings.EqualFold(token.Data, "img") {
			continue
		}
		for _, attribute := range token.Attr {
			if strings.EqualFold(attribute.Key, "src") {
				consider(tokenOffset, attribute.Val)
				break
			}
		}
	}

	if bestURL == "" {
		return nil
	}
	return &bestURL
}

func maskMarkdownCode(content string) string {
	source := []byte(content)
	masked := append([]byte(nil), source...)
	var fenceMarker byte
	fenceLength := 0

	for lineStart := 0; lineStart < len(source); {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		marker, length, after := markdownFence(source, lineStart, lineEnd)
		switch {
		case fenceMarker == 0 && length >= 3:
			fenceMarker, fenceLength = marker, length
			maskBytes(masked, lineStart, lineEnd)
		case fenceMarker != 0:
			isClosing := marker == fenceMarker && length >= fenceLength && onlyMarkdownWhitespace(source, after, lineEnd)
			maskBytes(masked, lineStart, lineEnd)
			if isClosing {
				fenceMarker, fenceLength = 0, 0
			}
		default:
			maskInlineCode(source, masked, lineStart, lineEnd)
		}
		lineStart = lineEnd + 1
	}
	return string(masked)
}

func markdownFence(source []byte, start, end int) (byte, int, int) {
	position := start
	for position < end && position-start < 3 && source[position] == ' ' {
		position++
	}
	if position >= end || (source[position] != '`' && source[position] != '~') {
		return 0, 0, position
	}
	marker := source[position]
	runStart := position
	for position < end && source[position] == marker {
		position++
	}
	return marker, position - runStart, position
}

func onlyMarkdownWhitespace(source []byte, start, end int) bool {
	for position := start; position < end; position++ {
		if source[position] != ' ' && source[position] != '\t' && source[position] != '\r' {
			return false
		}
	}
	return true
}

func maskInlineCode(source, masked []byte, start, end int) {
	for position := start; position < end; {
		if source[position] != '`' {
			position++
			continue
		}
		runStart := position
		for position < end && source[position] == '`' {
			position++
		}
		runLength := position - runStart
		closing := findClosingBackticks(source, position, end, runLength)
		if closing < 0 {
			continue
		}
		maskBytes(masked, runStart, closing+runLength)
		position = closing + runLength
	}
}

func findClosingBackticks(source []byte, start, end, length int) int {
	for position := start; position < end; {
		if source[position] != '`' {
			position++
			continue
		}
		runStart := position
		for position < end && source[position] == '`' {
			position++
		}
		if position-runStart == length {
			return runStart
		}
	}
	return -1
}

func maskBytes(value []byte, start, end int) {
	for position := start; position < end; position++ {
		if value[position] != '\r' && value[position] != '\n' {
			value[position] = ' '
		}
	}
}
