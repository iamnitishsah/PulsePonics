# syntax=docker/dockerfile:1
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/simulator .

FROM alpine:3.22
COPY --from=build /out/simulator /usr/local/bin/simulator
USER 10001
ENTRYPOINT ["simulator"]
