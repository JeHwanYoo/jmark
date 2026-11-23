package benchmark

import (
	"strings"
	"unicode"
)

// QuoteArg returns s as one word of a command template. Plain words stay unchanged.
func QuoteArg(s string) string {
	if s != "" && strings.IndexFunc(s, needsQuote) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func needsQuote(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("'\"\\$`;&|<>()*?[]#~!", r)
}

// SplitArgs splits a command template into argv. Single quotes, double quotes,
// and backslashes outside quotes group words the way a POSIX shell does.
func SplitArgs(cmd string) []string {
	var args []string
	var word strings.Builder
	inWord := false
	var quote rune
	escaped := false
	for _, r := range cmd {
		switch {
		case escaped:
			word.WriteRune(r)
			escaped = false
		case quote != 0:
			if r == quote {
				quote = 0
			} else if r == '\\' && quote == '"' {
				escaped = true
			} else {
				word.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == '\\':
			escaped = true
			inWord = true
		case unicode.IsSpace(r):
			if inWord {
				args = append(args, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		args = append(args, word.String())
	}
	return args
}
