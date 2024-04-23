package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/mproffitt/crossbuilder/pkg/generate/composition/build"
	"github.com/pkg/errors"
)

var packages []build.CompositionBuilder

func compilePlugins() (paths []string) {
	paths = make([]string, 0)
	var (
		cwd string
		err error
	)

	if cwd, err = os.Getwd(); err != nil {
		log.Fatal(err)
	}

	var comppath = filepath.Join(cwd, "compositions")
	compositions, err := filePathWalkDir(comppath)
	if err != nil {
		log.Fatal(err)
	}

	/*if err := createGoMod(); err != nil {
		log.Fatal(err)
	}*/

	for _, composition := range compositions {
		var basename string = filepath.Base(composition)
		var path string = filepath.Join("plugins", fmt.Sprintf("%s.so", basename))
		if err := compile(path, composition); err != nil {
			log.Println(err)
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func createGoMod() (err error) {
	if err = copyFile("/crossbuilder/go.mod", "go.mod"); err != nil {
		return
	}

	if err = copyFile("/crossbuilder/go.sum", "go.sum"); err != nil {
		return
	}

	return tidy()
}

func tidy() (err error) {
	cmd := exec.Command("go", "mod", "tidy")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		err = errors.Wrap(err, "error starting go mod tidy")
		return err
	}

	err = cmd.Wait()
	in := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for in.Scan() {
		log.Println(in.Text())
	}

	if err != nil {
		err = errors.Wrap(err, "error running go mod tidy")
		return err
	}

	return nil
}

func copyFile(src, dst string) (err error) {
	src = filepath.Clean(src)
	var input []byte
	if input, err = os.ReadFile(src); err != nil {
		return
	}

	return os.WriteFile(dst, input, 0600)
}

func compile(path, composition string) (err error) {
	// Dynamically compile the plugin
	// nolint:gosec
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-trimpath", "-o", path, composition)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CGO_ENABLED=1")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error starting build for %q", path))
		return err
	}

	err = cmd.Wait()
	in := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for in.Scan() {
		log.Println(in.Text())
	}

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error running build for %q", path))
	}

	log.Println("compiled plugin", path)
	return nil
}

func loadPlugin(path string) (err error) {
	var (
		plug    *plugin.Plugin
		sym     plugin.Symbol
		builder build.CompositionBuilder
		ok      bool
	)
	if plug, err = plugin.Open(path); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error opening plugin %q", path))
		return
	}

	if sym, err = plug.Lookup("Builder"); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error loading symbol 'Builder' from plugin %q", path))
		return
	}

	if builder, ok = sym.(build.CompositionBuilder); !ok {
		err = errors.New("unexpected type from module symbol - must be of type 'CompositionBuilder'")
		return
	}

	packages = append(packages, builder)
	return
}

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		var rcount = strings.Count(root, "/")
		if d.IsDir() && strings.Count(path, "/") == rcount+1 {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func main() {
	if packages == nil {
		packages = make([]build.CompositionBuilder, 0)
	}

	var (
		cwd string
		err error
	)

	if cwd, err = os.Getwd(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(cwd)
	fmt.Println("=====================================")
	var paths []string = compilePlugins()
	for _, path := range paths {
		if err := loadPlugin(path); err != nil {
			log.Println("error:", err)
			continue
		}
	}

	// Clean up
	_ = os.RemoveAll("plugins")

	runner := build.NewRunner(build.RunnerConfig{
		Writer:  build.NewDirectoryWriter("package/compositions"),
		Builder: packages,
	})

	if err := runner.Build(); err != nil {
		log.Fatal(err)
	}

}
