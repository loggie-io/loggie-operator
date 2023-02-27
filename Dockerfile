# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.18 as builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /workspace

# Copy the go source
COPY . .

# Build
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

# Run
FROM --platform=$BUILDPLATFORM debian:buster-slim

WORKDIR /
COPY --from=builder /workspace/loggie-operator  /usr/local/bin/
ENTRYPOINT ["loggie-operator"]