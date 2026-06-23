package account_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

func TestAccAccountDataSource_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	dataSourceName := "data.scality_account.lookup"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
  custom_attributes = {
    env = "test"
  }
}

data "scality_account" "lookup" {
  name = scality_account.test.name
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "arn"),
					resource.TestCheckResourceAttrSet(dataSourceName, "canonical_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "create_date"),
					resource.TestCheckResourceAttr(dataSourceName, "email_address", name+"@test.local"),
					resource.TestCheckResourceAttr(dataSourceName, "custom_attributes.env", "test"),
					resource.TestCheckResourceAttrPair(dataSourceName, "id", "scality_account.test", "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "arn", "scality_account.test", "arn"),
					resource.TestCheckResourceAttrPair(dataSourceName, "canonical_id", "scality_account.test", "canonical_id"),
				),
			},
		},
	})
}

func TestAccAccountDataSource_noCustomAttributes(t *testing.T) {
	name := acctest.RandomName("acctest")
	dataSourceName := "data.scality_account.lookup"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
}

data "scality_account" "lookup" {
  name = scality_account.test.name
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", name),
					resource.TestCheckResourceAttr(dataSourceName, "custom_attributes.%", "0"),
				),
			},
		},
	})
}

func TestAccAccountDataSource_notFound(t *testing.T) {
	missing := acctest.RandomName("acctest-missing")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
data "scality_account" "lookup" {
  name = %[1]q
}
`, missing),
				ExpectError: regexp.MustCompile(`Account Not Found`),
			},
		},
	})
}
