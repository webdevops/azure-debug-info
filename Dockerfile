FROM golang:1.17 as build

WORKDIR /go/src/github.com/webdevops/azure-debug-info

# Get deps (cached)
COPY ./go.mod /go/src/github.com/webdevops/azure-debug-info
COPY ./go.sum /go/src/github.com/webdevops/azure-debug-info
COPY ./Makefile /go/src/github.com/webdevops/azure-debug-info
RUN make dependencies

# Compile
COPY ./ /go/src/github.com/webdevops/azure-debug-info
RUN make test
RUN make lint
RUN make build
RUN ./azure-debug-info --help

#############################################
# FINAL IMAGE
#############################################
FROM gcr.io/distroless/static
ENV LOG_JSON=1
COPY --from=build /go/src/github.com/webdevops/azure-debug-info/azure-debug-info /
USER 1000:1000
ENTRYPOINT ["/azure-debug-info"]
