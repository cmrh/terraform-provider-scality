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
│                    Terraform Core                           │
│           (Handles plan, apply, destroy lifecycle)          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ Plugin Protocol (gRPC)
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Scality Terraform Provider                     │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Provider   │  │  Resources   │  │ Data Sources │    │
│  │  (provider.go)│  │              │  │  (future)    │    │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘    │
│         │                  │                               │
│         ▼                  ▼                               │
│  ┌──────────────────────────────────────────────┐         │
│  │         Client Layer (Abstraction)           │         │
│  │                                              │         │
│  │  ┌────────────────┐  ┌────────────────────┐│         │
│  │  │  ScalityClient │  │  ConsoleClient     ││         │
│  │  │  (IAM API)     │  │  (Console API)     ││         │
│  │  └────────┬───────┘  └────────┬───────────┘│         │
│  │           │                    │            │         │
│  └───────────┼────────────────────┼────────────┘         │
│              │                    │                       │
└──────────────┼────────────────────┼───────────────────────┘
               │                    │
               ▼                    ▼
    ┌──────────────────┐  ┌──────────────────┐
    │   Scality IAM    │  │ Scality Console  │
    │      API         │  │      API         │
    │ (AWS Sig V4)     │  │  (JWT Token)     │
    └──────────────────┘  └──────────────────┘
```

### Key Components

| Component | Purpose | Files |
|-----------|---------|-------|
| **Provider** | Configuration entry point | `provider.go` |
| **IAM Client** | AWS Signature V4 API communication | `client.go` |
| **Console Client** | JWT-based API communication | `console_client.go` |
| **Resources** | Terraform resource implementations | `resource_*.go` |
| **Data Sources** | Read-only data retrieval (future) | `data_source_*.go` |

---

## Design Principles

### 1. Separation of Concerns

Decision: Split IAM and Console clients into separate files.

Reasoning:
- Different authentication mechanisms (AWS Sig V4 vs JWT)
- Different API patterns and endpoints
- Independent evolution of each API
- Easier to maintain and test

Example:
```
client.go          → IAM API (Accounts, IAM policies, etc.)
console_client.go  → Console API (Users, permissions, etc.)
```

Benefits: Clear responsibility boundaries, no coupling between different API types, easy to add third API client if needed.

---

### 2. DRY Principle with Pragmatism

Decision: Apply DRY where it reduces complexity, not religiously.

#### Where We Applied DRY

**1. Shared Constants**
```go
// Used by both clients
const (
    httpMethodPost     = "POST"
    defaultHTTPTimeout = 30 * time.Second
)
```

Reasoning: Changes once, benefits everywhere.

**2. Helper Method in IAM Client**
```go
func (c *ScalityClient) doSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error)
```

Reasoning: Same request pattern for all IAM API calls. Reduces 160 lines of duplication to 40-line helper. Single place to add features (retry, metrics, etc.).

#### Where We Didn't Apply DRY

**1. Console Client HTTP Calls**

Decision: Did NOT create a shared helper between IAM and Console clients.

Reasoning:
```go
// IAM uses AWS Signature V4 signing
headers, err := c.signRequest(method, url, payload)

// Console uses JWT token
httpReq.Header.Set("x-access-token", c.token)

// Different enough that abstraction would add complexity
```

Rule of Thumb:
- Duplicate when abstractions would be more complex than the duplication
- Abstract when the pattern is identical and changes together

---

### 3. Context-First Design

Decision: Every public API method accepts `context.Context` as the first parameter.

Signature Pattern:
```go
func (c *Client) MethodName(ctx context.Context, params...) (result, error)
```

Reasoning:

1. Cancellation Support
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()

   account, err := client.CreateAccount(ctx, req)
   // Automatically cancelled after 5 seconds
   ```

2. Integration with Terraform
   - Terraform provides context to resources
   - Propagating it through enables proper lifecycle management

3. Future-Proofing
   - Distributed tracing (OpenTelemetry)
   - Request cancellation
   - Deadline propagation

4. Standard Go Practice
   - Idiomatic since Go 1.7
   - Expected by Go developers

Example Flow:
```
Terraform → Resource.Create(ctx) → Client.CreateAccount(ctx) → HTTP Request (with context)
                                                                         ↓
                                                        Respects timeout/cancellation
```

---

### 4. Constants Over Magic Values

Decision: All configuration values are package-level constants.

Before (Magic Values):
```go
params.Set("Version", "2010-05-08")
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
if tokenAge >= 84600 {
```

After (Constants):
```go
const (
    apiVersion      = "2010-05-08"
    contentTypeForm = "application/x-www-form-urlencoded"
    tokenSafetyMargin = 84600
)

params.Set("Version", apiVersion)
req.Header.Set("Content-Type", contentTypeForm)
if tokenAge >= tokenSafetyMargin {
```

Benefits:

| Benefit | Example |
|---------|---------|
| Single source of truth | Update API version in one place |
| Self-documenting | `tokenSafetyMargin` explains purpose |
| Type safety | Compiler catches typos |
| Easy refactoring | Find all usages via IDE |

Placement Strategy:
```go
// Group by purpose
const (
    // AWS Signature V4 constants
    awsService = "iam"
    awsRegion  = "us-east-1"
)

const (
    // API constants
    apiVersion = "2010-05-08"
)

const (
    // HTTP constants
    httpMethodPost = "POST"
)
```

---

## Client Architecture

### IAM Client Design (`client.go`)

#### Key Components

```go
type ScalityClient struct {
    Endpoint   string
    AccessKey  string
    SecretKey  string
    HTTPClient *http.Client
}
```

Design Decision: Simple struct with minimal fields.

Reasoning: Easy to instantiate, clear dependencies, immutable after creation, no hidden state.

#### The Helper Method Pattern

Core Design:
```go
// Private helper - handles HTTP mechanics
func (c *ScalityClient) doSignedRequest(
    ctx context.Context,
    action string,
    params url.Values,
) ([]byte, int, error) {
    // 1. Set common parameters
    params.Set("Action", action)
    params.Set("Version", apiVersion)

    // 2. Sign request
    // 3. Create HTTP request with context
    // 4. Execute
    // 5. Return body, status, error
}

// Public methods - handle business logic
func (c *ScalityClient) CreateAccount(
    ctx context.Context,
    req AccountCreateRequest,
) (*AccountCreateResponse, error) {
    // 1. Build parameters
    params := url.Values{}
    params.Set("name", req.Name)

    // 2. Call helper
    body, statusCode, err := c.doSignedRequest(ctx, "CreateAccount", params)

    // 3. Handle response codes
    if statusCode == 409 {
        return nil, fmt.Errorf("account already exists")
    }

    // 4. Parse and return
}
```

Why This Pattern Works:

1. **Clear Separation**
   - Helper: HTTP/network concerns
   - Method: Business logic

2. **Easy to Extend**
   ```go
   // Adding retry logic? Just update helper:
   func (c *ScalityClient) doSignedRequest(...) {
       for attempt := 0; attempt < maxRetries; attempt++ {
           // existing code
       }
   }
   // All methods get retry automatically!
   ```

3. **Testability**
   - Mock HTTP client in tests
   - Test helper separately from business logic

4. **Consistency**
   - All API calls follow same pattern
   - New developers know what to expect

---

### Console Client Design (`console_client.go`)

#### Token Management

**Design Decision**: Built-in token caching with expiration.

```go
type ConsoleClient struct {
    Endpoint   string
    Username   string
    Password   string
    HTTPClient *http.Client
    token      string  // Cached token (private)
}
```

Key Methods:
```go
// Automatic token management
func (c *ConsoleClient) Authenticate(ctx context.Context) error {
    // 1. Try cached token first
    if cachedToken, err := c.getCachedToken(); err == nil {
        c.token = cachedToken
        return nil
    }

    // 2. Authenticate and cache
}

// Called automatically by public methods
func (c *ConsoleClient) CreateConsoleAccount(ctx context.Context, ...) {
    if c.token == "" {
        if err := c.Authenticate(ctx); err != nil {
            return nil, err
        }
    }
    // Use token...
}
```

Why This Design:

1. **Automatic Token Refresh**
   - Methods check token existence
   - Authenticate on-demand
   - No manual token management

2. **Performance**
   - Cache tokens for 23.5 hours
   - Avoid re-authentication on every call
   - File-based cache survives provider restarts

3. **Security**
   - Cache files have 0600 permissions
   - Unique cache per endpoint/username
   - Automatic cleanup on expiry

4. **Simplicity for Callers**
   ```go
   // Users don't think about tokens
   client := NewConsoleClient(endpoint, user, pass)
   account, err := client.CreateConsoleAccount(ctx, req)
   // Token handled automatically
   ```

#### Why NOT Use Helper Method Here?

Decision: Console client doesn't use `doSignedRequest()` pattern.

Reasoning:

1. **Different Authentication**
   - Each request needs token header, not signature
   - No signing process involved
   - Simpler pattern doesn't warrant abstraction

2. **Varied Endpoints**
   ```go
   // Different URL patterns
   POST /_/console/vault/accounts        // Create
   POST /_/console/vault/accounts/{id}/keys  // Generate keys
   DELETE /_/console/vault/accounts/{id}     // Delete account
   DELETE /_/console/vault/accounts/{id}/user // Delete user
   ```

3. **Method-Specific Logic**
   - `DeleteConsoleAccount` is two-step process
   - Different status code handling per endpoint
   - Helper would need too many parameters

Guideline:
- If methods are 70% similar: Extract helper
- If methods are 40% similar: Keep separate
- Console client is ~40% similar: Kept separate

---

## Resource Pattern

### Terraform Resource Lifecycle

```go
type ScalityAccountResource struct {
    client *ScalityClient
}

// Terraform calls these in order:
Create(ctx, req, resp)  // terraform apply (new resource)
Read(ctx, req, resp)    // terraform refresh/plan
Update(ctx, req, resp)  // terraform apply (changes)
Delete(ctx, req, resp)  // terraform destroy
```

### Standard Resource Pattern

File Structure: `resource_<name>.go`

Example: `resource_account.go`

```go
// 1. Define schema
func (r *ScalityAccountResource) Schema(ctx context.Context, ...) schema.Schema {
    return schema.Schema{
        Attributes: map[string]schema.Attribute{
            "name": schema.StringAttribute{
                Required: true,
                // ...
            },
            // ...
        },
    }
}

// 2. Implement Create
func (r *ScalityAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // a. Read plan data
    var data AccountResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

    // b. Call client
    account, err := r.client.CreateAccount(ctx, createReq)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", ...)
        return
    }

    // c. Update state
    data.ID = types.StringValue(account.Account.Data.ID)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// 3. Implement Read (for drift detection)
func (r *ScalityAccountResource) Read(ctx context.Context, ...) {
    // Get current state from API
    // Update Terraform state
    // If not found, remove from state
}

// 4. Implement Delete
func (r *ScalityAccountResource) Delete(ctx context.Context, ...) {
    // Call delete API
    // Remove from state (automatic)
}
```

Design Decisions:

1. **No Update Method for Accounts**
   - Many Scality account fields are immutable
   - Changes require recreation (ForceNew)
   - Simpler than partial updates

2. **Sensitive Data Handling**
   ```go
   "access_key": schema.StringAttribute{
       Computed:  true,
       Sensitive: true,  // Hidden in terraform show
   },
   ```

3. **Error Handling**
   ```go
   // Don't return errors, add to diagnostics
   if err != nil {
       resp.Diagnostics.AddError(
           "Client Error",
           fmt.Sprintf("Unable to create account: %s", err),
       )
       return
   }
   ```

---

## Adding New API Calls

### Step-by-Step Guide

#### Scenario: Adding "List Accounts" Functionality

### Step 1: Add to Client

Choose the Right Client:
- IAM API call: `client.go`
- Console API call: `console_client.go`

For IAM API Example:

```go
// 1. Define response type (in client.go)
type AccountListResponse struct {
    Accounts []AccountData `json:"accounts"`
}

// 2. Add the method
func (c *ScalityClient) ListAccounts(ctx context.Context, maxResults int) (*AccountListResponse, error) {
    params := url.Values{}
    if maxResults > 0 {
        params.Set("MaxResults", fmt.Sprintf("%d", maxResults))
    }

    // Use the helper!
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

The helper handles:
- Setting Action and Version
- AWS Signature signing
- HTTP request creation with context
- Request execution
- Error handling

### Step 2: Add Data Source (Read-Only Resource)

Create `data_source_account_list.go`:

```go
package provider

type AccountListDataSource struct {
    client *ScalityClient
}

func (d *AccountListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_account_list"
}

func (d *AccountListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "List Scality accounts",
        Attributes: map[string]schema.Attribute{
            "accounts": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                            Computed: true,
                        },
                        "name": schema.StringAttribute{
                            Computed: true,
                        },
                        // ...
                    },
                },
            },
        },
    }
}

func (d *AccountListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data AccountListDataSourceModel

    // Call the client method we just created
    accounts, err := d.client.ListAccounts(ctx, 0)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list accounts: %s", err))
        return
    }

    // Map to Terraform data model
    for _, account := range accounts.Accounts {
        data.Accounts = append(data.Accounts, AccountModel{
            ID:   types.StringValue(account.Data.ID),
            Name: types.StringValue(account.Data.Name),
            // ...
        })
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

### Step 3: Register Data Source

In `provider.go`:

```go
func (p *ScalityProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        func() datasource.DataSource {
            return &AccountListDataSource{client: p.iamClient}
        },
        // Add more data sources...
    }
}
```

### Step 4: Usage Example

```hcl
# In user's Terraform code
data "scality_account_list" "all" {}

output "account_count" {
  value = length(data.scality_account_list.all.accounts)
}
```

---

### Adding Resource-Based API Calls

#### Example: Add "Update Account Quota"

```go
// 1. Add to client.go
func (c *ScalityClient) UpdateAccountQuota(ctx context.Context, accountName string, quotaMax int64) error {
    params := url.Values{}
    params.Set("AccountName", accountName)
    params.Set("QuotaMax", fmt.Sprintf("%d", quotaMax))

    body, statusCode, err := c.doSignedRequest(ctx, "UpdateAccountQuota", params)
    if err != nil {
        return err
    }

    if statusCode != 200 {
        return fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
    }

    return nil
}

// 2. Add Update method to resource_account.go
func (r *ScalityAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan, state AccountResourceModel

    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

    // Only quota changed?
    if plan.QuotaMax != state.QuotaMax {
        err := r.client.UpdateAccountQuota(
            ctx,
            plan.Name.ValueString(),
            plan.QuotaMax.ValueInt64(),
        )
        if err != nil {
            resp.Diagnostics.AddError("Update Error", err.Error())
            return
        }
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// 3. Update schema to allow updates
"quota_max": schema.Int64Attribute{
    Optional: true,
    // Remove ForceNew if present
},
```

---

### Complexity Decision Tree

When adding new functionality:

```
Is this an API call?
├─ Yes
│  ├─ IAM API (AWS Signature)?
│  │  └─ Add to client.go, use doSignedRequest helper
│  │     → Low complexity (helper does heavy lifting)
│  │
│  └─ Console API (JWT)?
│     └─ Add to console_client.go, check token first
│        → Medium complexity (handle authentication)
│
└─ No
   └─ Is this a Terraform resource?
      ├─ Read-only → Create data source
      └─ Create/Update/Delete → Create resource
```

---

## Error Handling Strategy

### Principles

1. **Wrap errors with context**
2. **Meaningful messages for users**
3. **Distinguish error types**
4. **Never swallow errors**

### Error Wrapping Pattern

```go
// Good: Wrap with context
if err != nil {
    return nil, fmt.Errorf("failed to create account: %w", err)
}

// Bad: Lose context
if err != nil {
    return nil, err
}

// Bad: Break error chain
if err != nil {
    return nil, fmt.Errorf("failed to create account: %s", err)
}
```

Why use `%w` instead of `%s`?

```go
err := client.CreateAccount(ctx, req)
// err wraps HTTP error wraps network error

// With %w: Can unwrap to find root cause
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout specifically
}

// With %s: Error chain is broken
// Can't detect what actually failed
```

### HTTP Status Code Handling

Pattern Used:

```go
body, statusCode, err := c.doSignedRequest(ctx, action, params)
if err != nil {
    return nil, err  // Network/signing error
}

// Handle specific status codes
switch statusCode {
case 200, 201:
    // Success, parse response
case 404:
    return nil, nil  // Resource doesn't exist (for Read)
case 409:
    return nil, fmt.Errorf("account already exists")
case 403:
    return nil, fmt.Errorf("permission denied: check credentials")
default:
    return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
}
```

Design Decision: Return status code from helper.

Different methods need different status handling. A 404 means different things (error in Create, ok in Read), so business logic decides how to interpret codes.

### User-Facing Error Messages

In Resources:

```go
if err != nil {
    resp.Diagnostics.AddError(
        "Client Error",  // Summary (shown bold)
        fmt.Sprintf("Unable to create account '%s': %s",
            data.Name.ValueString(),
            err,  // Detailed message
        ),
    )
    return
}
```

Special Case: Helpful Delete Errors:

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

This helps users solve problems without reading API docs.

---

## State Management

### Terraform State Basics

Terraform state is the record of infrastructure stored in `terraform.tfstate`. It maps resources to real-world objects. Our responsibility is to keep state accurate.

### Read Method: Drift Detection

```go
func (r *ScalityAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data AccountResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

    // Check if resource still exists
    account, err := r.client.GetAccount(ctx, data.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Client Error", ...)
        return
    }

    // Resource was deleted outside Terraform
    if account == nil {
        resp.State.RemoveResource(ctx)
        return
    }

    // Update state with current values
    data.QuotaMax = types.Int64Value(account.Data.QuotaMax)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

If someone deletes an account outside Terraform, the next `terraform plan` will detect the drift and plan to recreate it.

### Sensitive Data in State

Problem: Access keys are stored in the state file.

Solution: Mark as sensitive.

```go
"access_key": schema.StringAttribute{
    Computed:  true,
    Sensitive: true,  // Masked in output, but still in state
},
```

Important: State file itself must be secured.

```hcl
# terraform.tf - Secure state
terraform {
  backend "s3" {
    bucket         = "terraform-state"
    key            = "scality/terraform.tfstate"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}
```

Documentation should tell users to protect the state file.

---

## Security Patterns

### 1. Credential Handling

Never log credentials:

```go
// Good: Credentials stay private
tflog.Debug(ctx, "Creating account", map[string]interface{}{
    "name": data.Name.ValueString(),
    // DO NOT log access_key or secret_key
})

// Bad: Exposes credentials
tflog.Debug(ctx, "Account created", map[string]interface{}{
    "access_key": account.AccessKey,  // NEVER DO THIS
})
```

### 2. Token Caching Security

Console Client Token Cache:

```go
// Use secure permissions
const filePermissions = 0600  // Owner read/write only

if err := os.WriteFile(cacheFile, data, filePermissions); err != nil {
    return err
}
```

Cache Location:
```go
// Temporary directory (usually /tmp or C:\Temp)
cacheDir := os.TempDir()

// Unique filename using hash
cacheFile := filepath.Join(cacheDir, tokenCachePrefix+hash)
```

This design uses temporary directories that are cleaned on reboot, creates unique filenames per endpoint/username to avoid collisions, and sets restrictive permissions so only the current user can access tokens.

### 3. Environment Variable Support

Example configuration:
```bash
export SCALITY_ENDPOINT="http://10.164.169.247"
export SCALITY_ACCESS_KEY="..."
export SCALITY_SECRET_KEY="..."
```

Provider reads automatically:
```go
func (p *ScalityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    // Prefer config, fallback to env
    endpoint := config.Endpoint.ValueString()
    if endpoint == "" {
        endpoint = os.Getenv("SCALITY_ENDPOINT")
    }
    // ...
}
```

Benefits: credentials are not committed to version control, CI/CD integration is simpler, and follows 12-factor app principles.

---

## Testing Strategy

### Unit Testing Approach

Mock HTTP Client:

```go
// Create mock HTTP client
type MockTransport struct {
    Response *http.Response
    Err      error
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    return m.Response, m.Err
}

// Test example
func TestCreateAccount(t *testing.T) {
    // Setup mock
    mockResp := &http.Response{
        StatusCode: 201,
        Body:       io.NopCloser(strings.NewReader(`{"account":{"data":{"id":"123"}}}`)),
    }

    client := &ScalityClient{
        Endpoint:   "http://test",
        HTTPClient: &http.Client{Transport: &MockTransport{Response: mockResp}},
    }

    // Test
    ctx := context.Background()
    account, err := client.CreateAccount(ctx, req)

    assert.NoError(t, err)
    assert.Equal(t, "123", account.Account.Data.ID)
}
```

### Integration Testing

Acceptance Tests (with real API):

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

### Table-Driven Tests

Recommended Pattern:

```go
func TestHandleStatusCode(t *testing.T) {
    tests := []struct {
        name           string
        statusCode     int
        body           string
        expectedError  string
        expectedResult bool
    }{
        {
            name:           "Success 200",
            statusCode:     200,
            body:           `{"success":true}`,
            expectedError:  "",
            expectedResult: true,
        },
        {
            name:          "Conflict 409",
            statusCode:    409,
            body:          `{"error":"already exists"}`,
            expectedError: "already exists",
        },
        // More cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

---

## Future Extensibility

### Designed for Growth

#### 1. Adding More Resources

Current resources:
- `scality_account` (IAM)
- `scality_console_account` (Console)

Easy to add:
```
resource_bucket.go          → S3 bucket resource
resource_iam_user.go        → IAM user resource
resource_iam_policy.go      → IAM policy resource
resource_group.go           → User group resource
```

Pattern:
1. Add client method
2. Create resource file
3. Register in provider.go
4. Write tests

#### 2. Data Sources for Read Operations

Future data sources:
```go
data_source_account.go       → Read single account
data_source_account_list.go  → List all accounts
data_source_bucket.go        → Read bucket info
```

Usage:
```hcl
# Read existing account
data "scality_account" "existing" {
  name = "production"
}

# Use in other resources
resource "aws_iam_policy" "s3_access" {
  policy = jsonencode({
    Statement = [{
      Resource = "arn:aws:s3:::${data.scality_account.existing.canonical_id}/*"
    }]
  })
}
```

#### 3. Client Enhancements

Adding Features to Helper Method:

```go
// Current
func (c *ScalityClient) doSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error) {
    // ...existing code...
}

// Future: Add retry logic
func (c *ScalityClient) doSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error) {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        body, status, err := c.executeRequest(ctx, action, params)

        // Success
        if err == nil && status < 500 {
            return body, status, nil
        }

        // Retry on 5xx or network errors
        if status >= 500 || err != nil {
            lastErr = err
            time.Sleep(backoff(attempt))
            continue
        }

        return body, status, err
    }

    return nil, 0, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

All API calls get retry automatically when added to the helper.

#### 4. Observability

Future Enhancement:

```go
func (c *ScalityClient) doSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error) {
    // Add metrics
    start := time.Now()
    defer func() {
        metrics.RecordAPICall(action, time.Since(start))
    }()

    // Add tracing
    ctx, span := tracer.Start(ctx, "scality."+action)
    defer span.End()

    // Existing code...
}
```

Benefits: performance monitoring, distributed tracing, and error rate tracking can be added without changing resources.

#### 5. Alternative Authentication

Adding OAuth2 Support:

```go
// New client type
type OAuthClient struct {
    Endpoint     string
    ClientID     string
    ClientSecret string
    HTTPClient   *http.Client
    token        *oauth2.Token
}

// Implement same interface
func (c *OAuthClient) CreateAccount(ctx context.Context, req AccountCreateRequest) (*AccountCreateResponse, error) {
    // OAuth-specific implementation
}

// Resources can use either client
type ScalityAccountResource struct {
    client AccountCreator  // Interface, not concrete type
}

type AccountCreator interface {
    CreateAccount(ctx context.Context, req AccountCreateRequest) (*AccountCreateResponse, error)
    // ... other methods
}
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
2. Can you simplify logic?
3. Is this inherently complex? (then document well)

### Cognitive Load Reduction

Good:
```go
// Clear, linear flow
func CreateAccount(ctx context.Context, req Request) (*Response, error) {
    // 1. Build parameters
    params := buildParams(req)

    // 2. Make request
    body, status, err := c.doSignedRequest(ctx, "CreateAccount", params)
    if err != nil {
        return nil, err
    }

    // 3. Handle response
    return parseAccountResponse(body, status)
}
```

Bad:
```go
// Nested, hard to follow
func CreateAccount(ctx context.Context, req Request) (*Response, error) {
    if req.Name != "" {
        if req.Email != "" {
            params := url.Values{}
            if req.Quota > 0 {
                params.Set("quota", ...)
                if req.External != "" {
                    // 5 levels deep!
                }
            }
        }
    }
}
```

---

## Maintenance Guidelines

### Code Review Checklist

When adding new code:

- [ ] Uses context.Context as first parameter
- [ ] Constants defined for magic values
- [ ] Errors wrapped with `%w`
- [ ] Sensitive data marked in schema
- [ ] HTTP requests use `NewRequestWithContext`
- [ ] Follows existing patterns (IAM helper or Console token check)
- [ ] Includes godoc comments
- [ ] Tests added (unit or acceptance)

### Documentation Standards

Every Public Function:
```go
// CreateAccount creates a new Scality account with S3 credentials.
//
// The account is created via the IAM API and access keys are automatically
// generated. Returns AccountCreateResponse containing the account details
// and credentials, or an error if creation fails.
func (c *ScalityClient) CreateAccount(ctx context.Context, req AccountCreateRequest) (*AccountCreateResponse, error)
```

Complex Logic:
```go
// Check if token is expired
// Use safety margin (23.5 hours) to refresh before actual expiry (24 hours)
// This prevents race conditions where token expires during a request
tokenAge := time.Now().Unix() - int64(cache.Timestamp)
if tokenAge >= tokenSafetyMargin {
    _ = os.Remove(cacheFile)
    return "", fmt.Errorf("token expired")
}
```

### Versioning Strategy

When to Bump Version:

| Change | Version | Example |
|--------|---------|---------|
| Add resource | Minor (0.2.0) | Add `scality_bucket` |
| Add optional field | Patch (0.1.1) | Add `external_id` to account |
| Change required field | Major (2.0.0) | Remove `email` requirement |
| Bug fix | Patch (0.1.1) | Fix state drift issue |
| Breaking change | Major (2.0.0) | Change attribute names |

---

## Common Patterns Reference

### Adding New IAM API Method

```go
// 1. Define request/response types (if needed)
type NewFeatureRequest struct {
    Field1 string
    Field2 int
}

type NewFeatureResponse struct {
    Result string `json:"result"`
}

// 2. Add method using helper
func (c *ScalityClient) NewFeature(ctx context.Context, req NewFeatureRequest) (*NewFeatureResponse, error) {
    params := url.Values{}
    params.Set("Field1", req.Field1)
    params.Set("Field2", fmt.Sprintf("%d", req.Field2))

    body, statusCode, err := c.doSignedRequest(ctx, "ActionName", params)
    if err != nil {
        return nil, err
    }

    if statusCode != 200 {
        return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
    }

    var result NewFeatureResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    return &result, nil
}
```

### Adding New Console API Method

```go
func (c *ConsoleClient) NewFeature(ctx context.Context, param string) (*Response, error) {
    // 1. Ensure authenticated
    if c.token == "" {
        if err := c.Authenticate(ctx); err != nil {
            return nil, err
        }
    }

    // 2. Build URL
    url := fmt.Sprintf("%s%s/%s", c.Endpoint, consoleAccountPath, param)

    // 3. Create request
    httpReq, err := http.NewRequestWithContext(ctx, httpMethodPost, url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // 4. Set headers
    httpReq.Header.Set("Content-Type", consoleContentType)
    httpReq.Header.Set("x-access-token", c.token)

    // 5. Execute and handle response
    resp, err := c.HTTPClient.Do(httpReq)
    // ... standard response handling
}
```

---

## Summary

### Key Takeaways

1. Separation of Concerns
   - IAM client for AWS Signature V4 API
   - Console client for JWT-based API
   - Clean boundaries, independent evolution

2. DRY with Pragmatism
   - Helper methods where patterns are identical
   - Keep separate when abstraction adds complexity
   - Share constants, not necessarily code

3. Context-First Design
   - All public methods accept context
   - Enables cancellation and timeouts
   - Future-proof for tracing

4. Constants for Configuration
   - No magic values
   - Self-documenting code
   - Easy maintenance

5. Simple Error Handling
   - Wrap with context (`%w`)
   - Meaningful messages
   - Help users solve problems

6. Extensibility by Design
   - Easy to add new API calls
   - Pattern established for resources
   - Helper methods benefit all callers

Design Philosophy:

"Make it work, make it right, make it fast"

We prioritize:
1. Correctness (does it work?)
2. Maintainability (can we fix/extend it easily?)
3. Performance (is it fast enough?)

In that order.

Final Notes:

This architecture balances:
- Simplicity vs DRY: Simple wins when DRY adds complexity
- Abstraction vs Clarity: Clarity wins when abstraction obscures
- Features vs Complexity: Only add what's needed

The goal is code that a new developer can understand in 30 minutes and confidently modify.

---

Document Version: 1.0
Last Updated: 2024-01-13
Author: Technical Architecture Team
Status: Living Document (update as architecture evolves)

