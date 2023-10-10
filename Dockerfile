FROM golang:1.21.1-alpine3.15 as builder

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build -o /supernats

FROM scratch
COPY --from=builder /supernats /
ENTRYPOINT [ "/supernats" ]
