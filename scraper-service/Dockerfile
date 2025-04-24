FROM golang:1.24.2

WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o scraper-service ./cmd/main.go

CMD ["./scraper-service"]