# Build stage
FROM golang:1.27-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION}" -o /pando .

# Run stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /pando /pando
VOLUME /backup
ENV BACKUP_FOLDER_PATH=/backup
EXPOSE 8080
ENTRYPOINT ["/pando"]