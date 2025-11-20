# GitHub Actions Workflows

This directory contains GitHub Actions workflows for the go-workflow project, providing comprehensive CI/CD, security, and quality assurance.

## 🔄 Workflows Overview

### 1. Tests (`test.yml`)
**Triggers:** Push/PR to `main` or `develop` branches

**Features:**
- **Multi-version testing**: Tests against Go 1.20, 1.21, and 1.22
- **Race condition detection**: Uses `-race` flag during testing
- **Code coverage**: Generates coverage reports and uploads to Codecov
- **Dependency caching**: Speeds up builds with Go module caching
- **Linting**: Runs golangci-lint for code quality checks
- **Example builds**: Validates all example programs compile successfully

### 2. SLSA Go Releaser (`slsa-goreleaser.yml`)
**Triggers:** Manual dispatch or release creation

**Features:**
- **SLSA Level 3 compliance**: Provides supply chain security
- **Provenance generation**: Creates verifiable build provenance
- **Secure builds**: Uses hardened build environment
- **Release automation**: Automatically builds and uploads release assets

**Configuration:** Requires `.slsa-goreleaser.yml` in project root

### 3. Code Quality (`quality.yml`)
**Triggers:** Push/PR to `main` or `develop` branches

**Features:**
- **Security scanning**: Uses Gosec for security vulnerability detection
- **Dependency checking**: Runs govulncheck for known vulnerabilities
- **Module verification**: Ensures go.mod and go.sum integrity

## 📋 Configuration Files

### `.slsa-goreleaser.yml`
```yaml
version: 1
goos: linux
goarch: amd64
binary: go-workflow
ldflags:
  - "-s -w"
  - "-X main.version={{ .Version }}"
  - "-X main.commit={{ .FullCommit }}"
  - "-X main.date={{ .Date }}"
```

### `.golangci.yml`
Comprehensive linting configuration with:
- 30+ enabled linters
- Custom rules for test files
- Performance and security checks
- Code style enforcement

## 🚀 Getting Started

### Prerequisites
1. Go 1.20+ installed
2. Project uses Go modules
3. Tests located in `*_test.go` files

### Local Development
```bash
# Run tests locally
go test -v -race ./...

# Run linter locally
golangci-lint run

# Check for security issues
gosec ./...

# Check for vulnerabilities
govulncheck ./...
```

### Release Process
1. Create a new release on GitHub
2. SLSA workflow automatically triggers
3. Binary built with provenance
4. Assets uploaded to release

## 📊 Status Badges

Add these badges to your main README:

```markdown
[![Tests](https://github.com/ComingCL/go-workflow/actions/workflows/test.yml/badge.svg)](https://github.com/ComingCL/go-workflow/actions/workflows/test.yml)
[![Code Quality](https://github.com/ComingCL/go-workflow/actions/workflows/quality.yml/badge.svg)](https://github.com/ComingCL/go-workflow/actions/workflows/quality.yml)
[![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev)
```

## 🔧 Customization

### Adding New Linters
Edit `.golangci.yml` to enable additional linters:
```yaml
linters:
  enable:
    - your-new-linter
```

### Changing Go Versions
Update the matrix in `test.yml`:
```yaml
strategy:
  matrix:
    go-version: [ '1.21', '1.22', '1.23' ]
```

### Custom Build Flags
Modify `.slsa-goreleaser.yml` ldflags:
```yaml
ldflags:
  - "-X main.customFlag=value"
```

## 🛡️ Security Features

- **Dependency scanning**: Automated vulnerability detection
- **Security linting**: Static analysis for security issues
- **Supply chain security**: SLSA Level 3 compliance
- **Provenance verification**: Cryptographically signed build artifacts

## 📈 Monitoring

- **Test results**: View in Actions tab
- **Coverage reports**: Available on Codecov
- **Security alerts**: GitHub Security tab
- **Quality metrics**: golangci-lint reports

## 🤝 Contributing

When contributing:
1. Ensure all tests pass locally
2. Run linters before submitting PR
3. Add tests for new functionality
4. Update documentation as needed

The CI will automatically run all checks on your PR.