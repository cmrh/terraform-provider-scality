package user_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func TestAccUser_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_user"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
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
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", name+"-user"),
					resource.TestCheckResourceAttrSet(resourceName, "user_id"),
					resource.TestCheckResourceAttrSet(resourceName, "arn"),
					resource.TestCheckResourceAttrSet(resourceName, "path"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                     acctest.ImportStateIdFunc(resourceName, "username"),
				ImportStateVerify:                     true,
				ImportStateVerifyIdentifierAttribute:  "username",
			},
		},
	})
}
