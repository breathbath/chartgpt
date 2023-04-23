FROM golang:1.20 AS build-env

RUN mkdir -p /build/chartgpt

WORKDIR /build/chartgpt
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

ARG version
RUN echo "Version $version"
# Build
RUN go build -a -installsuffix cgo \
        -o /build/chartgpt/chartgpt \
        -ldflags \
        "-X breathbathChartGPT/pkg/cmd.Version=$version" \
        main.go

# -------------
# Image creation stage

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app
USER app:app

COPY --from=build-env /build/chartgpt/chartgpt /app/
COPY .env.default /app/.env.default
COPY .env /app/.env
CMD /app/chartgpt telegram
