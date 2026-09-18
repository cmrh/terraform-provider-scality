package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	provschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

func getProviderSchema(t *testing.T) provschema.Schema {
	t.Helper()
	p := New("test")()
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider Schema() returned errors: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestProviderSchema_AssumeRoleBlock(t *testing.T) {
	s := getProviderSchema(t)

	block, ok := s.Blocks["assume_role"]
	if !ok {
		t.Fatal("assume_role block not found in provider schema")
	}

	nested, ok := block.(provschema.SingleNestedBlock)
	if !ok {
		t.Fatalf("assume_role is not a SingleNestedBlock, got %T", block)
	}

	// role_arn is intentionally Optional, not Required: a Required attribute
	// inside a SingleNestedBlock forces the block onto every provider config
	// (Terraform errors "Missing Configuration for Required Attribute" when the
	// block is omitted). Presence is enforced at Configure time instead.
	roleArn, ok := nested.Attributes["role_arn"].(provschema.StringAttribute)
	if !ok {
		t.Fatal("assume_role.role_arn is not a StringAttribute")
	}
	if roleArn.Required {
		t.Error("assume_role.role_arn must be Optional, not Required (a Required attr in a SingleNestedBlock forces the block onto every config)")
	}
	if !roleArn.Optional {
		t.Error("assume_role.role_arn must be Optional")
	}

	sessionName, ok := nested.Attributes["session_name"].(provschema.StringAttribute)
	if !ok {
		t.Fatal("assume_role.session_name is not a StringAttribute")
	}
	if !sessionName.Optional {
		t.Error("assume_role.session_name must be Optional")
	}
	if sessionName.Required {
		t.Error("assume_role.session_name must not be Required")
	}
}
