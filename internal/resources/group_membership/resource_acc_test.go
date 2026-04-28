package groupmembership_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func testAccGroupMembershipBase(name string) string {
	return acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}

resource "scality_user" "test1" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = "%[1]s-user1"
}

resource "scality_user" "test2" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = "%[1]s-user2"
}

resource "scality_group" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  group_name         = "%[1]s-group"
}
`, name)
}

func TestAccGroupMembership_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_group_membership.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_group_membership"),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupMembershipBase(name) + `
resource "scality_group_membership" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  group_name         = scality_group.test.group_name
  users = [
    scality_user.test1.username,
    scality_user.test2.username,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "group_name", name+"-group"),
					resource.TestCheckResourceAttr(resourceName, "users.#", "2"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc(resourceName, "group_name"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "group_name",
			},
		},
	})
}

func TestAccGroupMembership_update(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_group_membership.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_group_membership"),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupMembershipBase(name) + `
resource "scality_group_membership" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  group_name         = scality_group.test.group_name
  users = [
    scality_user.test1.username,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "group_name", name+"-group"),
					resource.TestCheckResourceAttr(resourceName, "users.#", "1"),
				),
			},
			{
				Config: testAccGroupMembershipBase(name) + `
resource "scality_group_membership" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  group_name         = scality_group.test.group_name
  users = [
    scality_user.test1.username,
    scality_user.test2.username,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "group_name", name+"-group"),
					resource.TestCheckResourceAttr(resourceName, "users.#", "2"),
				),
			},
		},
	})
}
