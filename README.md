# FoGit

[![CI](https://github.com/eg3r/fogit/actions/workflows/ci.yml/badge.svg)](https://github.com/eg3r/fogit/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/eg3r/fogit)](https://goreportcard.com/report/github.com/eg3r/fogit)
[![Go Reference](https://pkg.go.dev/badge/github.com/eg3r/fogit.svg)](https://pkg.go.dev/github.com/eg3r/fogit)
[![GitHub Release](https://img.shields.io/github/v/release/eg3r/fogit)](https://github.com/eg3r/fogit/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**FoGit (Feature-Oriented Git)** - A Git-native feature tracking system.

## Overview

FoGit tracks features, requirements, and design elements as first-class entities alongside your code. While Git tracks files and lines, FoGit tracks features and relationships.

## Installation

```bash
# Using Go (recommended - easiest updates)
go install github.com/eg3r/fogit@latest

# Update to latest version
fogit self-update
```

### Windows Installer

Download the MSI from [releases](https://github.com/eg3r/fogit/releases/latest).

**Interactive install:** Double-click the MSI and choose per-user or per-machine.

**Silent install (command line):**
```powershell
# Per-user install (no admin required, recommended)
msiexec /i fogit_windows_amd64_setup.msi /quiet

# Per-machine install (requires admin)
msiexec /i fogit_windows_amd64_setup.msi /quiet ALLUSERS=1

# Silent uninstall
msiexec /x fogit_windows_amd64_setup.msi /quiet
```

## Quick Start

```bash
# Initialize FoGit in your repository
fogit init

# Create a feature
fogit feature "User Authentication" -d "A login and signup feature"

# List features
fogit list

# Show feature details
fogit show "User Authentication"

# Link features
fogit link "User Authentication" "Database Schema" depends-on

# Search features
fogit search auth

# Commit with feature tracking
fogit commit -m "Add login form"

# Finish feature
fogit finish
# or 
fogit merge 
```

## Key Features

- **Git-Native**: Stores data in `.fogit/` directory, versioned by Git
- **Human-Readable**: YAML files with clear diffs in pull requests
- **Relationship-Aware**: Track dependencies, compositions, references, conflicts
- **Flexible**: User-defined features (code, design, requirements, specs)
- **Portable**: Works with any Git hosting (GitHub, GitLab, Bitbucket)

## Documentation

See the [spec/](spec/) directory for complete documentation:

- [Specification](spec/specification/) - Complete technical specification
- [Guides](spec/guides/) - Implementation and usage guides
- [Reference](spec/reference/) - CLI commands, file formats, glossary
- [Examples](spec/examples/) - Use cases and workflows

## Development

```bash
git clone https://github.com/eg3r/fogit.git
cd fogit
git submodule update --init --recursive
go build -o fogit .
go test ./...
golangci-lint run
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## License

[MIT](LICENSE)
