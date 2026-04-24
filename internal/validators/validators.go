package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type stringLengthBetween struct {
	min int
	max int
}

func (v stringLengthBetween) Description(_ context.Context) string {
	return fmt.Sprintf("must be between %d and %d characters", v.min, v.max)
}

func (v stringLengthBetween) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringLengthBetween) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if len(val) < v.min || len(val) > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Length",
			fmt.Sprintf("must be between %d and %d characters, got %d", v.min, v.max, len(val)),
		)
	}
}

type stringMatchRegex struct {
	regex   *regexp.Regexp
	message string
}

func (v stringMatchRegex) Description(_ context.Context) string {
	return v.message
}

func (v stringMatchRegex) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringMatchRegex) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if !v.regex.MatchString(val) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Value",
			fmt.Sprintf("%s, got: %q", v.message, val),
		)
	}
}

type stringOneOf struct {
	values []string
}

func (v stringOneOf) Description(_ context.Context) string {
	return fmt.Sprintf("must be one of: %s", strings.Join(v.values, ", "))
}

func (v stringOneOf) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringOneOf) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	for _, allowed := range v.values {
		if val == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Value",
		fmt.Sprintf("must be one of: %s, got: %q", strings.Join(v.values, ", "), val),
	)
}

type jsonString struct{}

func (v jsonString) Description(_ context.Context) string {
	return "must be valid JSON"
}

func (v jsonString) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonString) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if !json.Valid([]byte(val)) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid JSON",
			"must be a valid JSON document",
		)
	}
}

type int64AtLeast struct {
	min int64
}

func (v int64AtLeast) Description(_ context.Context) string {
	return fmt.Sprintf("must be at least %d", v.min)
}

func (v int64AtLeast) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int64AtLeast) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueInt64()
	if val < v.min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Value",
			fmt.Sprintf("must be at least %d, got %d", v.min, val),
		)
	}
}

var (
	accountNameRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	bucketNameRegex  = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]*[a-z0-9]$`)
	iamNameRegex     = regexp.MustCompile(`^[a-zA-Z0-9_+=,.@-]+$`)
	emailRegex       = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	arnRegex         = regexp.MustCompile(`^arn:aws:iam::[^:]*:policy/.+$`)
)

func AccountName() []validator.String {
	return []validator.String{
		stringLengthBetween{min: 1, max: 128},
		stringMatchRegex{regex: accountNameRegex, message: "must contain only alphanumeric characters and hyphens"},
	}
}

func BucketName() []validator.String {
	return []validator.String{
		stringLengthBetween{min: 3, max: 63},
		stringMatchRegex{regex: bucketNameRegex, message: "must contain only lowercase letters, numbers, hyphens, and periods, and must start and end with a letter or number"},
	}
}

func Email() []validator.String {
	return []validator.String{
		stringMatchRegex{regex: emailRegex, message: "must be a valid email address"},
	}
}

func IAMName(maxLen int) []validator.String {
	return []validator.String{
		stringLengthBetween{min: 1, max: maxLen},
		stringMatchRegex{regex: iamNameRegex, message: "must contain only alphanumeric characters and _+=,.@-"},
	}
}

func PolicyARN() []validator.String {
	return []validator.String{
		stringMatchRegex{regex: arnRegex, message: "must be a valid IAM policy ARN (arn:aws:iam::*:policy/*)"},
	}
}

func JSONDocument() []validator.String {
	return []validator.String{
		jsonString{},
	}
}

func OneOf(values ...string) []validator.String {
	return []validator.String{
		stringOneOf{values: values},
	}
}

func Int64AtLeast(min int64) []validator.Int64 {
	return []validator.Int64{
		int64AtLeast{min: min},
	}
}
