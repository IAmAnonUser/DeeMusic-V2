# Contributing to DeeMusic Go

Thank you for your interest in contributing to DeeMusic Go! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions with the project and community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/deemusic-go.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Test your changes: `make test`
6. Commit your changes: `git commit -am 'Add some feature'`
7. Push to the branch: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

### Prerequisites

**Required:**
- Go 1.21 or higher
- .NET 8.0 SDK or higher
- C Compiler (MinGW-w64 or TDM-GCC for CGO)
- Git

**Optional:**
- Visual Studio 2022 (for C# development)
- NSIS 3.0+ (for building installer)
- Make (for build automation)

### Installation

```powershell
# Clone the repository
git clone https://github.com/deemusic/deemusic-go.git
cd deemusic-go

# Verify prerequisites
.\scripts\verify-dependencies.ps1

# Build Go backend DLL
.\scripts\build-dll.ps1

# Build C# WPF frontend
.\scripts\build-wpf.ps1

# Or build everything
.\scripts\build-all.ps1

# Run tests
go test ./...
dotnet test tests/IntegrationTests
dotnet test tests/UITests
```

## Coding Standards

### Go Code

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Run `go vet` before committing
- Write tests for new functionality
- Maintain test coverage above 70%
- Document all exported functions
- Use meaningful variable names
- Keep functions focused and small

### C# Code

- Follow [C# Coding Conventions](https://docs.microsoft.com/en-us/dotnet/csharp/fundamentals/coding-style/coding-conventions)
- Use PascalCase for public members
- Use camelCase for private fields (with _ prefix)
- Follow MVVM pattern for UI code
- Use async/await for I/O operations
- Document public APIs with XML comments
- Write unit tests for ViewModels
- Keep Views simple (logic in ViewModels)

### Commit Messages

- Use clear and descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Reference issue numbers when applicable

Example:
```
Add support for FLAC downloads

- Implement FLAC decryption
- Add quality selection in UI
- Update tests

Fixes #123
```

## Testing

### Go Backend Tests

```powershell
# Run all Go tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/download -v
```

### C# Integration Tests

```powershell
# Run integration tests (C# ↔ Go)
dotnet test tests/IntegrationTests

# Run UI tests
dotnet test tests/UITests

# Run performance tests
dotnet test tests/PerformanceTest
```

### Test Requirements

- Write unit tests for all new Go functions
- Write integration tests for P/Invoke interactions
- Write UI tests for new ViewModels
- Ensure all tests pass before submitting PR
- Maintain test coverage above 70% for Go code
- Test on Windows (required for WPF)

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update documentation for any API changes
3. Ensure all tests pass
4. Request review from maintainers
5. Address any feedback from code review
6. Once approved, your PR will be merged

## Reporting Bugs

When reporting bugs, please include:

- Operating system and version
- Go version
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Any error messages or logs

## Feature Requests

We welcome feature requests! Please:

- Check if the feature has already been requested
- Clearly describe the feature and its use case
- Explain why it would be valuable to users

## Development Workflow

### Branch Strategy

- `main`: Stable release branch
- `develop`: Development branch
- `feature/*`: Feature branches
- `bugfix/*`: Bug fix branches
- `hotfix/*`: Urgent fixes for production

### Making Changes

1. Create a feature branch from `develop`
2. Make your changes
3. Write/update tests
4. Update documentation
5. Commit with clear messages
6. Push and create PR to `develop`

### Code Review

- All PRs require review
- Address feedback promptly
- Keep PRs focused and small
- Ensure CI passes

## Architecture

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) and [design.md](.kiro/specs/standalone-windows-app/design.md) for detailed architecture documentation.

### Key Components

**Go Backend (DLL):**
- **Exported Functions**: C exports for P/Invoke
- **Download Manager**: Queue and worker pool management
- **API Clients**: Deezer and Spotify integration
- **Decryption**: Audio stream decryption
- **Metadata**: Tag and artwork embedding
- **Store**: SQLite database operations
- **Callbacks**: Progress notifications to C#

**C# Frontend (WPF):**
- **ViewModels**: MVVM pattern, business logic
- **Views**: XAML UI definitions
- **Services**: P/Invoke wrapper, system integration
- **Models**: Data transfer objects
- **Controls**: Custom WPF controls

### Adding New Features

**Backend Feature:**
1. Implement in appropriate `internal/` package
2. Export function in `cmd/deemusic-core/exports.go`
3. Add P/Invoke declaration in C# `GoBackendService.cs`
4. Write Go tests
5. Write integration tests

**Frontend Feature:**
1. Create/update ViewModel
2. Create/update XAML View
3. Implement data binding
4. Add commands if needed
5. Write UI tests

**Full-Stack Feature:**
1. Design the feature (backend + frontend)
2. Implement backend first
3. Add P/Invoke wrapper
4. Implement frontend
5. Write integration tests
6. Update documentation

## P/Invoke API Development

### Adding New Go Exports

1. **Implement in Go:**
   ```go
   //export NewFunction
   func NewFunction(param *C.char) *C.char {
       // Implementation
       result := doSomething(C.GoString(param))
       return C.CString(result)
   }
   ```

2. **Add P/Invoke Declaration in C#:**
   ```csharp
   [DllImport("deemusic-core.dll")]
   public static extern IntPtr NewFunction(string param);
   ```

3. **Add Wrapper Method:**
   ```csharp
   public async Task<string> NewFunctionAsync(string param)
   {
       return await Task.Run(() => {
           IntPtr ptr = NewFunction(param);
           return Marshal.PtrToStringAnsi(ptr);
       });
   }
   ```

4. **Write Tests:**
   - Go unit test
   - C# integration test
   - Test memory management

### Adding Callbacks

1. **Define in Go:**
   ```go
   type NewCallback func(data *C.char)
   var newCallback NewCallback
   
   //export SetNewCallback
   func SetNewCallback(cb NewCallback) {
       newCallback = cb
   }
   ```

2. **Define Delegate in C#:**
   ```csharp
   public delegate void NewCallback(string data);
   
   [DllImport("deemusic-core.dll")]
   public static extern void SetNewCallback(NewCallback callback);
   ```

3. **Implement Handler:**
   ```csharp
   private void OnNewCallback(string data)
   {
       // Marshal to UI thread if needed
       Application.Current.Dispatcher.Invoke(() => {
           // Update UI
       });
   }
   ```

## Database Changes

### Migrations

1. Create migration in `internal/store/migrations.go`
2. Increment schema version
3. Test upgrade and downgrade
4. Update documentation

### Schema Changes

- Always use migrations
- Never modify existing migrations
- Test with existing data
- Consider backwards compatibility

## Frontend Development

### Setup

```powershell
# Open in Visual Studio
start DeeMusic.Desktop.sln

# Or use VS Code
code DeeMusic.Desktop
```

### Structure

```
DeeMusic.Desktop/
├── ViewModels/      # MVVM ViewModels
├── Views/           # XAML Views
├── Models/          # Data models
├── Services/        # P/Invoke, system services
├── Controls/        # Custom WPF controls
└── Resources/       # Styles, themes, icons
```

### Guidelines

- Follow MVVM pattern strictly
- Use data binding (avoid code-behind)
- Implement INotifyPropertyChanged
- Use RelayCommand for commands
- Keep Views simple (logic in ViewModels)
- Use async/await for I/O operations
- Marshal callbacks to UI thread
- Write accessible XAML (AutomationProperties)

## Performance Considerations

- Profile before optimizing
- Use goroutines appropriately
- Avoid blocking operations
- Cache when beneficial
- Monitor memory usage

## Security Guidelines

- Never log sensitive data (ARL tokens)
- Validate all inputs
- Use parameterized queries
- Encrypt sensitive data at rest
- Follow OWASP guidelines

## Documentation

### Code Documentation

- Document all exported functions
- Use godoc format
- Include examples for complex functions
- Keep comments up to date

### User Documentation

- Update relevant docs with changes
- Include screenshots for UI changes
- Write clear, concise instructions
- Test documentation accuracy

## Release Process

1. Update version in code
2. Update CHANGELOG.md
3. Create release branch
4. Build and test installers
5. Tag release
6. Create GitHub release
7. Update documentation

## Questions?

If you have questions, feel free to:

- Open an issue with the "question" label
- Join our community discussions
- Check [ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Review [API_REFERENCE.md](docs/API_REFERENCE.md)

Thank you for contributing to DeeMusic Go!
