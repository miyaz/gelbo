FROM golang:1.16 as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
WORKDIR /go/src/work

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /go/bin/gelbo -ldflags '-s -w'

FROM alpine as runner

EXPOSE 80
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/gelbo /app/gelbo

ENTRYPOINT ["/app/gelbo"]
