FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/pokedex ./cmd/pokedex

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/pokedex /pokedex
EXPOSE 8080
ENTRYPOINT ["/pokedex"]
