FROM golang:latest as builder
ARG VERSION=dev

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/krzysztof-gzocha/prometheus2mqtt/config.Version=$VERSION" -o prometheus2mqtt .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/prometheus2mqtt .
CMD ["./prometheus2mqtt"]
