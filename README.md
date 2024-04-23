# Crossbuilder

Crossbuilder is a tool that allows the generation of Crossplane XRDs and
compositions from go code.

This tool builds on [mistermx/crossbuilder] to provide two tools, `xrd-gen` and
`xrc-gen` which can be used to build crossplane definitions and compositions.

## XRD Generation

Crossbuilder's `xrd-gen` wraps around [kubebuilder] and [controller-gen] but
instead of generating CRDs (Custom Resource Definitions) it generates XRDs
(Composite Resource Definitions) that are part of the Crossplane ecosystem.

Crossbuilder has every feature that Kubebuilder's CRD generator provides plus
the ability to define XRD specific fields.

Take a look at the [api examples] for
more details.

## Composition Generation

> [!Warning]
> Composition generation builds and executes code during runtime.
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
when running `xrc-gen` against a `Resource` type composition. It is not
applicable for running against a `Pipeline` mode composition as resources may
come from custom composition functions and templates making them ineligible for
build-time discovery.

Compositions are written as `go` plugins and must implement the
`CompositionBuilder` interface.

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

### Structuring repositories

> [!Note]
> When structuring repositories intended to be used for compositions it should
> be noted that this function will copy `go.mod` and `go.sum` into the repo at
> runtime. This is to ensure that the plugins are built with the same version
> information that the `xrc-gen` binary is compiled with.
>
> Whilst these files are relevant for building locally, they should be added to
> `.gitignore` or added to the tree in such a way that they will not be shared
> with the container.

Whilst this is opinionated, when structuring your git repository for use with
these tools, the `compositions` directory must exist at the root of the mounted
directory when running inside a container.

`xrc-gen` will write `package/compositions` at this location and this is
currently immutable.

Each composition folder can contain exactly one composition although multiple
compositions may exist for the same XRD.

```nohighlight
.
├── apis
│   ├── generate.go
│   └── v1alpha1
│       ├── doc.go
│       ├── example_types.go
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── compositions
│   ├── example
│   │   └── example.go
│   └── pipelineexample
│       ├── pipeline.go
│       └── templates
│           ├── 01.tpl
│           └── 02.tpl
├── hack
│   └── boilerplate.go.txt
└── package
    ├── compositions
    │   ├── example.yaml
    │   └── pipelineexample.yaml
    └── xrds
        └── test.example.com_xexamples.yaml
```

Given this current limitation, the above structure is recommended at the root
of the mounted directory where:

- `apis` contains the XRD specification
- `compositions` contains the compositions relevant for the defined api
- `hack` contains license header to inject into the generated API files (e.g.
  those starting with `zz_generated`)
- `package` contains automatically generated yaml

When working with compositions, you may often want multiple compositions but only
a single XRD. They may be different varients on the same provider, or for
different cloud providers.

With the countainer mount limitation in mind, if you required multiple XRDs in a
single directory (e.g. a monolithic repo), then abstract one layer above to
become the API group:

For example:

```nohighlight
.
├── my.custom.composition.io
│   ├── apis
│   ├── compositions
│   ├── ...
├── a.different.custom.composition.io
│   ├── apis
│   ├── compositions
│   ├── ...
...
```

you can then mount at `my.custom.composition.io` and the structure will be
recognisable to the tooling.

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

[mistermx/crossbuilder]: https://github.com/mproffitt/crossbuilder
[kubebuilder]: https://github.com/kubernetes-sigs/kubebuilder
[controller-gen]: https://github.com/kubernetes-sigs/controller-tools
[api examples]: ./examples/apis/v1alpha1
