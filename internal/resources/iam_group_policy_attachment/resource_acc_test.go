package iamgrouppolicyattachment_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
	"github.com/cmrh/terraform-provider-scality/internal/client"
)

const resourceName = "scality_iam_group_policy_attachment.test"

func testAccConfig(name string) string {
	return acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
}

resource "scality_iam_policy" "test" {
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
  policy_name        = "%[1]s-policy"
  policy_document    = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":\"s3:Get*\",\"Resource\":\"*\"}]}"
}

resource "scality_group" "test" {
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
  group_name         = "%[1]s-group"
}

resource "scality_iam_group_policy_attachment" "test" {
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
  group_name         = scality_group.test.group_name
  policy_arn         = scality_iam_policy.test.arn
}
`, name)
}

// outOfBand runs fn against the attachment's group and policy, simulating a change made outside Terraform.
func outOfBand(fn func(ctx context.Context, c *client.IAMClient, ak, sk, group, arn string) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not in state", resourceName)
		}
		a := rs.Primary.Attributes
		c := client.NewIAMClient(os.Getenv("SCALITY_ENDPOINT"), "", "", true)
		return fn(context.Background(), c, a["account_access_key"], a["account_secret_key"], a["group_name"], a["policy_arn"])
	}
}

// TestAccIAMGroupPolicyAttachment_basic also covers destroy: the group and
// policy can only be deleted once the attachment is detached.
func TestAccIAMGroupPolicyAttachment_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "group_name", name+"-group"),
					resource.TestCheckResourceAttrPair(resourceName, "policy_arn", "scality_iam_policy.test", "arn"),
					outOfBand(func(ctx context.Context, c *client.IAMClient, ak, sk, group, arn string) error {
						policies, err := c.ListAttachedGroupPolicies(ctx, ak, sk, group)
						if err != nil {
							return err
						}
						if len(policies) != 1 || policies[0].PolicyArn != arn {
							return fmt.Errorf("want %s attached, got %+v", arn, policies)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc(resourceName, "group_name", "policy_arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "group_name",
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, resourceName, "group_name", "policy_arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "group_name",
			},
		},
	})
}

func TestAccIAMGroupPolicyAttachment_detachedOutOfBand(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(name),
				Check: outOfBand(func(ctx context.Context, c *client.IAMClient, ak, sk, group, arn string) error {
					return c.DetachGroupPolicy(ctx, ak, sk, group, arn)
				}),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccIAMGroupPolicyAttachment_parentGroupDeletedOutOfBand(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(name),
				Check: outOfBand(func(ctx context.Context, c *client.IAMClient, ak, sk, group, arn string) error {
					if err := c.DetachGroupPolicy(ctx, ak, sk, group, arn); err != nil {
						return err
					}
					return c.DeleteGroup(ctx, ak, sk, group)
				}),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
