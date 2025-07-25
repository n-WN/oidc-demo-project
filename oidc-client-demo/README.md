# OIDC Client Demo - Relying Party Application

## Overview
This is a demonstration client application (Relying Party) that integrates with the OIDC Provider to showcase how modern web applications implement authentication using OpenID Connect. The project features a modular architecture with advanced debugging and decoding capabilities.

## 🏗️ Modular Architecture

### File Structure
```
oidc-client-demo/
├── main.go      # Core OIDC client logic and HTTP handlers
├── debug.go     # HTTP request/response debugging transport
├── decoder.go   # Smart data format decoder module
├── go.mod       # Go module dependencies
├── README.md    # This file
└── README_zh.md # Chinese documentation
```

### Module Responsibilities

#### `main.go` - Core Logic
- OIDC authentication flow
- HTTP route handlers
- Session management
- State validation

#### `debug.go` - Network Debugging
- HTTP request/response interception
- Transport-level logging
- Network timing analysis
- Integration with decoder module

#### `decoder.go` - Smart Decoding
- Automatic format detection (JSON, JWT, URL-encoded, Base64)
- Beautiful output formatting
- JWT claims explanation
- Recursive decoding capabilities

## Core Concepts in Action

### The Authentication Triangle
```mermaid
graph TD
    A[User] -->|1. Requests Access| B[Client App - Port 8080]
    B -->|2. Redirects for Auth| C[OIDC Provider - Port 9090]
    C -->|3. Authenticates| A
    C -->|4. Returns Tokens| B
    B -->|5. Grants Access| A
  
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style C fill:#e8f5e8
```

### OAuth2 vs OIDC vs SSO - What This Demo Shows

#### OAuth2 Layer (Authorization)
- **What it answers**: "Can this app access user resources?"
- **Demo demonstrates**: 
  - Authorization code exchange
  - Access token usage
  - Secure redirect handling

#### OIDC Layer (Authentication) 
- **What it answers**: "Who is this user?"
- **Demo demonstrates**:
  - ID Token validation
  - JWT signature verification
  - User claims extraction

#### SSO Experience (User Journey)
- **What it provides**: "Login once, access everywhere"
- **Demo demonstrates**:
  - Seamless authentication redirect
  - Automatic session establishment
  - User-friendly login experience

## Technical Architecture

### Client-Side Implementation Pattern
```mermaid
sequenceDiagram
    participant Browser
    participant Client as Client App
    participant Provider as OIDC Provider
  
    Browser->>Client: GET /protected
    Client->>Client: Check session
    alt No valid session
        Client->>Browser: Redirect to /auth/login
        Browser->>Client: GET /auth/login
        Client->>Provider: Redirect to authorization endpoint
        Provider->>Browser: Show login form
        Browser->>Provider: Submit credentials
        Provider->>Browser: Redirect with auth code
        Browser->>Client: GET /auth/callback?code=xyz
        Client->>Provider: POST /token (exchange code)
        Provider->>Client: Return ID Token + Access Token
        Client->>Client: Validate ID Token
        Client->>Client: Create session
        Client->>Browser: Redirect to /protected
    end
    Client->>Browser: Serve protected content
```

### Key Components

#### 1. Authentication Middleware
- Session management
- Token validation
- Automatic OIDC redirects

#### 2. Token Handling
- **ID Token Validation**:
  - Signature verification using JWKS
  - Issuer (`iss`) validation
  - Audience (`aud`) validation
  - Expiration (`exp`) checking

#### 3. Callback Processing
- Authorization code reception
- Token exchange with provider
- User session establishment

## Quick Start

### Prerequisites
- Go 1.21 or higher
- OIDC Provider running on port 9090 (see `../simple-oidc-provider/`)

### Installation & Run
```bash
cd oidc-client-demo
go mod tidy
go run main.go
```

The client will start on `http://127.0.0.1:8080`

### Default Configuration
```go
oidcConfig := OIDCConfig{
    ClientID:     "my-client-app",
    ClientSecret: "my-client-secret",
    RedirectURI:  "http://127.0.0.1:8080/auth/callback",
    ProviderURL:  "http://127.0.0.1:9090",
}
```

## User Journey Demonstration

### Step-by-Step Flow
1. **Access Protected Resource**: Visit `http://127.0.0.1:8080/protected`
2. **Automatic Redirect**: Browser redirects to OIDC provider login
3. **User Authentication**: Enter credentials (demo/password)
4. **Consent Grant**: Approve application access
5. **Return to Client**: Automatic redirect back with tokens
6. **Access Granted**: View protected content with user information

### What You'll See
- **Login Page**: OIDC provider's authentication interface
- **Consent Screen**: Permission grant for client application
- **Protected Page**: User profile information from ID Token
- **Session Management**: Persistent login across page refreshes

## Advanced Go Programming Concepts

### Context in HTTP Debugging

This client demonstrates sophisticated Go programming patterns using the `context` package for HTTP debugging and dependency injection:

#### How Context Powers OAuth2 Debugging

The project showcases how OAuth2 libraries accept custom HTTP clients through context:

```go
// Create debugging HTTP client
client := &http.Client{
    Transport: NewDebugTransport(),
}

// Inject via context - this is dependency injection in Go!
ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

// OAuth2 library automatically uses our debug client
token, err := oauth2Config.Exchange(ctx, code)
```

#### Key Benefits

1. **Non-intrusive Debugging**: Add HTTP tracing without changing core logic
2. **Request Correlation**: Track requests across function calls
3. **Flexible Configuration**: Pass timeouts, clients, or custom headers
4. **Production-Ready**: Easy to enable/disable debugging

#### Modular Design

- **`debug.go`**: Custom HTTP transport with request/response logging
- **`decoder.go`**: Smart data format detection and pretty-printing
- **`main.go`**: Clean business logic that uses debugging components

### HTTP Request Tracing

The debug module provides detailed insights into OAuth2 network traffic:

```go
// See exactly what OAuth2 library sends
POST /token HTTP/1.1
Content-Type: application/x-www-form-urlencoded

client_id=demo-client&code=abc123&grant_type=authorization_code...

// And exactly what it receives  
HTTP/1.1 200 OK
Content-Type: application/json

{"access_token":"...", "id_token":"...", "token_type":"Bearer"}
```

## Educational Features

### 1. Real OIDC Implementation
Demonstrates production-ready patterns:
- Proper state parameter usage (CSRF protection)
- Secure token storage
- JWT validation best practices

### 2. Security Demonstrations
Shows critical security measures:
- **PKCE Flow**: Proof Key for Code Exchange (recommended for SPAs)
- **State Validation**: Prevents CSRF attacks
- **Nonce Checking**: Prevents replay attacks
- **Token Verification**: Cryptographic signature validation

### 3. Error Handling
Includes proper error scenarios:
- Invalid authorization codes
- Expired tokens
- Network failures
- Malformed responses

## API Routes

| Route | Method | Purpose | Authentication |
|-------|--------|---------|----------------|
| `/` | GET | Home page | None |
| `/protected` | GET | Protected resource | Required |
| `/auth/login` | GET | Initiate OIDC flow | None |
| `/auth/callback` | GET | Handle OIDC response | None |
| `/auth/logout` | GET | Clear session | None |
| `/profile` | GET | User profile page | Required |

## Configuration Options

### Environment Variables
```bash
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"
export OIDC_PROVIDER_URL="https://your-provider.com"
export OIDC_REDIRECT_URI="http://localhost:8080/auth/callback"
```

### Custom Provider Integration
To integrate with other OIDC providers (Google, Auth0, etc.):
```go
// Update provider discovery URL
ProviderURL: "https://accounts.google.com"

// Or specify endpoints directly
AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth"
TokenEndpoint: "https://oauth2.googleapis.com/token"
JWKSEndpoint: "https://www.googleapis.com/oauth2/v3/certs"
```

## Key Learning Points

### 1. Protocol Understanding
- **Authorization vs Authentication**: OAuth2 handles "what can you do", OIDC handles "who are you"
- **Token Types**: Access tokens for API access, ID tokens for identity
- **Security Model**: Why signatures and validation matter

### 2. Integration Patterns
- **Discovery Protocol**: How clients find provider endpoints
- **Dynamic Configuration**: Adapting to different OIDC providers
- **Session Management**: Bridging stateless tokens with stateful sessions

### 3. User Experience
- **Seamless Redirects**: Invisible authentication flow
- **Error Recovery**: Graceful handling of auth failures
- **Logout Handling**: Proper session termination

## Testing & Development

### Local Testing
1. Start OIDC Provider: `cd ../simple-oidc-provider && go run main.go`
2. Start Client: `go run main.go`
3. Test flow: Visit `http://127.0.0.1:8080/protected`

### Integration with Real Providers
This client can be configured to work with:
- Google Identity Platform
- Auth0
- Microsoft Azure AD
- Okta
- Any compliant OIDC provider

### Debugging Tips
- Check browser network tab for redirect flows
- Validate JWT tokens at [jwt.io](https://jwt.io)
- Monitor provider discovery endpoint responses
- Verify JWKS public key matching

## Production Considerations

### Security Enhancements
- **HTTPS Only**: Never run OIDC over HTTP in production
- **Secure Cookies**: HTTPOnly, Secure, SameSite attributes
- **PKCE Implementation**: Enhanced security for public clients
- **Token Refresh**: Implement refresh token flow

### Scalability Features
- **Distributed Sessions**: Redis or database-backed sessions
- **Token Caching**: Cache JWKS keys and discovery metadata
- **Load Balancing**: Session affinity considerations

### Monitoring & Observability
- Authentication success/failure rates
- Token validation errors
- Provider response times
- User authentication patterns

This demo provides a complete foundation for understanding and implementing OIDC authentication in modern web applications, bridging the gap between OAuth2 authorization and SSO user experience.
