# Attention!
# Here expected that "bootstrap.sh" script was successfully executed,
# and project folder contains actual version of js-bundles and others files.

# Typical usage:
#   docker build --progress=plain --pull --rm -f "Dockerfile" -t hms:latest "."
#   docker run -v F:/Music:/mnt/music -d -p 80:80 hms

##
## Build stage
##

# Use image with golang last version as builder.
FROM golang:1.19-bullseye AS build

# See https://stackoverflow.com/questions/64462922/docker-multi-stage-build-go-image-x509-certificate-signed-by-unknown-authorit
RUN apt-get update && apt-get install -y ca-certificates openssl
ARG cert_location=/usr/local/share/ca-certificates
# Get certificate from "github.com".
RUN openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${cert_location}/github.crt
# Get certificate from "proxy.golang.org".
RUN openssl s_client -showcerts -connect proxy.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM >  ${cert_location}/proxy.golang.crt
# Update certificates.
RUN update-ca-certificates

# Make project root folder as current dir.
WORKDIR $GOPATH/src/github.com/schwarzlichtbezirk/hms
# Copy only go.mod and go.sum to prevent downloads all dependencies again on any code changes.
COPY go.* .
# Download all dependencies pointed at go.mod file.
RUN go mod download
# Copy all files and subfolders in current state as is.
COPY . .

# Set executable rights to all shell-scripts.
RUN chmod +x task/*.sh

# Compile WPK-builder and build packages.
RUN task/make-builder.sh

# Build WPK-files.
RUN task/wpk-app.sh
RUN task/wpk-edge.sh

# Compile project for Linux amd64.
RUN task/build-linux.x64.sh

##
## Deploy stage
##

# Thin deploy image.
FROM gcr.io/distroless/base-debian11

# Copy compiled executable and packages to new image destination.
COPY --from=build /go/bin/hms* /go/bin/
# Copy configuration files.
COPY --from=build /go/bin/config/* /go/bin/config/

# Open REST listen port.
EXPOSE 80 8080

# Run application with full path representation.
# Without shell to get signal for graceful shutdown.
ENTRYPOINT ["/go/bin/hms.linux.x64.exe"]
