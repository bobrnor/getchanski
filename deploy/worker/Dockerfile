FROM alpine:3.7

RUN apk add --no-cache youtube-dl

COPY worker .

ENTRYPOINT ["worker"]