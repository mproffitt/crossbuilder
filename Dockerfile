# syntax=docker/dockerfile:1
ARG GO_VERSION=1
ARG TARGETOS=linux
ARG TARGETARCH=amd64

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS image
RUN mkdir /.cache
RUN chmod a+rwx -R /.cache


ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

WORKDIR /crossbuilder

COPY pkg pkg
COPY cmd cmd
COPY scripts scripts
COPY scripts/init /usr/local/bin/init
RUN chmod +x /usr/local/bin/init
RUN chmod a+rwx -R /crossbuilder

USER 1000:1000
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

ENTRYPOINT [ "/usr/local/bin/init" ]
WORKDIR /build