FROM golang:1.24.3-alpine as builder

RUN mkdir /app

COPY . /app

WORKDIR /app

RUN CGO_ENABLED=0 go build -o paymentApp ./cmd/main.go 

RUN chmod +x /app/paymentApp

# build tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app/paymentApp /app
COPY --from=builder /app/frontend ./frontend

CMD ["/app/paymentApp"]
