FROM golang:1.21.1-alpine3.17 as builder

WORKDIR /src
COPY go.mod .
COPY go.sum .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build -o /supernats

FROM scratch
COPY --from=builder /supernats /
ENTRYPOINT [ "/supernats" ]
