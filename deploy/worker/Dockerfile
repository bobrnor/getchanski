FROM alpine:3.7

RUN apk add --no-cache youtube-dl ffmpeg

COPY worker .

ENTRYPOINT ["./worker"]