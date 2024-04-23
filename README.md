# Crossbuilder

Crossbuilder is a tool that allows the generation of Crossplane XRDs and
compositions from go code.

This tool builds on [`mistermx/crossbuilder`](https://github.com/MisterMX/crossbuilder)
to provide two tools, `xrd-gen` and `xrc-gen` which can be used to build crossplane
definitions and compositions.

## XRD Generation

Crossbuilder's `xrd-gen` wraps around [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
and [controller-gen](https://github.com/kubernetes-sigs/controller-tools) but
instead of generating CRDs (Custom Resource Definitions) it generates XRDs
(Composite Resource Definitions) that are part of the Crossplane ecosystem.

Crossbuilder has every feature that Kubebuilder's CRD generator provides plus
the ability to define XRD specific fields.

Take a look at the [xrd-gen examples](./examples/xrd-gen/apis/generate.go) for
more details.

## Composition Generation

> **Warning** Composition generation builds and executes code during runtime.
>
> This can be dangerous as it may impact the system on which the code is being
> executed.
>
> As a result, it is **not** recommended that these tools ever be installed on
> production or mission critical systems and instead, it is recommended that
> the tools be executed from inside a container environment with only the
> required directories mounted. See below for details.

Crossbuilder provides a toolkit that allows building compositions from Go and
write them out as YAML.

Both `Resource` type compositions and `Pipeline` compositions are
supported.

Since go is a statically typed language, Crossbuilder is able to perform
additional validation checks, such as patch path validation, that is a common
cause of errors when writing Crossplane compositions. This is only applicable
when running `xrc-gen` against a `Resource` type composition. It is not applicable
for running against a `Pipeline` mode composition as resources may come from
custom composition functions and templates making them ineligible for build-time
discovery.

Compositions are written as `go` plugins and must implement the `CompositionBuilder`
interface.

```golang
package main

import (
    "github.com/mproffitt/crossbuilder/pkg/generate/composition/build"
    ...
)

type builder struct {}

var Builder builder

func (b *builder) GetCompositeTypeRef() build.ObjectKindReference {
    // return object kind information
}

func (b *builder) Build(c build.CompositionSkeleton) {
    // implement pipeline here
}
```

The tool will look for all directories under `compositions` at the current
location with a max depth of 1 and builds a plugin from each discovered
directory, these are output to the `plugins` directory from where they will be
loaded.

### Running in a container

The recommended way to execute these tools is via a container.

Build the container with:

```bash
docker build . -t xrdtools
```

Execution of `xrd-gen` is via go generate:

```bash
docker run -it -v $(pwd)/examples:/tmp/crossbuilder:rw xrdtools go generate ./...
```

`xrc-gen` must be called directly

```bash
docker run -it -v $(pwd)/examples:/tmp/crossbuilder:rw xrdtools xrc-gen
```
