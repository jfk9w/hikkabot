FROM --platform=$BUILDPLATFORM golang:alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
ADD . .
ARG VERSION=dev
RUN apk add --no-cache make gcc musl-dev
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH VERSION=$VERSION make bin

FROM alpine:latest
COPY --from=builder /src/bin/* /usr/local/bin/
RUN apk add --no-cache tzdata ffmpeg
ENTRYPOINT ["hikkabot"]
