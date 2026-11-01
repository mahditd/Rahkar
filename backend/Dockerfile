FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && CGO_ENABLED=0 go build -o /out/importer ./cmd/importer && CGO_ENABLED=0 go build -o /out/worker ./cmd/worker

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/importer /app/importer
COPY --from=build /out/worker /app/worker
EXPOSE 8080
ENTRYPOINT ["/app/api"]
