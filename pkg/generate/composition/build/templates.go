package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var templates []string

func AddTemplate(path string) {
	templates = append(templates, path)
}

func LoadTemplate(path string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	path = strings.TrimSuffix(path, "*")

	templates = make([]string, 0)

	if !strings.HasSuffix(path, "/") {
		templates = append(templates, path)
	} else {
		if err := filepath.WalkDir(filepath.Join(cwd, path), walk); err != nil {
			return "", err
		}
	}

	var contents string
	for _, t := range templates {
		if _, err := os.Stat(t); err != nil {
			return "", err
		}

		var (
			b   []byte
			err error
		)

		t = filepath.Clean(t)
		if b, err = os.ReadFile(t); err != nil {
			return "", err
		}
		contents = strings.Join([]string{contents, string(b)}, "\n")
	}

	contents = strings.TrimSpace(contents)
	return contents, nil
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		templates = append(templates, s)
	}
	return nil
}
