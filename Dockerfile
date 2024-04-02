FROM golang:1.22.1-alpine3.19 AS builder

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /dcd cmd/dcd/main.go

FROM scratch

COPY --from=builder /dcd /dcd

ENTRYPOINT ["/dcd", "image-usage-message"]
