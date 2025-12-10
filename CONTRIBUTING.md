# Contributing to AgentMCP

Thank you for your interest in contributing to AgentMCP! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, constructive, and professional. We're building a tool for the community.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/aminghadersohi/agentmcp/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)
   - Relevant logs or error messages

### Suggesting Features

1. Check existing [Issues](https://github.com/aminghadersohi/agentmcp/issues) and [Discussions](https://github.com/aminghadersohi/agentmcp/discussions)
2. Create a new issue with:
   - Clear use case
   - Proposed solution
   - Alternative approaches considered
   - Willingness to implement

### Pull Requests

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR-USERNAME/agentmcp.git
   cd agentmcp
   ```

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Write clean, readable code
   - Follow existing code style
   - Add tests for new functionality
   - Update documentation as needed

4. **Test Your Changes**
   ```bash
   go test -v ./...
   go build -o agentmcp .
   ./agentmcp -agents ./agents -transport stdio
   ```

5. **Commit**
   ```bash
   git add .
   git commit -m "Add feature: description"
   ```

   Commit message format:
   - `Add: new feature`
   - `Fix: bug description`
   - `Update: what was updated`
   - `Refactor: what was refactored`
   - `Docs: documentation changes`

6. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```
   Then create a Pull Request on GitHub.

## Development Setup

### Prerequisites

- Go 1.22 or later
- Git
- (Optional) Docker for testing containers

### Building from Source

```bash
git clone https://github.com/aminghadersohi/agentmcp.git
cd agentmcp
go mod download
go build -o agentmcp .
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -v -run TestLoadAgents
```

### Code Style

- Use `gofmt` for formatting: `go fmt ./...`
- Use `go vet` for static analysis: `go vet ./...`
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Keep functions small and focused
- Write clear comments for exported functions
- Use meaningful variable names

### Project Philosophy

**Keep It Simple**:
- Prefer simple over clever
- Avoid premature optimization
- No unnecessary dependencies
- Code should be easy to read and maintain

**Do One Thing Well**:
- Focus on serving agent definitions
- Don't add features that belong in clients
- Resist feature creep

## Testing

### Writing Tests

- Add tests for all new functionality
- Test edge cases and error conditions
- Use table-driven tests where appropriate
- Keep tests fast and independent

Example:
```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := NewFeature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Documentation

- Update README.md for user-facing changes
- Add code comments for complex logic
- Update CHANGELOG.md (if exists)
- Add examples for new features

## Release Process

Releases are automated via GitHub Actions:

1. Update version in code if needed
2. Create and push a tag:
   ```bash
   git tag -a v1.1.0 -m "Release v1.1.0"
   git push origin v1.1.0
   ```
3. GitHub Actions will build and publish the release

## Questions?

- Open a [Discussion](https://github.com/aminghadersohi/agentmcp/discussions)
- Ask in issues
- Check existing documentation

## Recognition

Contributors will be recognized in:
- GitHub contributors page
- Release notes (for significant contributions)

Thank you for contributing to AgentMCP!
