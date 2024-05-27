FROM golang:1.22.3 as builder

ADD . /go/src/myapp
WORKDIR /go/src/myapp

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /go/src/myapp/app .

EXPOSE 80

CMD ["./app"]
