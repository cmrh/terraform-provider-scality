package iamuserpolicyattachment_test

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

const resourceName = "scality_iam_user_policy_attachment.test"

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

resource "scality_user" "test" {
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
  username           = "%[1]s-user"
}

resource "scality_iam_user_policy_attachment" "test" {
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
  username           = scality_user.test.username
  policy_arn         = scality_iam_policy.test.arn
}
`, name)
}

// outOfBand runs fn against the attachment's user and policy, simulating a change made outside Terraform.
func outOfBand(fn func(ctx context.Context, c *client.IAMClient, ak, sk, user, arn string) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not in state", resourceName)
		}
		a := rs.Primary.Attributes
		c := client.NewIAMClient(os.Getenv("SCALITY_ENDPOINT"), "", "", true)
		return fn(context.Background(), c, a["account_access_key"], a["account_secret_key"], a["username"], a["policy_arn"])
	}
}

// TestAccIAMUserPolicyAttachment_basic also covers destroy: the user and
// policy can only be deleted once the attachment is detached.
func TestAccIAMUserPolicyAttachment_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", name+"-user"),
					resource.TestCheckResourceAttrPair(resourceName, "policy_arn", "scality_iam_policy.test", "arn"),
					outOfBand(func(ctx context.Context, c *client.IAMClient, ak, sk, user, arn string) error {
						policies, err := c.ListAttachedUserPolicies(ctx, ak, sk, user)
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
				ImportStateIdFunc:                    acctest.ImportStateIdFunc(resourceName, "username", "policy_arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "username",
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, resourceName, "username", "policy_arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "username",
			},
		},
	})
}

func TestAccIAMUserPolicyAttachment_detachedOutOfBand(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(name),
				Check: outOfBand(func(ctx context.Context, c *client.IAMClient, ak, sk, user, arn string) error {
					return c.DetachUserPolicy(ctx, ak, sk, user, arn)
				}),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccIAMUserPolicyAttachment_parentUserDeletedOutOfBand(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(name),
				Check: outOfBand(func(ctx context.Context, c *client.IAMClient, ak, sk, user, arn string) error {
					if err := c.DetachUserPolicy(ctx, ak, sk, user, arn); err != nil {
						return err
					}
					return c.DeleteUser(ctx, ak, sk, user)
				}),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
