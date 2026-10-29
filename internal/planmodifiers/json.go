package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type semanticJSON struct{}

// SemanticJSON plans the state value when the configured JSON is semantically
// equal to it, so formatting-only differences (e.g. after import) plan no change.
// Place it before RequiresReplace.
func SemanticJSON() planmodifier.String {
	return semanticJSON{}
}

func (m semanticJSON) Description(_ context.Context) string {
	return "keeps the state value when the configured JSON is semantically equal to it"
}

func (m semanticJSON) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m semanticJSON) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	// Invalid JSON is left to the attribute's own validation.
	eq, diags := jsontypes.NewNormalizedValue(req.ConfigValue.ValueString()).StringSemanticEquals(ctx, jsontypes.NewNormalizedValue(req.StateValue.ValueString()))
	if !diags.HasError() && eq {
		resp.PlanValue = req.StateValue
	}
}
