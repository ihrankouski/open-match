# Golang application builder steps
FROM golang:1.11 as builder
WORKDIR /go/src/github.com/ihrankouski/open-match/cmd/frontendapi
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

FROM alpine:3.8
WORKDIR /go/src/github.com/ihrankouski/open-match/cmd/frontendapi
RUN ln -s ../../config/matchmaker_config.yaml
COPY --from=builder /go/src/github.com/ihrankouski/open-match/cmd/frontendapi/frontendapi .

ENTRYPOINT ["./frontendapi"]
