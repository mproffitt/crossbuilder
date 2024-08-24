package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/logr"
	"github.com/mproffitt/crossbuilder/pkg/generate/composition/build"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

var packages []build.CompositionBuilder

type t struct {
	p string
	e error
}

var log logr.Logger

func compilePlugins() []string {
	var (
		err   error
		paths []string
		cwd   string
	)

	paths, err = filePathWalkDir(".", "main.go")
	if err != nil {
		log.Error(err, "error walking directory")
		return nil
	}

	for _, path := range paths {
		log.Info("found composition", "path", path)
	}

	cwd, err = os.Getwd()
	if err != nil {
		log.Error(err, "error getting current working directory")
	}

	if packages == nil {
		packages = make([]build.CompositionBuilder, 0)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(paths))

	plugins := make(chan t, len(paths))
	defer close(plugins)

	for _, composition := range paths {
		basename := strings.Join(strings.Split(composition, "/"), "_")
		plugin := filepath.Join(cwd, "plugins", fmt.Sprintf("%s.so", basename))
		go compile(plugin, composition, &wg, plugins)
	}

	wg.Wait()
	log.Info("done compiling plugins")

	pluginPaths := make([]string, 0)

	var i int = 1
	for plugin := range plugins {
		if plugin.e != nil {
			log.Info("error compiling", "plugin", plugin.e)
		} else {
			pluginPaths = append(pluginPaths, plugin.p)
			log.Info(fmt.Sprintf("(%d of %d)", i, len(paths)), "compiled plugin", plugin.p)
		}

		if i == len(paths) {
			break
		}
		i++
	}
	log.Info("compiled all plugins")
	return pluginPaths
}

// Dynamically compile the plugin
// nolint:gosec
func compile(path, composition string, wg *sync.WaitGroup, plugins chan t) {
	defer wg.Done()

	plugin := t{
		p: path,
	}

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

	for retries := 3; retries > 0; retries-- {
		if err := runCmd(args, env, composition); err != nil {
			plugin.e = errors.Wrap(err, fmt.Sprintf("error compiling plugin %q", composition))
			break
		}

		log.Info("checking for plugin", path)
		_, err := os.Stat(path)
		if err == nil {
			break
		}

		if os.IsNotExist(err) && retries > 0 {
			<-time.After(10 * time.Millisecond)
		}

		if retries <= 0 {
			plugin.e = fmt.Errorf("error compiling plugin '%q' - retries exhausted", composition)
			break
		}

		log.Info("error: plugin failed to show up, will retry", "compositition", composition)
	}
	plugins <- plugin
}

func runCmd(args, env []string, wd string) error {
	// MAX wait time for a build to complete is 5 minutes
	var duration time.Duration = 300 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = wd
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)
	cmd.WaitDelay = duration

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cmd.Stdout = stdout
	cmd.Stderr = stderr
	in := bufio.NewScanner(io.MultiReader(stdout, stderr))

	log.Info(fmt.Sprintf("running '%q", cmd.String()), "in", wd)
	if err := cmd.Start(); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error starting build for %q", wd))
		return err
	}

	go func() {
		<-ctx.Done()
		_ = cmd.Cancel()
		if ctx.Err() == context.DeadlineExceeded {
			log.Info("timeout exceeded")
		}
	}()

	err := cmd.Wait()
	for in.Scan() {
		log.Info(in.Text())
	}

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error running build for %q", wd))
	}
	log.Info(fmt.Sprintf("finished '%q'", cmd.String()), "in", wd)
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
			if err == nil {
				switch {
				case strings.HasPrefix(path, "crossbuilder"),
					strings.HasPrefix(path, "internal"),
					strings.HasPrefix(path, "pkg"):
					break
				default:
					files = append(files, path)
				}
			}
		}
		return nil
	})
	return files, err
}

func runGenerators() {
	var (
		err                                 error
		paths, primaryPaths, secondaryPaths []string
		cwd                                 string
	)

	cwd, err = os.Getwd()
	if err != nil {
		log.Error(err, "error getting current working directory")
		return
	}

	paths, err = filePathWalkDir(".", "generate.go")
	if err != nil {
		log.Error(err, "error walking directory")
		return
	}

	// Both internal and pkg should be compiled first
	// to ensure that any required code is available
	// for compositions to embed.
	for _, path := range paths {
		switch {
		case strings.HasPrefix(path, "internal"):
			fallthrough
		case strings.HasPrefix(path, "pkg"):
			primaryPaths = append(primaryPaths, path)
		default:
			secondaryPaths = append(secondaryPaths, path)
		}
	}

	for _, path := range append(primaryPaths, secondaryPaths...) {
		var args []string = []string{
			"generate", "./...",
		}

		var env []string = []string{
			"PATH=" + os.Getenv("PATH") + ":" + filepath.Join(cwd, "crossbuilder", "bin"),
		}

		err = runCmd(args, env, path)
		if err != nil {
			log.Error(err, "error running generator", "path", path)
			return
		}
	}
}

func main() {
	zl := zap.New(zap.UseDevMode(true))
	log := zl.WithName("crossbuilder")
	ctrl.SetLogger(zl)

	log.Info("running generators")
	runGenerators()
	log.Info("Compiling plugins")

	var paths []string = compilePlugins()
	for _, path := range paths {
		log.Info("loading", "plugin", path)
		if err := loadPlugin(path); err != nil {
			log.Error(err, "error loading plugin", "path", path)
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
		log.Error(err, "error building compositions")
	}
}
