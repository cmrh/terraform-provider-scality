package iampolicy_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func testAccIAMPolicyConfig(name, policyDoc string) string {
	return acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}

resource "scality_iam_policy" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  policy_name        = "%[1]s-policy"
  policy_document    = %[2]q
}
`, name, policyDoc)
}

func TestAccIAMPolicy_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_iam_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_iam_policy"),
		Steps: []resource.TestStep{
			{
				Config: testAccIAMPolicyConfig(name, `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "policy_name", name+"-policy"),
					resource.TestCheckResourceAttrSet(resourceName, "arn"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc(resourceName, "arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "arn",
				ImportStateVerifyIgnore:              []string{"policy_document"},
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, resourceName, "arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "arn",
				ImportStateVerifyIgnore:              []string{"policy_document"},
			},
		},
	})
}

func TestAccIAMPolicy_update(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_iam_policy.test"

	doc1 := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	doc2 := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetObject","s3:ListBucket"],"Resource":"*"}]}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_iam_policy"),
		Steps: []resource.TestStep{
			{
				Config: testAccIAMPolicyConfig(name, doc1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "arn"),
				),
			},
			{
				Config: testAccIAMPolicyConfig(name, doc2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "arn"),
				),
			},
		},
	})
}
