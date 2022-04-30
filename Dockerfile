FROM golang:1.18-alpine3.15 AS builder
WORKDIR /src
ADD . .
ARG VERSION=dev
RUN apk add --no-cache gcc
RUN go build -buildvcs=false -ldflags "-X main.GitCommit=$VERSION" -o /app

FROM alpine:3.15
COPY --from=builder /app /usr/bin/app
RUN apk add --no-cache tzdata ffmpeg
ENTRYPOINT ["app"]
