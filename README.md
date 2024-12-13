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

Crossbuilder provides a toolkit that allows building compositions from Go and
write them out as YAML.

> [!Important]
> Whilst both `Resource` type compositions and `Pipeline` compositions are
> supported, no future development will be made on `Resource` mode pipelines
> within this fork, only where they are merged in from upstream.

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
a `main.go` file, ignoring anything found under `pkg`, `internal` or
`crossbuilder` subdirectories.

Discovered locations are then compiled into the `plugins` folder which will be
created if it does not exist.

Once all plugins are compiled, they are loaded and executed to generate
composition manifests which are stored under `apis/<group-prefix>` - for example
if your composition has the type `xexample.crossplane.example.io` it will be
output to `apis/xexample/composition_name.yaml`

> [!Note]
> The first time you run crossbuilder over a new repository, it may take a long
> time to run. This is also true if you are running inside a docker container
> as it takes time to cache all the dependencies required by both crossbuilder
> and your compositions.

## Getting Started

The easiest way to get started with `crossbuilder` is to create a new, empty
folder and run the following command:

```bash
curl -sq https://raw.githubusercontent.com/mproffitt/crossbuilder/refs/heads/main/setup.sh | bash
```

This command will download the script [setup.sh](./setup.sh) and execute it,
prompting you for input as required.

On first setup you will be asked:

- If you want to create a new git repository. This is only asked if there is not
  already a `.git` folder in the current directory or any of its parents.
  Valid answers: `yes` and `no`
- If you are initialising a new repository, you are asked for the remote URL.
  This is normally the `ssh` URL used for interacting with the repo, however
  you may use any valid git URL.
- The API extension to use. For example `crossplane.example.com` This address
  gets baked into the [templates/create.sh](./setup.sh) script.

Crossbuilder will then

- Add itself as a submodule to the current git repository
- Copy the [`template`](./template/) folder to the root of the current repo
- Copy the `setup.sh` script and modify it to set the base path
- Copy [`template/files/Makefile`](./template/files/Makefile) to the root of the
  current repo
- Copy [`template/files/Dockerfile`](./template/files/Dockerfile) to the root of
  the current repo

Once these steps have been completed, the setup script will then trigger
`make create` to guide you through setting up a new API.

### Creating a new API

```bash
make create
```

When adding a new API to your repository, the `make create` invocation can help
guide you through this process.

`make create` will ask you the following questions to help with the process

- The name of the group to add the API to. (example `xtest`)
- The name of the composition to create. This should be a lowercase, hyphenated
  string. For example `my-first-composition`
- The group class to which this composition belongs. This name is perhaps a
  little misleading as this becomes the name of your XRD Kind. This should be a
  camel cased string, for example `TestComposition`
- The Shortname to give to your XRD. This is normally 2 or 3 characters long,
  for example `tc` for `TestComposition`
- If the composition should be enforced. Valid answers are `yes` and `no`.
  If you enforce the composition, then this is the only composition that can
  be used by this XRD. This is fine if you're only creating a single purpose
  composition (for example AWS RDS), but if you then want to use that XRD for
  Azure or GCP, you would have to change the enforcement by modifying the
  kubebuilder comments on top of the main XRD root struct.

These steps will create a few directories if they don't already exist, and also
create your stub composition for you.

> [!Note]
> This command will create a stub composition in Pipeline mode that includes
> `function-auto-ready` for you.

### Working with `crossplane` packages

```bash
CONTAINER_REGISTRY=ghcr.io/example make crossplane
```

Once you've written your composition and are ready to test it, you will need to
compile it. `make build` will do this for you and write its output to the `apis`
folder under the group name you chose for this composition (or set of
compositions).

In this state, it's not very useful to you as it still needs to be applied to
the cluster so you'll need to create a `crossplane.yaml` file in that folder
to support packaging.

> [!Note]
> At present the `crossplane.yaml` file is not auto-generated for you. A future
> iteration of this project may support doing so.

If a `crossplane.yaml` file is found in the APIs group folder, there are helper
commands to build and package these for you, only working with APIs that have
changed since the last tag, or root of the repo if there are no tags.

Lets say you have your group `xtest`. This is compiled into `apis/xtest` folder.

Inside this folder, create a new file `crossplane.yaml` with at the very least
the following configuration.

```yaml
---
apiVersion: meta.pkg.crossplane.io/v1alpha1
kind: Configuration
metadata:
  name: xtest
spec:
  crossplane:
    version: ">=v1.17.0"
  dependsOn:
  - function: xpkg.upbound.io/crossplane-contrib/function-auto-ready
    version: ">=0.3.0"
```

Add any new providers, functions and/or configurations you require to the
`dependsOn` list.

Once this file is assembled, running `make crossplane` will help you by
re-compiling your compositions, packaging them and pushing them to the OCI
compliant artifact repository you specify.

### Working with KCL

```bash
CONTAINER_REGISTRY=ghcr.io/example make kcl
```

I have composed the Makefile to be able to package and push KCL OCI containers
as this is one of my primary languages for working with crossplane.

To add new KCL modules to your composition, create a new folder under (example)

```plaintext
crossplane.example.com
├── xtest
│   ├── compositions
│   │    ├── my-composition
│   │    │   ├── modules
│   │    │   │   ├── my-cool-kcl-module
```

Change to this location and run `kcl mod init` to set up the module.

KCL packaging works in a similar way to the crossplane packaging in that it will
detect the existance of KCL modules by searching the repo for `kcl.mod` files
that have changed since the last tag, or root commit if there are no tags on the
repo.

This step is a little more involved as KCL modules may have dependencies between
them that need to have the versions packaged in to the module before they can be
pushed.

If your module contains a dependency to another module in the repo, and that
module has also changed then the Makefile will attempt to update the `kcl.mod`
to set the version for both modules (the current, and the dependency), and will
also update the lock file for the current module.

> [!Caution]
> Only modules that have changed since the last tag are affected by this change.
> If you have older modules or compositions in the repository that have not
> been touched, they are unaffected by versioning.
>
> References to your KCL module inside `main.go` or any other `go` files used
> to support your composition are also unaffected by the versioning.
>
> This is deliberate. It is your responsibility to set the correct version of
> the KCL module required by your composition.

Once packaged, each module will be pushed to the container registry where it can
then be used by your composition.

To include a KCL function in your pipeline which uses an OCI repository, add the
following to your `main.go` file:

```golang
c.NewPipelineStep("step-kcl-do-something").
    WithFunctionRef(xapiextv1.FunctionReference{
        Name: "function-kcl",
    }).
    WithInput(build.ObjectKindReference{
        Object: &xkcl.KCLInput{
            TypeMeta: metav1.TypeMeta{
                APIVersion: "krm.kcl.dev/v1alpha1",
                Kind:       "KCLInput",
            },
            Spec: xkcl.RunSpec{
                Source: "oci://ghcr.io/example/my-cool-kcl-module:0.0.1",
            },
        },
    })
```

> [!Important]
> KCL module dependency management is done via a third party TOML command
> <https://github.com/diversable/toml-editor-cli>. This is a rust based toml CLI
> command and was the only decent compiled one I could find.
>
> This CLI is only released for `linux` platforms. To use it on any other
> platform will require you to build it yourself.
>
> It is included in the Dockerfile placed at the root of your repo and can be
> executed with:
>
> ```bash
> docker build . -t crossbuilder
> docker run -it crossbuilder toml
> ```

### Other makefile targets

There are two other `Makefile` targets of interest:

```bash
CONTAINER_REGISTRY=ghcr.io/example make all
```

This will build all compositions, package and push all crossplane and package
and push all KCL that has changed since the last tag.

```bash
CONTAINER_REGISTRY=ghcr.io/example make indocker
```

Exactly the same as `make all` but builds a docker container and runs everything
inside that.

> [!Note]
> `make indocker` mounts the local directory, and the `~/.docker` directory
> so it can access the auth keys stored in `~/.docker/config.json` to be able to
> push the crossplane and kcl modules to remote artifact store.
>
> It is not currently possible to overwride the docker location as part of the
> `Makefile`.

### Structuring repositories

Each composition must be structured in such a way that crossbuilder can detect
the files it needs to operate.

Each composition folder must contain a `generate.go` file, and each composition
a `main.go` file.

In order to use the boilerplate generation, a few guidelines are in place but
these are not strictly enforced by the tool.

- It is RECOMMENDED that you name the composition folder the same as the base
  domain for your compositions. Compositions will sit under this and it makes
  readable sense to group them together in this way. For example if your base
  domain is `example.com` then your uncompiled APIs may sit under the folder
  `crossplane.example.com`

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
