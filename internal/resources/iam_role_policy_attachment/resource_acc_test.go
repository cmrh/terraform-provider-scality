package iamrolepolicyattachment_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

func TestAccIAMRolePolicyAttachment_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_iam_role_policy_attachment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_iam_role_policy_attachment"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}

resource "scality_iam_policy" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  policy_name        = "%[1]s-policy"
  policy_document    = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":\"s3:*\",\"Resource\":\"*\"}]}"
}

resource "scality_iam_role" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  role_name          = "%[1]s-role"
  assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"backbeat\"},\"Action\":\"sts:AssumeRole\"}]}"
}

resource "scality_iam_role_policy_attachment" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  role_name          = scality_iam_role.test.role_name
  policy_arn         = scality_iam_policy.test.arn
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "role_name", name+"-role"),
					resource.TestCheckResourceAttrSet(resourceName, "policy_arn"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc(resourceName, "role_name", "policy_arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "role_name",
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, resourceName, "role_name", "policy_arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "role_name",
			},
		},
	})
}
