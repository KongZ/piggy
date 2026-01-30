# Contributing to Piggy

We welcome contributions! Here's how you can help.

## Development Setup

1.  **Clone the repo**: `git clone https://github.com/KongZ/piggy.git`
2.  **Go version**: Ensure you are using Go 1.21+.
3.  **Local Build**:
    ```bash
    cd piggy-webhooks
    go build -o piggy-webhooks
    ```

## Testing

Always run tests before submitting a PR:

```bash
cd piggy-webhooks
go test -v ./...
```

To check coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Pull Request Process

1.  **Create a branch**: `git checkout -b feature/your-feature`
2.  **Linting**: We use `golangci-lint`. Please ensure your code passes local linting.
3.  **Tests**: Ensure all tests pass and coverage does not decrease.
4.  **Commits**: Use clear, descriptive commit messages.
5.  **Documentation**: Update relevant documentation (README, annotations, etc.) if you change functionality.
