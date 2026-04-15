# ---- BUILD STAGE -----
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app ./cmd/api/main.go


# ---- RUN STAGE -----
FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 8080

CMD [ "./app" ]
