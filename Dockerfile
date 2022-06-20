FROM golang:alpine as build

MAINTAINER Alexander Lindner <25225552+alexander-lindner@users.noreply.github.com>
LABEL org.opencontainers.image.title="go fileshare"
LABEL org.opencontainers.image.version="0.1.0"
LABEL org.opencontainers.image.licenses="MPL-2.0"
LABEL org.opencontainers.image.url="https://github.com/alexander-lindner/go-fileshare"
LABEL org.opencontainers.image.description="A quick and dirty image for sharing files with a go webserver."

WORKDIR /src
RUN apk add --no-cache git

COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -installsuffix 'static' -o /app .

FROM gcr.io/distroless/static AS final

USER nonroot:nonroot
COPY --from=build --chown=nonroot:nonroot /app /app
WORKDIR /workdir
ENTRYPOINT ["/app"]