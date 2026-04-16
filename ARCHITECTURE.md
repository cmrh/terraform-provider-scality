# Scality Terraform Provider - Technical Architecture Document

## Executive Summary

This document explains the architectural decisions, design patterns, and implementation strategies used in the Scality Terraform Provider. The primary goals are:

1. **Maintainability** - Easy to understand and modify
2. **Extensibility** - Simple to add new API calls and resources
3. **Reliability** - Consistent error handling and state management
4. **Simplicity** - Reduce complexity while maintaining DRY principles

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Design Principles](#design-principles)
3. [Client Architecture](#client-architecture)
4. [Resource Pattern](#resource-pattern)
5. [Adding New API Calls](#adding-new-api-calls)
6. [Error Handling Strategy](#error-handling-strategy)
7. [State Management](#state-management)
8. [Security Patterns](#security-patterns)
9. [Testing Strategy](#testing-strategy)
10. [Future Extensibility](#future-extensibility)

---

## Architecture Overview

### High-Level Structure

```
┌─────────────────────────────────────────────────────────────┐
│                    Terraform / OpenTofu Core                 │
│           (Handles plan, apply, destroy lifecycle)           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ Plugin Protocol (gRPC)
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Scality Terraform Provider                     │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   Provider   │  │  Resources   │  │ Data Sources │     │
│  │ (provider.go)│  │   (3 total)  │  │   (future)   │     │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘     │
│         │                  │                                │
│         ▼                  ▼                                │
│  ┌──────────────────────────────────────────────┐          │
│  │          Client Layer (ProviderClients)       │          │
│  │                                               │          │
│  │  ┌─────────────────┐  ┌─────────────────┐   │          │
│  │  │  ScalityClient  │  │  ConsoleClient   │   │          │
│  │  │  (AWS SigV4)    │  │  (JWT Token)     │   │          │
│  │  └────────┬────────┘  └────────┬─────────┘   │          │
│  │           │                    │              │          │
│  │           │           ┌────────┴─────────┐   │          │
│  │           │           │  File-Based      │   │          │
│  │           │           │  Token Cache     │   │          │
│  │           │           └──────────────────┘   │          │
│  └───────────┼────────────────────┼──────────────┘          │
│              │                    │                          │
└──────────────┼────────────────────┼──────────────────────────┘
               │                    │
               ▼                    ▼
    ┌──────────────────┐  ┌──────────────────┐
    │   Scality IAM    │  │ Scality Console  │
    │      API         │  │      API         │
    │  AWS SigV4       │  │  JWT / REST      │
    └──────────────────┘  └──────────────────┘
```

### Key Components

| Component | Purpose | Files |
|-----------|---------|-------|
| **Provider** | Configuration, auth, client setup | `internal/provider/provider.go` |
| **IAM Client** | AWS SigV4 signed API calls | `internal/client/iam.go` |
| **Console Client** | JWT-authenticated REST API | `internal/client/console.go` |
| **ProviderClients** | Bundles both clients | `internal/client/provider_clients.go` |
| **Resources** | Terraform resource implementations | `internal/resources/*/` |

### Two API Surfaces

Scality exposes two distinct APIs that the provider interacts with:

| API | Auth | Format | Purpose |
|-----|------|--------|---------|
| **IAM API** | AWS Signature V4 (admin credentials) | JSON/Form-encoded actions | Accounts, access keys |
| **Console API** | JWT Bearer (`x-access-token` header) | JSON/REST | Accounts (password-free), access keys |

### Directory Structure

```
terraform-provider-scality/
├── main.go                                      # Provider server entry point
├── go.mod                                       # Module: github.com/scality/terraform-provider-scality
├── Makefile                                     # build, install, test, testacc, fmt
├── examples/
│   ├── main.tf                                  # Basic IAM usage example
│   └── multiple-accounts.tf                     # Multi-account with for_each
├── internal/
│   ├── client/
│   │   ├── provider_clients.go                  # ProviderClients bundle
│   │   ├── iam.go                               # IAM SigV4 client + data types + crypto helpers
│   │   └── console.go                           # Console JWT client + token cache
│   ├── provider/
│   │   └── provider.go                          # Schema, Configure, resource registration
│   └── resources/
│       ├── account/                             # scality_account (IAM)
│       │   ├── model.go
│       │   └── resource.go
│       ├── console_account/                     # scality_console_account (Console)
│       │   ├── model.go
│       │   └── resource.go
│       └── account_access_key/                  # scality_account_access_key (IAM)
│           ├── model.go
│           └── resource.go
```

Each resource lives in its own package with exactly two files:
- `model.go` — Terraform schema model struct with `tfsdk` tags
- `resource.go` — CRUD implementation, schema, and conversion helpers

---

## Design Principles

### 1. Separation of Concerns

Decision: Split IAM and Console clients into separate files with a shared bundle.

Reasoning:
- Different authentication mechanisms (AWS SigV4 vs JWT)
- Different wire formats (form-encoded actions vs JSON/REST)
- Different API patterns and endpoints
- Independent evolution of each API surface

```
internal/client/iam.go      → IAM API (accounts, access keys)
internal/client/console.go  → Console API (accounts, access keys)
```

The `ProviderClients` struct bundles both clients into a single value passed through `resp.ResourceData`, so each resource extracts only the client it needs:

```go
// IAM resources:
r.client = clients.IAMClient

// Console resources:
r.client = clients.ConsoleClient
```

Benefits: Clear responsibility boundaries, no coupling between API types, easy to add a third client if needed.

---

### 2. DRY Principle with Pragmatism

Decision: Apply DRY where it reduces complexity, not religiously.

#### Where We Applied DRY

**1. Shared Constants**

```go
// client.go — IAM / AWS SigV4 constants
const (
    awsService         = "iam"
    awsRegion          = "us-east-1"
    awsAlgorithm       = "AWS4-HMAC-SHA256"
    apiVersion         = "2010-05-08"
    defaultHTTPTimeout = 30 * time.Second
    contentTypeForm    = "application/x-www-form-urlencoded"
)

// console_client.go — Console API constants
const (
    consoleAuthPath    = "/_/console/authenticate"
    consoleAccountPath = "/_/console/vault/accounts"
    consoleContentType = "application/json"
    tokenCachePrefix   = ".scality_console_token_"
    tokenSafetyMargin  = 84600  // 23.5 hours in seconds
    filePermissions    = 0600
)
```

Reasoning: Changes once, benefits everywhere. Self-documenting names replace magic values.

**2. IAM Helper Method**

```go
func (c *ScalityClient) doSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error)
```

The helper sets `Action` and `Version` automatically, handles SigV4 signing, HTTP execution, and returns the raw response. All 5 IAM operations (CreateAccount, GenerateAccountAccessKey, DeleteAccessKey, GetAccount, DeleteAccount) are thin wrappers around this helper.

#### Where We Didn't Apply DRY

**Console client HTTP calls are kept separate from IAM.**

```go
// IAM: AWS SigV4 authorization, form-encoded body
headers, err := c.signRequest(method, url, payload)

// Console: JWT token header, JSON body
httpReq.Header.Set("x-access-token", c.token)
```

These are different enough that a shared abstraction would add complexity without reducing it.

Rule of Thumb:
- Duplicate when abstractions would be more complex than the duplication
- Abstract when the pattern is identical and changes together

---

### 3. Context-First Design

Decision: Every public API method accepts `context.Context` as the first parameter.

```go
func (c *Client) MethodName(ctx context.Context, params...) (result, error)
```

Reasoning:
1. **Cancellation** — Terraform provides context to resources; propagating it through enables proper lifecycle management
2. **Timeouts** — HTTP clients respect context deadlines
3. **Logging** — `tflog` uses context for structured logging
4. **Future-proofing** — Distributed tracing, OpenTelemetry integration

Flow:
```
Terraform → Resource.Create(ctx) → Client.CreateAccount(ctx) → HTTP Request (with context)
                                                                         ↓
                                                        Respects timeout/cancellation
```

---

### 4. Constants Over Magic Values

Decision: All configuration values are package-level constants.

Before (magic values):
```go
params.Set("Version", "2010-05-08")
region := "us-east-1"
if tokenAge >= 84600 {
```

After (constants):
```go
params.Set("Version", apiVersion)
region := awsRegion
if tokenAge >= tokenSafetyMargin {
```

Benefits:

| Benefit | Example |
|---------|---------|
| Single source of truth | Update API version in one place |
| Self-documenting | `tokenSafetyMargin` explains purpose |
| Type safety | Compiler catches typos |
| Easy refactoring | Find all usages via IDE |

---

### 5. Atomic Create Pattern

Decision: Save resource state immediately after account creation, before generating access keys.

Create operations are two-step: create the account, then generate S3 access keys. If key generation fails after account creation, the resource must still be tracked in state — otherwise it becomes orphaned (exists on server but unknown to Terraform).

```go
// 1. Create account
account, err := r.client.CreateAccount(ctx, createReq)
if err != nil { ... return }

// 2. Save state immediately — resource is now tracked
data.ID = types.StringValue(account.Account.Data.ID)
resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

// 3. Generate keys — partial failure is now recoverable
accessKey, err := r.client.GenerateAccountAccessKey(ctx, data.Name.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Client Error",
        "Account created successfully but access key generation failed. "+
        "The account exists and is tracked in state. Run apply again or use "+
        "scality_account_access_key to generate keys separately: " + err.Error())
    return  // state already saved above
}

// 4. Update state with key data
data.AccessKey = types.StringValue(accessKey.Data.ID)
data.SecretKey = types.StringValue(accessKey.Data.Value)
resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
```

Both `scality_account` and `scality_console_account` use this pattern.

---

## Client Architecture

### IAM Client (`client.go`)

#### Design

```go
type ScalityClient struct {
    Endpoint   string
    AccessKey  string
    SecretKey  string
    HTTPClient *http.Client
}
```

Simple struct with clear dependencies. Immutable after creation.

#### Helper Method Pattern

```go
// Private helper — handles HTTP mechanics
func (c *ScalityClient) doSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error) {
    // 1. Set Action and Version automatically
    params.Set("Action", action)
    params.Set("Version", apiVersion)

    // 2. Sign request with AWS SigV4
    // 3. Create HTTP request with context
    // 4. Execute
    // 5. Return body, status code, error
}

// Public methods — handle business logic and status code interpretation
func (c *ScalityClient) CreateAccount(ctx context.Context, req AccountCreateRequest) (*AccountCreateResponse, error) {
    params := url.Values{}
    params.Set("name", req.Name)

    body, statusCode, err := c.doSignedRequest(ctx, "CreateAccount", params)
    if statusCode == 409 {
        return nil, fmt.Errorf("account already exists")
    }
    // Parse and return
}
```

The helper returns status codes to callers because different operations interpret the same code differently — a 404 means "not found" in Read (return nil) but is unexpected in Create.

#### SigV4 Signing

```go
func (c *ScalityClient) signRequest(method, requestURL, payload string) (map[string]string, error) {
    // 1. Parse URL to get host
    // 2. Create canonical request (method, path, headers, payload hash)
    // 3. Create string to sign (timestamp, credential scope, request hash)
    // 4. Derive signing key (HMAC chain: date → region → service → "aws4_request")
    // 5. Calculate signature
    // 6. Return Authorization, X-Amz-Date, X-Amz-Content-Sha256 headers
}
```

---

### Console Client (`console_client.go`)

#### Design

```go
type ConsoleClient struct {
    Endpoint   string
    Username   string
    Password   string
    HTTPClient *http.Client
    token      string  // Cached token (private)
}
```

#### Token Lifecycle

The Console client manages JWT tokens with automatic file-based caching:

```go
func (c *ConsoleClient) Authenticate(ctx context.Context) error {
    // 1. Try cached token from file
    if cachedToken, err := c.getCachedToken(); err == nil {
        c.token = cachedToken
        return nil
    }

    // 2. Authenticate via POST /_/console/authenticate
    // 3. Cache new token to file
}
```

Key design decisions:

1. **File-based caching** — Tokens are cached to `/tmp/.scality_console_token_<hash>` with 0600 permissions. Unique filenames per endpoint/username prevent collisions.
2. **Pre-expiry refresh** — Tokens refresh after `tokenSafetyMargin` (23.5 hours), before the 24-hour JWT expiry, to prevent mid-request failures.
3. **Lazy authentication** — Each public method checks `c.token` and calls `Authenticate(ctx)` on demand.
4. **Cache survives restarts** — File-based cache persists across provider invocations, avoiding re-authentication on every `plan`/`apply`.

#### Why No Shared Helper?

Console client methods don't use a `doRequest` helper like the IAM client.

```go
// Different URL patterns per operation
POST /_/console/vault/accounts              // Create
POST /_/console/vault/accounts/{id}/keys    // Generate keys
GET  /_/console/vault/accounts/{id}         // Read
DELETE /_/console/vault/accounts/{id}       // Delete account
DELETE /_/console/vault/accounts/{id}/user  // Delete user
```

Each method has different URL construction, status code handling, and request bodies. `DeleteConsoleAccount` is a two-step process (delete account, then delete user). A shared helper would need too many parameters to be useful.

Guideline:
- If methods are 70% similar: Extract helper
- If methods are 40% similar: Keep separate
- Console client is ~40% similar: Kept separate

---

## Resource Pattern

### Standard Resource Structure

Every resource follows this exact pattern:

```
internal/resources/<name>/
├── model.go      # Data model with tfsdk tags
└── resource.go   # Schema, Configure, CRUD, conversion helpers
```

#### Model (`model.go`)

```go
type AccountResourceModel struct {
    ID           types.String `tfsdk:"id"`
    Name         types.String `tfsdk:"name"`
    EmailAddress types.String `tfsdk:"email_address"`
    QuotaMax     types.Int64  `tfsdk:"quota_max"`
    AccessKey    types.String `tfsdk:"access_key"`
    SecretKey    types.String `tfsdk:"secret_key"`
    // ...
}
```

Models map directly to the Terraform schema via `tfsdk` struct tags. No business logic lives here.

#### Resource (`resource.go`)

Every resource implements these methods in order:

```go
// 1. Type assertions
var _ resource.Resource = &AccountResource{}
var _ resource.ResourceWithImportState = &AccountResource{}

// 2. Struct holds client reference
type AccountResource struct {
    client *client.IAMClient
}

// 3. Constructor
func NewAccountResource() resource.Resource { return &AccountResource{} }

// 4. Metadata — sets type name
func (r *AccountResource) Metadata(...)

// 5. Schema — defines attributes with plan modifiers
func (r *AccountResource) Schema(...)

// 6. Configure — extracts client from ProviderClients
func (r *AccountResource) Configure(...) {
    clients, ok := req.ProviderData.(*client.ProviderClients)
    r.client = clients.IAM
}

// 7. CRUD operations
func (r *AccountResource) Create(...)   // Atomic: create → save state → gen keys → save state
func (r *AccountResource) Read(...)     // Drift detection via API
func (r *AccountResource) Update(...)   // Error safety net (all fields RequiresReplace)
func (r *AccountResource) Delete(...)

// 8. Import
func (r *AccountResource) ImportState(...)
```

#### CRUD Pattern

```go
func (r *AccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // a. Read plan data
    var data AccountResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // b. Call client
    account, err := r.client.CreateAccount(ctx, createReq)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create account: %s", err))
        return
    }

    // c. Save state immediately (atomic pattern)
    data.ID = types.StringValue(account.Account.Data.ID)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

    // d. Generate keys, update state again
    accessKey, err := r.client.GenerateAccountAccessKey(ctx, data.Name.ValueString())
    // ...
}
```

### RequiresReplace on All Mutable Fields

Scality APIs do not support in-place updates for any account attribute. All user-configurable fields use `RequiresReplace` plan modifiers to force Terraform to destroy and recreate the resource when values change:

```go
"email_address": schema.StringAttribute{
    Required: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),
    },
},
```

The Update method exists (required by the interface) but serves as an error safety net:

```go
func (r *AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    resp.Diagnostics.AddError(
        "Update Not Supported",
        "All scality_account attributes require resource replacement. This is a provider bug if you see this error.",
    )
}
```

### Resource Summary

| Resource | Client | CRUD | Import | Notes |
|----------|--------|------|--------|-------|
| `scality_account` | IAM | CR_D | By name | Keys auto-generated, atomic create |
| `scality_console_account` | Console | CR_D | By name | Optional password gen, two-step delete |
| `scality_account_access_key` | IAM | CR_D | `account_name/access_key_id` | Secret only at creation, state-only read |

---

## Adding New API Calls

### Step-by-Step Guide

#### For IAM API Resources

**Step 1: Add client method to `internal/client/iam.go`**

```go
func (c *ScalityClient) ListAccounts(ctx context.Context, maxResults int) (*AccountListResponse, error) {
    params := url.Values{}
    if maxResults > 0 {
        params.Set("MaxResults", fmt.Sprintf("%d", maxResults))
    }

    // Version and Action are set automatically by doSignedRequest
    body, statusCode, err := c.doSignedRequest(ctx, "ListAccounts", params)
    if err != nil {
        return nil, err
    }

    if statusCode != 200 {
        return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
    }

    var result AccountListResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    return &result, nil
}
```

**Step 2: Create resource package**

```
internal/resources/widget/
├── model.go      # WidgetResourceModel struct
└── resource.go   # NewWidgetResource, CRUD
```

Follow the standard resource pattern (see [Resource Pattern](#resource-pattern)).

**Step 3: Register in `provider.go`**

```go
import widget "github.com/scality/terraform-provider-scality/internal/resources/widget"

func (p *ScalityProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        account.NewAccountResource,
        consoleaccount.NewConsoleAccountResource,
        accountaccesskey.NewAccountAccessKeyResource,
        widget.NewWidgetResource,  // new
    }
}
```

#### For Console API Resources

Same pattern, but add methods to `internal/client/console.go` and ensure token authentication:

```go
func (c *ConsoleClient) CreateWidget(ctx context.Context, name string) (*WidgetResponse, error) {
    // 1. Ensure authenticated
    if c.token == "" {
        if err := c.Authenticate(ctx); err != nil {
            return nil, err
        }
    }

    // 2. Build URL and request
    widgetURL := fmt.Sprintf("%s/_/console/widgets/%s", c.Endpoint, name)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", widgetURL, body)

    // 3. Set headers
    httpReq.Header.Set("Content-Type", consoleContentType)
    httpReq.Header.Set("x-access-token", c.token)

    // 4. Execute and handle response
}
```

### Complexity Decision Tree

```
Is this an API call?
├─ Yes
│  ├─ IAM API (SigV4)?
│  │  └─ Add to internal/client/iam.go, use DoSignedRequest helper
│  │     → Returns (body, statusCode, error)
│  │     → Caller handles status code interpretation
│  │
│  └─ Console API (JWT)?
│     └─ Add to internal/client/console.go, check token first
│        → Handle authentication on demand
│        → Each method constructs its own URL
│
└─ No
   └─ Is this a Terraform resource?
      ├─ Read-only → Create data source
      └─ Mutable → Create resource with model.go + resource.go in internal/resources/<name>/
```

---

## Error Handling Strategy

### Principles

1. **Wrap errors with context** using `%w`
2. **Meaningful messages** that help users solve problems
3. **Distinguish "not found" from real errors** — return `nil` for missing resources
4. **Never swallow errors**

### Error Wrapping

```go
// Good: Wrap with context, preserve error chain
if err != nil {
    return nil, fmt.Errorf("failed to create account: %w", err)
}

// Bad: Lose error chain
if err != nil {
    return nil, fmt.Errorf("failed to create account: %s", err)
}
```

Using `%w` preserves the error chain so callers can use `errors.Is()` and `errors.As()` to inspect root causes.

### User-Facing Error Messages

In resources, errors go through diagnostics with a summary and detail:

```go
resp.Diagnostics.AddError(
    "Client Error",   // Summary — shown bold
    fmt.Sprintf("Unable to create account '%s': %s", data.Name.ValueString(), err),
)
```

Special case — actionable delete errors:

```go
if statusCode == 409 {
    return fmt.Errorf(
        "cannot delete account '%s' - the account contains resources that must be removed first.\n\n"+
            "The account may contain:\n"+
            "  • IAM users\n"+
            "  • IAM policies\n"+
            "  • S3 buckets (empty or with data)\n\n"+
            "Required actions before deletion:\n"+
            "  1. Delete all IAM users in the account\n"+
            "  2. Delete all IAM policies in the account\n"+
            "  3. Delete all objects from S3 buckets\n"+
            "  4. Delete all S3 buckets\n"+
            "  5. Retry account deletion",
        accountName,
    )
}
```

### Not-Found Handling

IAM client returns `nil` for 404:

```go
if statusCode == 404 {
    return nil, nil  // resource doesn't exist
}
```

Resources remove the resource from state when the API returns nil:

```go
if account == nil {
    resp.State.RemoveResource(ctx)  // triggers recreation on next apply
    return
}
```

---

## State Management

### Drift Detection

On every `plan` or `apply`, Terraform calls `Read()` for each resource. The Read method checks the real infrastructure state:

```go
func (r *AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data AccountResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

    account, err := r.client.GetAccount(ctx, data.Name.ValueString())
    if account == nil {
        resp.State.RemoveResource(ctx)  // deleted outside Terraform
        return
    }

    data.ID = types.StringValue(account.Data.ID)
    data.EmailAddress = types.StringValue(account.Data.EmailAddress)
    data.QuotaMax = types.Int64Value(account.Data.QuotaMax)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

### Sensitive Fields Preservation

Access keys and secret keys cannot be retrieved after creation. These fields are preserved from state during Read by simply not overwriting them. Combined with `UseStateForUnknown` plan modifiers on the access key resource:

```go
"secret_key": schema.StringAttribute{
    Computed:  true,
    Sensitive: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},
```

### State-Only Read

The `scality_account_access_key` resource has no API endpoint to retrieve key details after creation. Its Read method preserves the current state:

```go
func (r *AccountAccessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data AccountAccessKeyResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    // No API call — preserve state as-is
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

---

## Security Patterns

### 1. Credential Handling

Never log credentials:

```go
// Good: Log operation context, not secrets
tflog.Debug(ctx, "Creating Scality account", map[string]any{
    "name": data.Name.ValueString(),
})

// Bad: Would expose credentials in logs
tflog.Debug(ctx, "Account created", map[string]any{
    "access_key": data.AccessKey.ValueString(), // NEVER
})
```

### 2. Schema-Level Sensitivity

All credential fields are marked `Sensitive: true`:

```go
"access_key": schema.StringAttribute{
    Computed:  true,
    Sensitive: true,  // masked in terraform show/plan output
},
```

This covers: provider access/secret keys, provider console password, account access/secret keys, and generated console passwords.

### 3. Environment Variable Support

All provider configuration supports environment variable overrides:

```bash
# IAM API
export SCALITY_ENDPOINT="http://scality.example.com"
export SCALITY_ACCESS_KEY="admin-access-key"
export SCALITY_SECRET_KEY="admin-secret-key"

# Console API
export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="mySuperPassword"

# TLS
export SCALITY_INSECURE_SKIP_VERIFY="true"
```

The provider reads config first, falls back to environment:

```go
endpoint := config.Endpoint.ValueString()
if endpoint == "" {
    endpoint = os.Getenv("SCALITY_ENDPOINT")
}
```

Benefits: credentials are not committed to version control, CI/CD integration is simpler, and follows 12-factor app principles.

### 4. TLS Configuration

TLS verification is enabled by default. The `insecure_skip_verify` option exists for development environments with self-signed certificates:

```go
if insecureSkipVerify {
    httpClient.Transport = &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true,
        },
    }
}
```

### 5. Token Cache Security

Console JWT tokens are cached with restrictive permissions:

```go
const filePermissions = 0600  // Owner read/write only

func (c *ConsoleClient) cacheToken(token string) error {
    cacheFile := c.getCacheFile()  // /tmp/.scality_console_token_<md5hash>
    return os.WriteFile(cacheFile, data, filePermissions)
}
```

Unique filenames per endpoint/username prevent collisions. Temporary directory is cleaned on reboot.

---

## Testing Strategy

### Acceptance Testing

Tests run against a real Scality instance with `TF_ACC=1`:

```go
func TestAccScalityAccount_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccScalityAccountConfig("test-account"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("scality_account.test", "name", "test-account"),
                    resource.TestCheckResourceAttrSet("scality_account.test", "id"),
                    resource.TestCheckResourceAttrSet("scality_account.test", "access_key"),
                ),
            },
        },
    })
}
```

### Manual Verification Workflow

Each resource has been verified with a full lifecycle:

1. `tofu plan` — verify plan output is correct
2. `tofu apply` — create the resource
3. `tofu plan` (again) — verify no drift
4. Verify in Scality UI — confirm resource exists
5. `tofu destroy` — delete the resource
6. Verify in Scality UI — confirm deletion

### Makefile Targets

```makefile
test:     go test ./... -v              # Unit tests
testacc:  TF_ACC=1 go test ./... -v     # Acceptance tests (requires real instance)
fmt:      gofmt -w .                     # Format
build:    go build ./...                 # Build
install:  # Install to ~/.terraform.d/plugins/
```

---

## Future Extensibility

### Adding More Resources

Pattern is established — for each new resource:
1. Add client methods to `internal/client/iam.go` or `internal/client/console.go`
2. Create `internal/resources/<name>/model.go` and `resource.go`
3. Register in `internal/provider/provider.go`

### Data Sources

Currently no data sources are implemented. Future candidates:

```go
func (p *ScalityProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // data.scality_account — look up existing account
        // data.scality_account_list — list all accounts
    }
}
```

### Client Enhancements

Adding features to the helper methods benefits all operations automatically:

```go
// Future: Add retry logic to doSignedRequest
func (c *ScalityClient) doSignedRequest(...) ([]byte, int, error) {
    for attempt := 0; attempt < maxRetries; attempt++ {
        body, status, err := c.executeRequest(...)
        if err == nil && status < 500 {
            return body, status, nil
        }
        time.Sleep(backoff(attempt))
    }
    return nil, 0, fmt.Errorf("max retries exceeded: %w", lastErr)
}
// All IAM operations get retry automatically.
```

---

## Complexity Management Rules

### When to Abstract

Extract when:
- Used 3+ times
- Changes together always
- Clear single responsibility
- Reduces complexity

Don't extract when:
- Used only twice
- Needs many parameters
- Changes independently
- Abstraction is more complex than duplication

### Complexity Budget

Target:
- Client method: < 50 lines
- Resource method: < 80 lines
- Helper function: < 30 lines

If exceeded:
1. Can you extract a helper?
2. Can you simplify the logic?
3. Is this inherently complex? (document well)

### Code Review Checklist

When adding new code:

- [ ] Uses `context.Context` as first parameter
- [ ] Constants defined for magic values
- [ ] Errors wrapped with `%w`
- [ ] Sensitive data marked in schema
- [ ] HTTP requests use `NewRequestWithContext`
- [ ] Follows existing patterns (IAM helper or Console token check)
- [ ] No credentials logged
- [ ] Resource follows model.go + resource.go structure
- [ ] Mutable fields use `RequiresReplace` (if API doesn't support update)

---

## Summary

### Key Takeaways

1. **Two API surfaces, two clients** — IAM (SigV4/JSON) and Console (JWT/JSON), bundled in ProviderClients
2. **DRY with pragmatism** — Helper methods where patterns are identical; separate when abstraction adds complexity
3. **Context-first** — All public methods accept context for cancellation, timeouts, and logging
4. **Constants over magic values** — No hardcoded strings
5. **Atomic Create** — Save state before key generation to prevent orphaned resources
6. **RequiresReplace everywhere** — All mutable fields force replacement since APIs don't support in-place updates
7. **Consistent resource structure** — Every resource is model.go + resource.go in its own package
8. **Sensitive field preservation** — UseStateForUnknown + state-only reads prevent false drift on credentials

Design Philosophy:

> "Make it work, make it right, make it fast" — in that order.

The goal is code that a new developer can understand in 30 minutes and confidently modify.

---

Document Version: 2.0
Last Updated: 2026-04-11
Status: Living Document (update as architecture evolves)
