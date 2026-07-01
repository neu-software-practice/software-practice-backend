package migrator

import "strings"

// splitStatements splits a SQL text into individual executable statements.
// It splits on semicolons, trims whitespace, and filters out empty statements
// and comment-only lines. This avoids requiring multiStatements=true in the DSN.
func splitStatements(content string) []string {
	raw := strings.Split(content, ";")
	var statements []string
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Skip lines that are comments only
		if isCommentOnly(s) {
			continue
		}
		statements = append(statements, s)
	}
	return statements
}

// isCommentOnly returns true if the statement consists solely of comment lines.
func isCommentOnly(s string) bool {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// If any non-empty line is not a comment, this is not comment-only
		if !strings.HasPrefix(trimmed, "--") && !strings.HasPrefix(trimmed, "#") {
			return false
		}
	}
	return true
}
