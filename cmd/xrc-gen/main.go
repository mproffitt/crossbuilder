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

	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/logr"
	"github.com/mproffitt/crossbuilder/pkg/generate/composition/build"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const poolSize = 10

type pathchan struct {
	p string // path
	e error  // error
}

type plug struct {
	plugin      string
	composition string
	plugins     chan pathchan
}

var packages []build.CompositionBuilder

func setupPool(plugins chan plug, wg *sync.WaitGroup, log logr.Logger) []chan bool {
	var stop []chan bool = make([]chan bool, poolSize)
	for i := 0; i < poolSize; i++ {
		stop[i] = make(chan bool)
		go func(i int) {
			for {
				select {
				case <-stop[i]:
					return
				case p := <-plugins:
					compile(p.plugin, p.composition, wg, p.plugins, log)
				}
			}
		}(i)
	}
	return stop
}

func compilePlugins(log logr.Logger) []string {
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

	plugins := make(chan plug, len(paths))
	pchan := make(chan pathchan, len(paths))
	defer close(plugins)
	defer close(pchan)

	log.Info("starting workers")
	stopchan := setupPool(plugins, &wg, log)

	for _, composition := range paths {
		basename := strings.Join(strings.Split(composition, "/"), "_")
		plugin := filepath.Join(cwd, "plugins", fmt.Sprintf("%s.so", basename))
		plugins <- plug{
			plugin:      plugin,
			composition: composition,
			plugins:     pchan,
		}
	}

	wg.Wait()
	log.Info("done compiling plugins")

	pluginPaths := make([]string, 0)

	var i int = 1
	for plugin := range pchan {
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

	log.Info("stopping all workers")
	for _, stop := range stopchan {
		stop <- true
		close(stop)
	}

	log.Info("compiled all plugins")
	return pluginPaths
}

// Dynamically compile the plugin
// nolint:gosec
func compile(path, composition string, wg *sync.WaitGroup, plugins chan pathchan, log logr.Logger) {
	defer wg.Done()

	plugin := pathchan{
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
		if err := runCmd(args, env, composition, log); err != nil {
			plugin.e = errors.Wrap(err, fmt.Sprintf("error compiling plugin %q", composition))
			break
		}

		log.Info("checking for plugin", "path", path)
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

func runCmd(args, env []string, wd string, log logr.Logger) error {
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
				case strings.HasPrefix(path, "crossbuilder"):
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

func runGenerators(log logr.Logger) {
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
		case strings.HasPrefix(path, "internal"), strings.HasPrefix(path, "pkg"):
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

		err = runCmd(args, env, path, log)
		if err != nil {
			log.Error(err, "error running generator", "path", path)
			return
		}
	}
}

func main() {
	zl := zap.New(zap.UseDevMode(true), zap.Level(zapcore.Level(-3)))
	log := zl.WithName("crossbuilder")
	ctrl.SetLogger(log)

	log.Info("running generators")
	runGenerators(log)
	log.Info("Compiling plugins")

	var paths []string = compilePlugins(log)
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
