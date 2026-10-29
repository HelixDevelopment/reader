FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY core ./core
COPY server ./server
WORKDIR /src/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/helix-server ./cmd/helix-server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates rclone && addgroup -S helix && adduser -S -G helix helix
COPY --from=build /out/helix-server /usr/local/bin/helix-server
USER helix
ENV HELIX_ADDRESS=0.0.0.0:8080 HELIX_COVER_CACHE=/data/covers
VOLUME ["/data"]
EXPOSE 8080/tcp 8080/udp
ENTRYPOINT ["/usr/local/bin/helix-server"]
