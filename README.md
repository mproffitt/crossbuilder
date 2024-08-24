# Crossbuilder

Crossbuilder is a tool that allows the generation of Crossplane XRDs and
compositions from go code.

This tool builds on [mistermx/crossbuilder] to provide two tools, `xrd-gen` and
`xrc-gen` which can be used to build crossplane definitions and compositions.

## Usage

The easiest way to use this build tool is to include it as a submodule to your
project at the `crossbuilder` path and then execute `./crossbuilder/scripts/gen`

```bash
git submodule add https://github.com/mproffitt/crossbuilder
git submodule init
./crossbuilder/scripts/gen
```

Alternatively you may use the docker container which will set this up for you

```bash
docker run -v $(pwd):build docker.io/choclab/xrdtools:latest
```

## XRD Generation

Crossbuilder's `xrd-gen` wraps around [kubebuilder] and [controller-gen] but
instead of generating CRDs (Custom Resource Definitions) it generates XRDs
(Composite Resource Definitions) that are part of the Crossplane ecosystem.

Crossbuilder has every feature that Kubebuilder's CRD generator provides plus
the ability to define XRD specific fields.

Take a look at the [api examples] for
more details.

## Composition Generation

Crossbuilder provides a toolkit that allows building compositions from Go and
write them out as YAML.

Both `Resource` type compositions and `Pipeline` compositions are
supported.

Since go is a statically typed language, Crossbuilder is able to perform
additional validation checks, such as patch path validation, that is a common
cause of errors when writing Crossplane compositions. This is only applicable
when running `xrc-gen` against a `Resource` type composition. It is currently not
applicable for running against a `Pipeline` mode composition as resources may
come from custom composition functions and templates making them ineligible for
build-time discovery.

Compositions are written as `go` plugins and must implement the
`CompositionBuilder` interface as well as exposing a `TemplateBasePath` string
variable which is injected during the runtime process and may be passed to
the build module when templates are used.

```golang
package main

import (
    "github.com/mproffitt/crossbuilder/pkg/generate/composition/build"
    ...
)

type builder struct {}

var Builder builder
var TemplateBasePath string

func (b *builder) GetCompositeTypeRef() build.ObjectKindReference {
    // return object kind information
}

func (b *builder) Build(c build.CompositionSkeleton) {
    build.TemplateBasePath = TemplateBasePath
    // implement pipeline here
}
```

`xrc-gen` operates by first running `xrd-gen` on all directories under the
repository root which contain a `generate.go` file.

Once this has completed, it then attempts to detect any folder which contains
a `main.go` file, ignoring anything found under `pkg`, `internal` or `crossbuilder`

These locations are then compiled into the `plugins` folder which will be created
if it does not exist.

Once all plugins are compiled, they are loaded and executed to generate
composition manifests which are stored under `apis/<group-prefix>` - for example
if your composition has the type `xexample.crossplane.example.io` it will be
output to `apis/xexample/composition_name.yaml`

### Structuring repositories

The current version of the tool attempts to remove most restrictions on
repository structure, other than composition locations must contain a `main.go`
file, and the API must contain a `generate.go` file:

```nohighlight
.
├── compositions
│   ├── example
│   │   ├── main.go
│   │   └── templates
│   │       ├── template1.k
│   │       └── template2.k
│   └── anotherexample
│       └── main.go
├── generate.go
└── v1alpha1
    ├── doc.go
    ├── groupversion.go
    ├── example_types.go
    ├── anotherexample_types.go
    └── zz_generated.deepcopy.go
```
