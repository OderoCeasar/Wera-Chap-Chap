FROM golang:1.22-alpine

WORKDIR /app
RUN apk add --no-cache git
RUN go install github.com/cosmtrek/air@v1.49.0

COPY go.mod ./
RUN go mod download
COPY . .

EXPOSE 8080
CMD ["air", "-c", ".air.toml"]
