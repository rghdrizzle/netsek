FROM golang:alpine3.19 as build

WORKDIR /app

COPY go.mod .

RUN go mod download

COPY . .

RUN go build -o netsek

FROM scratch as prod

COPY --from=build /app .

CMD ["./netsek"]