FROM golang:1.24.2

WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o api-service ./cmd/main.go

EXPOSE 8080
CMD ["./api-service"]