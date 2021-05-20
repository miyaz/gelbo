FROM golang:1.16 as builder

WORKDIR /go/src/work

COPY go.mod ./
RUN go mod download

COPY . .

ARG GOOS=linux
ARG GOARCH=amd64
#RUN go build -o /go/bin/gelbo -ldflags '-s -w'
RUN go build -o /go/bin/gelbo -ldflags '-s -w' main.go

FROM golang:1.16 as runner

COPY --from=builder /go/bin/gelbo /app/gelbo

ENTRYPOINT ["/app/gelbo"]
