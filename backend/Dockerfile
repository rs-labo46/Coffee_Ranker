# ---------- build ----------
FROM golang:1.26.2 AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api .

# ---------- run ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=build /bin/api /api

EXPOSE 8080

ENTRYPOINT ["/api"]