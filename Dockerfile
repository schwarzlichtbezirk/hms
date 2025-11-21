# Attention!
# Here expected that "bootstrap.sh" script was successfully executed,
# and project folder contains actual version of js-bundles and others files.

# Typical usage:
#   docker build --progress=plain --pull --rm -f "Dockerfile" -t schwarzlichtbezirk/hms:latest "."
#   docker run -v F:/Music:/mnt/music -d -p 80:80 schwarzlichtbezirk/hms

##
## Build stage
##

# Use image with golang last version as builder.
FROM golang:1.25-bookworm AS build

# Make project root folder as current dir.
WORKDIR /go/src/github.com/schwarzlichtbezirk/hms
# Copy only go.mod and go.sum to prevent downloads all dependencies again on any code changes.
COPY go.mod go.sum ./
# Download all dependencies pointed at go.mod file.
RUN go mod download
# Copy all files and subfolders in current state as is.
COPY . .

# Set executable rights to all shell-scripts.
RUN chmod +x ./task/*.sh
# Compile WPK-builder and build packages.
RUN ./task/make-builder.sh
# Build WPK-files.
RUN ./task/wpk-app.sh
RUN ./task/wpk-edge.sh
# Compile project for Linux amd64.
RUN ./task/build-docker.sh

##
## Deploy stage
##

# Thin deploy image.
FROM scratch

# Copy compiled executable and configuration files to new image destination.
COPY --from=build /go/bin /go/bin

# Open REST listen port.
EXPOSE 80 8804

# Run application with full path representation.
# Without shell to get signal for graceful shutdown.
ENTRYPOINT ["/go/bin/hms_linux_x64", "web"]
