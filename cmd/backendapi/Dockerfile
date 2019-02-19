# Base on your own image when changing shared code under /pkg/
# FROM $DOCKER_REGISTRY/openmatch-base:dev

# Use public golang image in other cases
FROM golang:1.11

WORKDIR /open-match/cmd/backendapi
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

FROM alpine:3.8
WORKDIR /open-match/cmd/backendapi
RUN ln -s ../../config/matchmaker_config.yaml
COPY --from=0 /open-match/cmd/backendapi/backendapi .

ENTRYPOINT ["./backendapi"]
