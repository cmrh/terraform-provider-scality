package account_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func TestAccAccountsDataSource_basic(t *testing.T) {
	name1 := acctest.RandomName("acctest")
	name2 := acctest.RandomName("acctest")
	name3 := acctest.RandomName("acctest")
	dataSourceName := "data.scality_accounts.all"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "a" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
}

resource "scality_account" "b" {
  name          = %[2]q
  email_address = "%[2]s@test.local"
}

resource "scality_account" "c" {
  name          = %[3]q
  email_address = "%[3]s@test.local"
}

data "scality_accounts" "all" {
  depends_on = [scality_account.a, scality_account.b, scality_account.c]
}
`, name1, name2, name3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "accounts.*", map[string]string{
						"name":          name1,
						"email_address": name1 + "@test.local",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "accounts.*", map[string]string{
						"name":          name2,
						"email_address": name2 + "@test.local",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "accounts.*", map[string]string{
						"name":          name3,
						"email_address": name3 + "@test.local",
					}),
				),
			},
		},
	})
}
