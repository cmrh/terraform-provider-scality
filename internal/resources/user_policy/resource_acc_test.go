package userpolicy_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func TestAccUserPolicy_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_user_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_user_policy"),
		Steps: []resource.TestStep{
			{
				Config: testAccUserPolicyConfig(name, `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:ListBucket","Resource":"*"}]}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", name+"-user"),
					resource.TestCheckResourceAttr(resourceName, "policy_name", name+"-policy"),
					resource.TestCheckResourceAttrSet(resourceName, "policy_document"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                     acctest.ImportStateIdFunc(resourceName, "username", "policy_name"),
				ImportStateVerify:                     true,
				ImportStateVerifyIdentifierAttribute:  "username",
				ImportStateVerifyIgnore:               []string{"policy_document"},
			},
		},
	})
}

func TestAccUserPolicy_update(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_user_policy.test"

	doc1 := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:ListBucket","Resource":"*"}]}`
	doc2 := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_user_policy"),
		Steps: []resource.TestStep{
			{
				Config: testAccUserPolicyConfig(name, doc1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "policy_document", doc1),
				),
			},
			{
				Config: testAccUserPolicyConfig(name, doc2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "policy_document", doc2),
				),
			},
		},
	})
}

func testAccUserPolicyConfig(name, policyDoc string) string {
	return acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}

resource "scality_user" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = "%[1]s-user"
}

resource "scality_user_policy" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = scality_user.test.username
  policy_name        = "%[1]s-policy"
  policy_document    = %[2]q
}
`, name, policyDoc)
}
