FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /portfolio-api .

FROM alpine:3.22
RUN adduser -D -H app && mkdir /data && chown app /data
COPY --from=build /portfolio-api /usr/local/bin/portfolio-api
USER app
ENV DATA_DIR=/data
VOLUME /data
EXPOSE 8001
ENTRYPOINT ["portfolio-api"]
