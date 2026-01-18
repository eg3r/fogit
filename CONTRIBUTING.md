# Contributing to FoGit

Thank you for your interest in contributing to FoGit! This document provides guidelines and information for contributors.

## Code of Conduct

Be respectful and constructive. We welcome contributors of all experience levels.

## Getting Started

### Prerequisites

- Go 1.23 or later
- Git
- golangci-lint (for linting)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/eg3r/fogit.git
cd fogit

# Initialize the specification submodule
git submodule update --init --recursive

# Install dependencies
go mod download

# Build
go build -o fogit .

# Run tests
go test ./...

# Run linter
golangci-lint run
```

## How to Contribute

### Reporting Issues

Before creating an issue:
1. Search existing issues to avoid duplicates
2. Use a clear, descriptive title
3. Include steps to reproduce (for bugs)
4. Include Go version and OS

### Submitting Changes

1. **Fork** the repository
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** following our coding standards
4. **Write tests** for new functionality
5. **Run tests and linter**:
   ```bash
   go test ./...
   golangci-lint run
   ```
6. **Commit** with a clear message
7. **Push** and create a Pull Request

### Pull Request Guidelines

- Reference any related issues
- Include a clear description of changes
- Ensure all tests pass
- Keep PRs focused (one feature/fix per PR)
- Update documentation if needed

## Development Guidelines

### Project Structure

```
fogit/
├── main.go              # Entry point
├── commands/            # CLI commands (Cobra)
├── pkg/fogit/           # Public API & domain models
├── internal/            # Private implementation
│   ├── storage/         # YAML file operations
│   ├── git/             # Git integration
│   ├── features/        # Feature business logic
│   └── ...
└── spec/                # Specification (git submodule)
```

### Coding Standards

- **Formatting**: Use `gofmt` (enforced by CI)
- **Linting**: All code must pass `golangci-lint`
- **Naming**: Follow Go conventions
  - `PascalCase` for exported identifiers
  - `camelCase` for unexported identifiers
- **Error messages**: Lowercase, no trailing punctuation
- **Comments**: Complete sentences with punctuation for exported items

### Testing Requirements

- **Write tests first** (TDD approach)
- **Table-driven tests** are preferred:
  ```go
  func TestFeature_Validate(t *testing.T) {
      tests := []struct {
          name    string
          setup   func() *Feature
          wantErr bool
      }{
          {
              name: "valid feature",
              setup: func() *Feature {
                  return NewFeature("Test")
              },
              wantErr: false,
          },
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              f := tt.setup()
              err := f.Validate()
              if (err != nil) != tt.wantErr {
                  t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
              }
          })
      }
  }
  ```
- **Coverage target**: 80%+ for new code
- **Integration tests**: Use `t.TempDir()` for file system tests

### Specification

The project follows a formal specification in the `spec/` submodule:

- **CLI commands**: `spec/specification/08-interface.md`
- **Data model**: `spec/specification/06-data-model.md`
- **Concepts**: `spec/specification/02-concepts.md`

Always check the spec before implementing new features.

## Commit Messages

Use clear, descriptive commit messages:

```
<type>: <short description>

<optional longer description>

<optional footer with issue references>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

Example:
```
feat: add version constraints to relationships

Implements version constraint syntax (>=1.0, <2.0) for relationship
definitions as specified in 06-data-model.md.

Closes #42
```

## Getting Help

- Check the [specification](spec/) for requirements
- Open an issue for questions or discussions

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
