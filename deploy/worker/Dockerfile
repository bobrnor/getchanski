FROM alpine:3.11

RUN apk add --no-cache youtube-dl ffmpeg

COPY worker .

ENTRYPOINT ["./worker"]