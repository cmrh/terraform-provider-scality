package validators

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runStringValidators(t *testing.T, validators []validator.String, value types.String, expectError bool) {
	t.Helper()
	anyError := false
	for _, v := range validators {
		req := validator.StringRequest{
			Path:        path.Root("test"),
			ConfigValue: value,
		}
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			anyError = true
		}
	}
	if expectError && !anyError {
		t.Errorf("expected error for %q, got none", value.ValueString())
	}
	if !expectError && anyError {
		t.Errorf("unexpected error for %q", value.ValueString())
	}
}

func runInt64Validators(t *testing.T, validators []validator.Int64, value types.Int64, expectError bool) {
	t.Helper()
	anyError := false
	for _, v := range validators {
		req := validator.Int64Request{
			Path:        path.Root("test"),
			ConfigValue: value,
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			anyError = true
		}
	}
	if expectError && !anyError {
		t.Errorf("expected error for %d, got none", value.ValueInt64())
	}
	if !expectError && anyError {
		t.Errorf("unexpected error for %d", value.ValueInt64())
	}
}

func TestAccountName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "my-account", false},
		{"valid alphanumeric", "Account123", false},
		{"valid single char", "a", false},
		{"valid max length", strings.Repeat("a", 128), false},
		{"valid with hyphens", "my-test-account-1", false},
		{"invalid empty", "", true},
		{"invalid too long", strings.Repeat("a", 129), true},
		{"invalid underscore", "my_account", true},
		{"invalid space", "my account", true},
		{"invalid special chars", "my@account!", true},
		{"invalid period", "my.account", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := AccountName()
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestAccountName_NullAndUnknown(t *testing.T) {
	v := AccountName()
	runStringValidators(t, v, types.StringNull(), false)
	runStringValidators(t, v, types.StringUnknown(), false)
}

func TestBucketName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "my-bucket", false},
		{"valid with periods", "my.bucket.name", false},
		{"valid min length", "abc", false},
		{"valid max length", strings.Repeat("a", 63), false},
		{"valid numbers", "bucket123", false},
		{"valid mixed", "my-bucket.v2", false},
		{"invalid too short", "ab", true},
		{"invalid too long", strings.Repeat("a", 64), true},
		{"invalid uppercase", "My-Bucket", true},
		{"invalid start with hyphen", "-bucket", true},
		{"invalid end with hyphen", "bucket-", true},
		{"invalid start with period", ".bucket", true},
		{"invalid end with period", "bucket.", true},
		{"invalid underscore", "my_bucket", true},
		{"invalid space", "my bucket", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := BucketName()
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestBucketName_NullAndUnknown(t *testing.T) {
	v := BucketName()
	runStringValidators(t, v, types.StringNull(), false)
	runStringValidators(t, v, types.StringUnknown(), false)
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "user@example.com", false},
		{"valid with dots", "first.last@example.com", false},
		{"valid with plus", "user+tag@example.com", false},
		{"valid with subdomain", "user@sub.example.com", false},
		{"valid with percent", "user%tag@example.com", false},
		{"invalid no at", "userexample.com", true},
		{"invalid no domain", "user@", true},
		{"invalid no tld", "user@example", true},
		{"invalid double at", "user@@example.com", true},
		{"invalid space", "user @example.com", true},
		{"invalid single char tld", "user@example.c", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Email()
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestIAMName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		maxLen  int
		wantErr bool
	}{
		{"valid simple", "myuser", 64, false},
		{"valid with allowed special chars", "user_+=,.@-test", 64, false},
		{"valid single char", "a", 64, false},
		{"valid at max length", strings.Repeat("a", 64), 64, false},
		{"valid at max 128", strings.Repeat("a", 128), 128, false},
		{"invalid empty", "", 64, true},
		{"invalid too long", strings.Repeat("a", 65), 64, true},
		{"invalid space", "my user", 64, true},
		{"invalid exclamation", "user!", 64, true},
		{"invalid hash", "user#1", 64, true},
		{"invalid colon", "user:name", 64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := IAMName(tt.maxLen)
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestIAMName_NullAndUnknown(t *testing.T) {
	v := IAMName(64)
	runStringValidators(t, v, types.StringNull(), false)
	runStringValidators(t, v, types.StringUnknown(), false)
}

func TestPolicyARN(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid standard", "arn:aws:iam::123456789012:policy/MyPolicy", false},
		{"valid with path", "arn:aws:iam::123456789012:policy/path/MyPolicy", false},
		{"valid empty account", "arn:aws:iam:::policy/MyPolicy", false},
		{"invalid missing policy prefix", "arn:aws:iam::123456789012:role/MyRole", true},
		{"invalid no policy name", "arn:aws:iam::123456789012:policy/", true},
		{"invalid wrong service", "arn:aws:s3:::my-bucket", true},
		{"invalid random string", "not-an-arn", true},
		{"invalid empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := PolicyARN()
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestJSONDocument(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid object", `{"key": "value"}`, false},
		{"valid array", `[1, 2, 3]`, false},
		{"valid nested", `{"a": {"b": [1, 2]}}`, false},
		{"valid string", `"hello"`, false},
		{"valid number", `42`, false},
		{"valid boolean", `true`, false},
		{"valid null", `null`, false},
		{"valid policy doc", `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`, false},
		{"invalid empty", ``, true},
		{"invalid truncated", `{"key":`, true},
		{"invalid plain text", `not json at all`, true},
		{"invalid trailing comma", `{"key": "value",}`, true},
		{"invalid single quotes", `{'key': 'value'}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := JSONDocument()
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestJSONDocument_NullAndUnknown(t *testing.T) {
	v := JSONDocument()
	runStringValidators(t, v, types.StringNull(), false)
	runStringValidators(t, v, types.StringUnknown(), false)
}

func TestOneOf(t *testing.T) {
	allowed := []string{"Enabled", "Disabled"}
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid first", "Enabled", false},
		{"valid second", "Disabled", false},
		{"invalid lowercase", "enabled", true},
		{"invalid other", "Suspended", true},
		{"invalid empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := OneOf(allowed...)
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestOneOf_ACLValues(t *testing.T) {
	allowed := []string{"private", "public-read", "public-read-write", "authenticated-read"}
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid private", "private", false},
		{"valid public-read", "public-read", false},
		{"valid public-read-write", "public-read-write", false},
		{"valid authenticated-read", "authenticated-read", false},
		{"invalid bucket-owner-full-control", "bucket-owner-full-control", true},
		{"invalid uppercase", "Private", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := OneOf(allowed...)
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestOneOf_SSEAlgorithm(t *testing.T) {
	allowed := []string{"AES256", "aws:kms"}
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid AES256", "AES256", false},
		{"valid aws:kms", "aws:kms", false},
		{"invalid lowercase aes", "aes256", true},
		{"invalid other", "aws:kms:s3", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := OneOf(allowed...)
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestOneOf_RetentionMode(t *testing.T) {
	allowed := []string{"GOVERNANCE", "COMPLIANCE"}
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid GOVERNANCE", "GOVERNANCE", false},
		{"valid COMPLIANCE", "COMPLIANCE", false},
		{"invalid lowercase", "governance", true},
		{"invalid mixed case", "Governance", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := OneOf(allowed...)
			runStringValidators(t, v, types.StringValue(tt.value), tt.wantErr)
		})
	}
}

func TestInt64AtLeast(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		min     int64
		wantErr bool
	}{
		{"valid at min", 16, 16, false},
		{"valid above min", 32, 16, false},
		{"valid large", 1000, 1, false},
		{"valid zero when min zero", 0, 0, false},
		{"invalid below min", 15, 16, true},
		{"invalid zero when min positive", 0, 1, true},
		{"invalid negative", -1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Int64AtLeast(tt.min)
			runInt64Validators(t, v, types.Int64Value(tt.value), tt.wantErr)
		})
	}
}

func TestInt64AtLeast_NullAndUnknown(t *testing.T) {
	v := Int64AtLeast(16)
	runInt64Validators(t, v, types.Int64Null(), false)
	runInt64Validators(t, v, types.Int64Unknown(), false)
}

func TestDescriptions(t *testing.T) {
	ctx := context.Background()

	strTests := []struct {
		name       string
		validators []validator.String
		contains   string
	}{
		{"AccountName length", AccountName()[:1], "between 1 and 128"},
		{"AccountName regex", AccountName()[1:], "alphanumeric"},
		{"BucketName length", BucketName()[:1], "between 3 and 63"},
		{"Email", Email(), "email"},
		{"IAMName length", IAMName(64)[:1], "between 1 and 64"},
		{"PolicyARN", PolicyARN(), "ARN"},
		{"JSONDocument", JSONDocument(), "JSON"},
		{"OneOf", OneOf("a", "b"), "one of"},
	}
	for _, tt := range strTests {
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range tt.validators {
				desc := v.Description(ctx)
				if !strings.Contains(strings.ToLower(desc), strings.ToLower(tt.contains)) {
					t.Errorf("Description() = %q, want it to contain %q", desc, tt.contains)
				}
				md := v.MarkdownDescription(ctx)
				if desc != md {
					t.Errorf("MarkdownDescription() = %q, want same as Description() = %q", md, desc)
				}
			}
		})
	}

	int64V := Int64AtLeast(10)
	for _, v := range int64V {
		desc := v.Description(ctx)
		if !strings.Contains(desc, "10") {
			t.Errorf("Int64AtLeast Description() = %q, want it to contain \"10\"", desc)
		}
		md := v.MarkdownDescription(ctx)
		if desc != md {
			t.Errorf("Int64AtLeast MarkdownDescription() = %q, want same as Description() = %q", md, desc)
		}
	}
}
