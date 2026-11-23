package artifact

import (
	"os"
	"path/filepath"
	"strings"
)

// CommentFile holds a one-line, human-written summary of what a run measured.
const CommentFile = "comment.txt"

// NormalizeComment folds a comment to one trimmed line so it fits a picker row.
func NormalizeComment(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// Comment returns the run directory's summary, or "" when none is set.
func Comment(dir string) string {
	data, err := optionalFile(filepath.Join(dir, CommentFile))
	if err != nil {
		return ""
	}
	return NormalizeComment(string(data))
}

// SetComment replaces a run's summary; an empty text removes it.
func SetComment(root, key, text string) error {
	dir, err := Resolve(root, key)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, CommentFile)
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return os.ErrInvalid
	}
	text = NormalizeComment(text)
	if text == "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	tmp, err := os.CreateTemp(dir, ".comment-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(text + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ProjectSummary is the first line of benches/<project>/AGENTS.md, used when a run has no comment.
func ProjectSummary(root, project string) string {
	data, err := optionalFile(filepath.Join(root, "benches", project, "AGENTS.md"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line = NormalizeComment(line); line != "" {
			return line
		}
	}
	return ""
}
