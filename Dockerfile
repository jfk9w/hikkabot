FROM golang:1.22.5-alpine3.20 AS builder
WORKDIR /src
ADD . .
ARG VERSION=dev
RUN apk add --no-cache gcc musl-dev
RUN go build -buildvcs=false -ldflags "-X main.GitCommit=$VERSION" -o /app

FROM alpine:3.20
COPY --from=builder /app /usr/bin/app
RUN apk add --no-cache tzdata ffmpeg
ENTRYPOINT ["app"]
