package planmodifiers_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cmrh/terraform-provider-scality/internal/planmodifiers"
)

func TestSemanticJSON(t *testing.T) {
	const compact = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	const pretty = "{\n  \"Statement\": [\n    {\"Resource\": \"*\", \"Effect\": \"Allow\", \"Action\": \"s3:*\"}\n  ],\n  \"Version\": \"2012-10-17\"\n}"
	const changed = `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:*","Resource":"*"}]}`

	cases := []struct {
		name          string
		state, config types.String
		wantState     bool
	}{
		{"formatting only", types.StringValue(compact), types.StringValue(pretty), true},
		{"real change", types.StringValue(compact), types.StringValue(changed), false},
		{"create", types.StringNull(), types.StringValue(pretty), false},
		{"unknown config", types.StringValue(compact), types.StringUnknown(), false},
		{"invalid config", types.StringValue(compact), types.StringValue(`{`), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{StateValue: tc.state, ConfigValue: tc.config, PlanValue: tc.config}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			planmodifiers.SemanticJSON().PlanModifyString(context.Background(), req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			want := tc.config
			if tc.wantState {
				want = tc.state
			}
			if !resp.PlanValue.Equal(want) {
				t.Fatalf("plan = %v, want %v", resp.PlanValue, want)
			}
		})
	}
}
