package core

import "bytes"

// SplitFrontMatter splits a leading block fenced by "---" lines off content.
// If there is no closed block, front is nil and body is content.
func SplitFrontMatter(content []byte) (front, body []byte) {
	lines := bytes.SplitAfter(content, []byte("\n"))

	if !isFrontMatterFence(lines[0]) {
		return nil, content
	}

	start := len(lines[0])
	offset := start

	for _, line := range lines[1:] {
		if isFrontMatterFence(line) {
			return content[start:offset], content[offset+len(line):]
		}

		offset += len(line)
	}

	return nil, content
}

// isFrontMatterFence reports whether line is "---" and its line ending, if any.
func isFrontMatterFence(line []byte) bool {
	rest, ok := bytes.CutPrefix(line, []byte("---"))

	return ok && (len(rest) == 0 || string(rest) == "\n" || string(rest) == "\r\n")
}
