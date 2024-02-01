FROM golang:1.20 AS build-env

RUN mkdir -p /build/chatgpt

WORKDIR /build/chatgpt
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

ARG version
RUN echo "Version $version"
# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
        -o /build/chatgpt/chatgpt \
        -ldflags \
        "-X breathbathChatGPT/pkg/cmd.Version=$version" \
        main.go

# -------------
# Image creation stage

FROM alpine:latest
RUN apk --no-cache add ca-certificates ffmpeg

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app
USER app:app

COPY --from=build-env /build/chatgpt/chatgpt /app/
COPY .env.default /app/.env.default
COPY source.ogg /app/source.ogg
CMD /app/chatgpt telegram
