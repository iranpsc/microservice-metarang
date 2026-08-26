package migrate

import (
	"strings"
	"unicode"
)

// SplitStatements splits a SQL script into executable statements.
// Semicolons inside single/double quotes or comments are ignored.
func SplitStatements(script string) []string {
	var (
		out        []string
		b          strings.Builder
		inSingle   bool
		inDouble   bool
		inLineCmt  bool
		inBlockCmt bool
	)

	flush := func() {
		stmt := strings.TrimSpace(b.String())
		b.Reset()
		if stmt != "" {
			out = append(out, stmt)
		}
	}

	runes := []rune(script)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}

		switch {
		case inLineCmt:
			b.WriteRune(r)
			if r == '\n' {
				inLineCmt = false
			}
		case inBlockCmt:
			b.WriteRune(r)
			if r == '*' && next == '/' {
				b.WriteRune(next)
				i++
				inBlockCmt = false
			}
		case inSingle:
			b.WriteRune(r)
			if r == '\\' && next != 0 {
				b.WriteRune(next)
				i++
				continue
			}
			if r == '\'' {
				if next == '\'' {
					b.WriteRune(next)
					i++
					continue
				}
				inSingle = false
			}
		case inDouble:
			b.WriteRune(r)
			if r == '\\' && next != 0 {
				b.WriteRune(next)
				i++
				continue
			}
			if r == '"' {
				inDouble = false
			}
		case r == '-' && next == '-':
			b.WriteRune(r)
			b.WriteRune(next)
			i++
			inLineCmt = true
		case r == '#':
			b.WriteRune(r)
			inLineCmt = true
		case r == '/' && next == '*':
			b.WriteRune(r)
			b.WriteRune(next)
			i++
			inBlockCmt = true
		case r == '\'':
			b.WriteRune(r)
			inSingle = true
		case r == '"':
			b.WriteRune(r)
			inDouble = true
		case r == ';':
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()

	filtered := make([]string, 0, len(out))
	for _, stmt := range out {
		if isCommentOnly(stmt) {
			continue
		}
		filtered = append(filtered, stmt)
	}
	return filtered
}

func isCommentOnly(stmt string) bool {
	s := strings.TrimSpace(stmt)
	for s != "" {
		switch {
		case strings.HasPrefix(s, "--") || strings.HasPrefix(s, "#"):
			if i := strings.IndexAny(s, "\r\n"); i >= 0 {
				s = strings.TrimSpace(s[i+1:])
				continue
			}
			return true
		case strings.HasPrefix(s, "/*"):
			end := strings.Index(s, "*/")
			if end < 0 {
				return true
			}
			s = strings.TrimSpace(s[end+2:])
		default:
			return false
		}
	}
	return true
}

func sanitizeMigrationSlug(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "_")
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "_")
}
