package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/catalog"
)

func ValidRun(key string) bool {
	parts := strings.Split(key, "/")
	return len(parts) == 3 && (parts[0] == "private" || parts[0] == "public") && catalog.ValidID(parts[1]) && catalog.ValidID(parts[2])
}

// List returns visibility/project/run keys. Hidden and symbolic directories are excluded.
func List(root, visibility, project string) ([]string, error) {
	if visibility != "" && visibility != "private" && visibility != "public" {
		return nil, fmt.Errorf("invalid visibility %q", visibility)
	}
	if project != "" && !catalog.ValidID(project) {
		return nil, fmt.Errorf("invalid project id %q", project)
	}
	var runs []string
	for _, scope := range []string{"private", "public"} {
		if visibility != "" && scope != visibility {
			continue
		}
		base := filepath.Join(root, "artifacts", scope)
		projects, err := directories(base)
		if err != nil {
			return nil, err
		}
		for _, id := range projects {
			if !catalog.ValidID(id) || (project != "" && id != project) {
				continue
			}
			entries, err := directories(filepath.Join(base, id))
			if err != nil {
				return nil, err
			}
			for _, run := range entries {
				if !catalog.ValidID(run) {
					continue
				}
				path := filepath.Join(base, id, run, "result.ndjson")
				info, err := os.Lstat(path)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return nil, err
				}
				if info.Mode().IsRegular() {
					runs = append(runs, scope+"/"+id+"/"+run)
				}
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(runs)))
	return runs, nil
}

func directories(path string) ([]string, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a regular directory: %s", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// Resolve rejects links and paths outside a complete artifact run.
func Resolve(root, key string) (string, error) {
	if !ValidRun(key) {
		return "", fmt.Errorf("invalid pick: %s", key)
	}
	path := filepath.Join(root, "artifacts")
	for _, part := range append([]string{""}, strings.Split(key, "/")...) {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("missing or unsafe artifact run: %s", key)
		}
	}
	info, err := os.Lstat(filepath.Join(path, "result.ndjson"))
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("missing artifact: %s/result.ndjson", key)
	}
	return path, nil
}

// Move changes visibility without modifying result contents or the Git index.
func Move(root, key, target string) (string, error) {
	if target != "public" && target != "private" {
		return "", fmt.Errorf("invalid visibility %q", target)
	}
	source, err := Resolve(root, key)
	if err != nil {
		return "", err
	}
	parts := strings.Split(key, "/")
	if parts[0] == target {
		return key, nil
	}
	destinationKey := target + "/" + parts[1] + "/" + parts[2]
	parent := filepath.Join(root, "artifacts", target, parts[1])
	// Ensure destination ancestors cannot redirect the move outside the store.
	for _, path := range []string{filepath.Dir(parent), parent} {
		if err := os.Mkdir(path, 0755); err != nil && !os.IsExist(err) {
			return "", err
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("unsafe artifact directory: %s", path)
		}
	}
	destination := filepath.Join(parent, parts[2])
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return "", fmt.Errorf("destination already exists: %s", destinationKey)
	}
	if err := os.Rename(source, destination); err != nil {
		return "", err
	}
	_ = os.Remove(filepath.Dir(source))
	return destinationKey, nil
}
