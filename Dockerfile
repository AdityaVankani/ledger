FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /expense-api ./cmd/api

FROM alpine:3.21
RUN adduser -D -H appuser
USER appuser
COPY --from=build /expense-api /expense-api
EXPOSE 8080
ENTRYPOINT ["/expense-api"]