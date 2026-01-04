# Contributing to technitium-companion

Thank you for your interest in contributing.

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Create a feature branch from `main`

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/technitium-companion.git
cd technitium-companion

# Install dependencies
go mod download

# Run tests
go test -v ./...

# Build locally
go build -o technitium-companion ./cmd/technitium-companion
```

## Making Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and add tests where appropriate

3. Run tests and linting:
   ```bash
   go test -v ./...
   go vet ./...
   ```

4. Commit with a descriptive message:
   ```bash
   git commit -m "feat: add support for X"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/) format:
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation only
   - `refactor:` code change that neither fixes a bug nor adds a feature
   - `test:` adding or updating tests
   - `chore:` maintenance tasks

5. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

6. Open a Pull Request against the `main` branch

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions and types
- Keep functions focused and reasonably sized

## Testing

- Add unit tests for new functionality
- Ensure existing tests pass before submitting
- Test with both standalone Docker and Swarm modes if applicable

## Questions

If you have questions, open an issue for discussion before starting work on large changes.
