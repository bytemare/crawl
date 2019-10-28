# 1. Build exec
FROM golang:1.13.3 AS build-env

WORKDIR $GOPATH/src/github.com/bytemare/crawl/
COPY *.go ./
COPY .git ./
COPY app/ ./app/
RUN go get -d ./...
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /bin/crawl ./app/crawl.go

# 2. Build image
FROM gcr.io/distroless/static
COPY --from=build-env /bin/crawl /bin/crawl
USER nonroot
ENTRYPOINT ["/bin/crawl"]
