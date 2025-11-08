# Security Package

This package provides security features for DeeMusic, including ARL token encryption and security middleware.

## Features

### ARL Token Encryption

The `TokenEncryptor` provides secure storage for Deezer ARL tokens using:

- **AES-256-GCM encryption** for file-based storage
- **Windows Credential Manager** integration (when available)
- **PBKDF2 key derivation** with machine-specific identifiers
- **Automatic fallback** from Credential Manager to file encryption

#### Usage

```go
import "github.com/deemusic/deemusic-go/internal/security"

// Create encryptor
encryptor := security.NewTokenEncryptor("/path/to/data/dir")

// Encrypt token
encrypted, err := encryptor.EncryptToken("your-arl-token")
if err != nil {
    log.Fatal(err)
}

// Decrypt token
decrypted, err := encryptor.DecryptToken(encrypted)
if err != nil {
    log.Fatal(err)
}
```

#### How It Works

1. **Windows Platform**: Attempts to store token in Windows Credential Manager first
   - If successful, returns "CREDENTIAL_MANAGER" marker
   - On retrieval, fetches from Credential Manager
   - Falls back to file encryption if Credential Manager fails

2. **File Encryption** (all platforms):
   - Generates a random salt on first use
   - Derives encryption key using PBKDF2 with:
     - Machine hostname + username as password
     - 100,000 iterations
     - SHA-256 hash function
   - Encrypts token using AES-256-GCM
   - Stores encrypted token in config file
   - Stores salt in `.key` file with 0600 permissions

3. **Security Properties**:
   - Tokens are never stored in plaintext
   - Each encryption uses a unique nonce (prevents replay attacks)
   - Key is derived from machine-specific data
   - Key file has restricted permissions (owner read/write only)
   - Uses authenticated encryption (GCM mode)

## Integration with Config

The config package automatically encrypts/decrypts ARL tokens:

```go
// Load config (automatically decrypts ARL)
cfg, err := config.Load("")
if err != nil {
    log.Fatal(err)
}

// Use decrypted ARL
client := api.NewDeezerClient(cfg.Deezer.ARL)

// Save config (automatically encrypts ARL)
err = cfg.Save(configPath)
```

## Security Considerations

1. **Machine Binding**: Encryption key is derived from machine-specific data, so encrypted tokens cannot be easily transferred between machines
2. **Backward Compatibility**: If a plaintext token is detected during load, it's automatically encrypted
3. **Key Storage**: The `.key` file should be protected and not shared
4. **Credential Manager**: On Windows, Credential Manager provides OS-level protection

## Security Middleware

The package provides several middleware components for Gin:

### CSRF Protection

Protects against Cross-Site Request Forgery attacks:

```go
csrfManager := security.NewCSRFManager(logger)
router.Use(security.CSRFMiddleware(csrfManager))
```

- Generates CSRF tokens for GET requests
- Validates tokens on state-changing requests (POST, PUT, DELETE)
- Tokens are stored in both cookies and headers
- Automatic token expiration (24 hours)

### Localhost-Only Enforcement

Ensures the server only accepts connections from localhost:

```go
router.Use(security.LocalhostOnlyMiddleware(logger))
```

- Rejects connections from non-localhost IPs
- Allows 127.0.0.1, ::1, and localhost
- Logs rejected connection attempts

### Input Sanitization

Sanitizes user inputs to prevent injection attacks:

```go
router.Use(security.InputSanitizationMiddleware())
```

- Removes null bytes
- Removes control characters (except newline and tab)
- Sanitizes query parameters

### Path Validation

Validates file paths to prevent directory traversal:

```go
router.Use(security.PathValidationMiddleware(logger))
```

- Detects path traversal attempts (../)
- Rejects absolute paths
- Validates path parameters in requests

Use `ValidateFilePath` for manual path validation:

```go
safePath, err := security.ValidateFilePath(baseDir, userPath)
if err != nil {
    // Handle invalid path
}
```

### Security Headers

Adds security headers to all responses:

```go
router.Use(security.SecurityHeadersMiddleware())
```

Headers added:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: ...`
- `Referrer-Policy: strict-origin-when-cross-origin`

## Testing

Run tests with:

```bash
go test ./internal/security/...
```

Tests cover:
- Basic encryption/decryption
- Key persistence across instances
- Invalid input handling
- Multiple encryptions of same token
- Key deletion
- CSRF token generation and validation
- Localhost-only enforcement
- Input sanitization
- Path validation
- Security headers
