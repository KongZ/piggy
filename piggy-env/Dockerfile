FROM golang:1.16.5-buster as builder
ARG VERSION
ARG COMMIT_HASH
ARG BUILD_DATE
ARG LDFLAGS
ENV LDFLAGS="${LDFLAGS} -w -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}"

# Install tools
RUN apt update && apt -y --no-install-recommends install \
    ca-certificates \
    git \
    tzdata

# Download modules
WORKDIR /piggy-env
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy files
COPY . .

RUN go get github.com/securego/gosec/cmd/gosec
RUN go get honnef.co/go/tools/cmd/staticcheck
RUN CGO_ENABLED=0 go vet ./...
RUN CGO_ENABLED=0 staticcheck -f "stylish" ./...
RUN gosec -fmt=text ./...

# Build executable binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o /piggy-env/piggy-env -ldflags="$LDFLAGS" .

################################
# Main image
################################
FROM scratch
ARG VERSION
ARG COMMIT_HASH
ARG BUILD_DATE
LABEL VERSION=${VERSION}
LABEL COMMIT_HASH=${COMMIT_HASH}
LABEL BUILD_DATE=${BUILD_DATE}
LABEL org.opencontainers.image.source https://github.com/KongZ/piggy
ENV VERSION=${VERSION}

# Copy files from builder image
COPY --from=builder /piggy-env/piggy-env /piggy-env
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Use an unprivileged user. Don't use named user to avoid PSP error
USER 10001
ENTRYPOINT ["/piggy-env"]
