FROM golang:1.16.5-buster AS builder

# Install tools
RUN apt update && apt -y --no-install-recommends install \
    ca-certificates \
    git \
    tzdata

# Download modules
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .

# Build executable binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o demo .


FROM debian:buster-slim
RUN apt update && apt -y install curl

# Create appuser
ENV USER=appuser
ENV UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

USER appuser:appuser

COPY --from=builder /app/demo /demo

# Use an unprivileged user.
ENTRYPOINT ["/demo"]
EXPOSE 8080
