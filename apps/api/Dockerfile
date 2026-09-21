# --- builder ---
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Dependency layer cached separately from source so a code-only change
# doesn't re-download modules.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/migrate ./cmd/migrate

# --- final ---
FROM gcr.io/distroless/static-debian12:nonroot
USER nonroot:nonroot

COPY --from=builder /app/api /app/api
COPY --from=builder /app/migrate /app/migrate

EXPOSE 8088
ENV HTTP_ADDR=:8088
ENV APP_ENV=production

CMD ["/app/api"]
