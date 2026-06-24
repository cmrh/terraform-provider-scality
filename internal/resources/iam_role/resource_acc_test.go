package iamrole_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

func TestAccIAMRole_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_iam_role.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_iam_role"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}

resource "scality_iam_role" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  role_name          = "%[1]s-role"
  assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"backbeat\"},\"Action\":\"sts:AssumeRole\"}]}"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "role_name", name+"-role"),
					resource.TestCheckResourceAttrSet(resourceName, "arn"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc(resourceName, "role_name"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "role_name",
				ImportStateVerifyIgnore:              []string{"assume_role_policy"},
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, resourceName, "role_name"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "role_name",
				ImportStateVerifyIgnore:              []string{"assume_role_policy"},
			},
		},
	})
}

// TestAccIAMRole_jsonWhitespace exercises the jsontypes.Normalized custom
// type: the config carries a pretty-printed trust policy while the API stores
// and returns compact, URL-encoded JSON. The framework's post-apply plan must
// report no changes — semantic JSON equality, not byte-equality.
func TestAccIAMRole_jsonWhitespace(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_iam_role.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_iam_role"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}

resource "scality_iam_role" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  role_name          = "%[1]s-role"
  assume_role_policy = <<EOT
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "backbeat"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOT
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "role_name", name+"-role"),
				),
			},
		},
	})
}
