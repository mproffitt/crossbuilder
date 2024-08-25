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
	"sync"
	"time"

	"github.com/mproffitt/crossbuilder/pkg/generate/composition/build"
	"github.com/pkg/errors"
)

var packages []build.CompositionBuilder

type t struct {
	p string
	e error
}

func compilePlugins() []string {
	var (
		err   error
		paths []string
		cwd   string
	)

	paths, err = filePathWalkDir(".", "main.go")
	if err != nil {
		log.Fatal(err)
	}

	for _, path := range paths {
		log.Println("found composition path", path)
	}

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if packages == nil {
		packages = make([]build.CompositionBuilder, 0)
	}

	wg := sync.WaitGroup{}
	plugins := make(chan t, len(paths)-1)
	defer close(plugins)

	for _, composition := range paths {
		basename := strings.Join(strings.Split(composition, "/"), "_")
		plugin := filepath.Join(cwd, "plugins", fmt.Sprintf("%s.so", basename))
		go compile(plugin, composition, &wg, plugins)
	}
	wg.Wait()

	pluginPaths := make([]string, 0)
	var i int = 1
	for plugin := range plugins {
		if plugin.e != nil {
			log.Println("error compiling plugin", plugin.e)
		} else {
			pluginPaths = append(pluginPaths, plugin.p)
			log.Printf("(%d of %d) compiled plugin %s", i, len(paths), plugin.p)
		}

		if i == len(paths) {
			break
		}
		i++
	}
	log.Println("compiled all plugins")
	return pluginPaths
}

// Dynamically compile the plugin
// nolint:gosec
func compile(path, composition string, wg *sync.WaitGroup, plugins chan t) {
	wg.Add(1)
	defer wg.Done()

	var args []string = []string{
		"build",
		"-buildmode=plugin",
		"-trimpath",
		"-o", path,
		fmt.Sprintf("-ldflags=-X main.TemplateBasePath=%s", composition),
		".",
	}

	var env []string = []string{
		"CGO_ENABLED=1",
	}

	retries := 3
	for {
		if err := runCmd(args, env, composition); err != nil {
			plugins <- t{
				e: err,
			}
			return
		}

		_, err := os.Stat(path)
		if err == nil {
			break
		}

		if retries == 0 {
			plugins <- t{
				e: errors.Wrap(err, fmt.Sprintf("error compiling plugin %q", composition)),
			}
			return
		}

		retries--

		if os.IsNotExist(err) && retries > 0 {
			log.Println("error: plugin failed to show up, will retry", composition)
			<-time.After(10 * time.Millisecond)
			continue
		}
	}

	plugins <- t{
		p: path,
	}
}

func runCmd(args, env []string, wd string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = wd
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	log.Println("running", cmd.String(), "in", wd)
	if err := cmd.Start(); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error starting build for %q", wd))
		return err
	}

	err := cmd.Wait()

	in := bufio.NewScanner(io.MultiReader(stdout, stderr))

	for in.Scan() {
		log.Println(in.Text())
	}

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error running build for %q", wd))
	}

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

func filePathWalkDir(root string, stopAt string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			_, err := os.Stat(filepath.Join(path, stopAt))
			if err == nil && !strings.Contains(path, "cmd") {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func runGenerators() {
	var (
		err   error
		paths []string
		cwd   string
	)

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	paths, err = filePathWalkDir(".", "generate.go")
	if err != nil {
		log.Fatal(err)
	}

	for _, path := range paths {
		var args []string = []string{"generate", "./..."}
		var env []string = []string{
			"PATH=" + os.Getenv("PATH") + ":" + filepath.Join(cwd, "crossbuilder", "bin"),
		}

		err = runCmd(args, env, path)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var (
		cwd string
		err error
	)

	if cwd, err = os.Getwd(); err != nil {
		log.Fatal(err)
	}

	log.Println("Running generators")
	runGenerators()

	log.Println(cwd)
	log.Println("Compiling plugins")

	var paths []string = compilePlugins()
	for _, path := range paths {
		log.Println("loading plugin", path)
		if err := loadPlugin(path); err != nil {
			log.Println("error:", err)
			continue
		}
	}

	runner := build.NewRunner(
		build.RunnerConfig{
			Writer:  build.NewDirectoryWriter("apis"),
			Builder: packages,
		},
	)

	if err := runner.Build(); err != nil {
		log.Fatal(err)
	}
}
