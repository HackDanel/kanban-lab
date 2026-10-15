FROM golang:1.27-alpine AS build
WORKDIR /src
# proxy.golang.org is unreliable from some networks; goproxy.io is the fallback.
ARG GOPROXY=https://proxy.golang.org|https://goproxy.io|direct
ENV GOPROXY=$GOPROXY
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
USER nonroot:nonroot
ENV HTTP_ADDR=:8080 UPLOAD_DIR=/data/uploads
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 CMD ["/server", "healthcheck"]
ENTRYPOINT ["/server"]
