FROM golang:1.23.4-alpine AS build

WORKDIR /usr/src/app

COPY go.mod ./
COPY main.go ./

RUN go build -ldflags="-s -w" -o /usr/local/bin/app main.go

FROM golang:1.23.4-alpine

COPY --from=build /usr/local/bin/app /app
COPY index.html ./
COPY states.html ./

CMD ["/app"]
