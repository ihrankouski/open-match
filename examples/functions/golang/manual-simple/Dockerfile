# Golang application builder steps
FROM gcr.io/open-match-public-images/openmatch-base:dev as builder
WORKDIR /go/src/github.com/ihrankouski/open-match/examples/functions/golang/manual-simple
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mmf .

#FROM scratch
#COPY --from=builder /go/src/github.com/ihrankouski/mmfstub/mmfstub mmfstub
CMD ["./mmf"]
