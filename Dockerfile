# syntax=docker/dockerfile:1

# We use the latest Go 1.x version unless asked to use something else.
# The GitHub Actions CI job sets this argument for a consistent Go version.
ARG GO_VERSION=1

# Setup the base environment. The BUILDPLATFORM is set automatically by Docker.
# The --platform=${BUILDPLATFORM} flag tells Docker to build the function using
# the OS and architecture of the host running the build, not the OS and
# architecture that we're building the function for.
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS build

WORKDIR /build

# Plugins require CGO so we enable it.
ENV CGO_ENABLED=1

# We run go mod download in a separate step so that we can cache its results.
# This lets us avoid re-downloading modules if we don't need to. The type=target
# mount tells Docker to mount the current directory read-only in the WORKDIR.
# The type=cache mount tells Docker to cache the Go modules cache across builds.
# RUN --mount=target=. --mount=type=cache,target=/go/pkg/mod go mod download

# The TARGETOS and TARGETARCH args are set by docker. We set GOOS and GOARCH to
# these values to ask Go to compile a binary for these architectures. If
# TARGETOS and TARGETOS are different from BUILDPLATFORM, Go will cross compile
# for us (e.g. compile a linux/amd64 binary on a linux/arm64 build machine).
ARG TARGETOS
ARG TARGETARCH
COPY . /build/

# In order for the plugins to build, they need exactly the same
# common code as the main binary.
# To achieve this, when building, we replace the module path in go.mod to
# "crossbuilder" instead of the actual module path. This forces a dependency
# on github.com/mproffitt/crossbuilder instead of the local path and ensures
# that the versions remain consistent.
RUN sed -i 's#github.com/mproffitt/crossbuilder#crossbuilder#g' go.mod
RUN sed -i 'x;/./{x;b};x;/require/h;//a\\tgithub.com/mproffitt/crossbuilder v0.0.1-dev7' go.mod
RUN go get crossbuilder/cmd/xrd-gen
RUN go mod download

# Build the builder binaries. The type=target mount tells Docker to mount the
# current directory read-only in the WORKDIR. The type=cache mount tells Docker
# to cache the Go modules cache across builds.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /crossbuilder/xrd-gen ./cmd/xrd-gen

# Produce a new build image using the same go version as the build image. This
# image will contain the builder binaries in /usr/local/bin.
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS image
WORKDIR /crossbuilder
COPY --from=build /crossbuilder /usr/local/bin
COPY --from=build /build/go.mod /build/go.sum /crossbuilder/
RUN mkdir -p cmd/xrc-gen
COPY pkg pkg

COPY cmd/xrc-gen cmd/xrc-gen
COPY scripts/gen /usr/local/bin/gen
RUN chmod +x /usr/local/bin/gen
ENTRYPOINT [ "/usr/local/bin/gen" ]
RUN mkdir /.cache
RUN chmod a+rwx -R /.cache
RUN chmod a+rwx -R /crossbuilder
ENV GOOS=linux
ENV GOARCH=amd64
USER 1000:1000
