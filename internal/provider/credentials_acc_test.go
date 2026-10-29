package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

// credentialResources are the account-level resources whose account credentials
// are switched by TestAccAccountCredentials_switch.
var credentialResources = []string{
	"scality_account_access_key.subject",
	"scality_user.test",
	"scality_user_policy.test",
	"scality_user_access_key.test",
	"scality_group.test",
	"scality_group_membership.test",
	"scality_iam_policy.test",
	"scality_iam_role.test",
	"scality_iam_role_policy_attachment.test",
	"scality_iam_group_policy_attachment.test",
}

// testAccCredentialsConfig manages one of each account-level resource in
// account a, all using the credentials named by ak/sk.
func testAccCredentialsConfig(name, ak, sk string) string {
	return acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "a" {
  name          = "%[1]s-a"
  email_address = "%[1]s-a@test.local"
}

resource "scality_account" "b" {
  name          = "%[1]s-b"
  email_address = "%[1]s-b@test.local"
}

resource "scality_account_access_key" "second" {
  account_access_key = scality_account.a.access_key
  account_secret_key = scality_account.a.secret_key
}

locals {
  ak = %[2]s
  sk = %[3]s
}

resource "scality_account_access_key" "subject" {
  account_access_key = local.ak
  account_secret_key = local.sk
}

resource "scality_user" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = "%[1]s-user"
}

resource "scality_user_policy" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = scality_user.test.username
  policy_name        = "inline"
  policy_document    = jsonencode({ Version = "2012-10-17", Statement = [{ Effect = "Allow", Action = "s3:ListBucket", Resource = "*" }] })
}

resource "scality_user_access_key" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = scality_user.test.username
}

resource "scality_group" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  group_name         = "%[1]s-group"
}

resource "scality_group_membership" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  group_name         = scality_group.test.group_name
  users              = [scality_user.test.username]
}

resource "scality_iam_policy" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  policy_name        = "%[1]s-policy"
  policy_document    = jsonencode({ Version = "2012-10-17", Statement = [{ Effect = "Allow", Action = "s3:Get*", Resource = "*" }] })
}

resource "scality_iam_role" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  role_name          = "%[1]s-role"
  assume_role_policy = jsonencode({ Version = "2012-10-17", Statement = [{ Effect = "Allow", Principal = { Service = "backbeat" }, Action = "sts:AssumeRole" }] })
}

resource "scality_iam_role_policy_attachment" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  role_name          = scality_iam_role.test.role_name
  policy_arn         = scality_iam_policy.test.arn
}

resource "scality_iam_group_policy_attachment" "test" {
  account_access_key = local.ak
  account_secret_key = local.sk
  group_name         = scality_group.test.group_name
  policy_arn         = scality_iam_policy.test.arn
}
`, name, ak, sk)
}

// TestAccAccountCredentials_switch guards #118: switching to another key of the
// same account updates in place; another account's key is refused.
func TestAccAccountCredentials_switch(t *testing.T) {
	name := acctest.RandomName("acctest")

	sameAccount := make([]plancheck.PlanCheck, 0, len(credentialResources))
	for _, r := range credentialResources {
		sameAccount = append(sameAccount, plancheck.ExpectResourceAction(r, plancheck.ResourceActionUpdate))
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: testAccCredentialsConfig(name, "scality_account.a.access_key", "scality_account.a.secret_key"),
			},
			{
				Config: testAccCredentialsConfig(name, "scality_account_access_key.second.access_key", "scality_account_access_key.second.secret_key"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: sameAccount,
				},
			},
			{
				Config:      testAccCredentialsConfig(name, "scality_account.b.access_key", "scality_account.b.secret_key"),
				ExpectError: regexp.MustCompile(`Credentials Belong to a Different Account`),
			},
		},
	})
}
