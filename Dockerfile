FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/reelmate .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/reelmate /app/reelmate
COPY web /app/web
RUN mkdir -p /app/data && adduser -D -H appuser && chown -R appuser:appuser /app
USER appuser
ENV ADDR=:8080 DATA_FILE=/app/data/state.json
EXPOSE 8080
CMD ["/app/reelmate"]
