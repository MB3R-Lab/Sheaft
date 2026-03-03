FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -o /out/sheaft ./cmd/sheaft

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/sheaft /usr/local/bin/sheaft
ENTRYPOINT ["sheaft"]

