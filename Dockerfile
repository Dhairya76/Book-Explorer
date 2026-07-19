# ---- build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /book-explorer .

# ---- run stage ----
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build /book-explorer /book-explorer
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/book-explorer"]
