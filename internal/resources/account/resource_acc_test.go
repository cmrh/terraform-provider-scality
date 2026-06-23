package account_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

func TestAccAccount_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_account.test"

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
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "arn"),
					resource.TestCheckResourceAttrSet(resourceName, "canonical_id"),
					resource.TestCheckResourceAttrSet(resourceName, "create_date"),
					resource.TestCheckResourceAttrSet(resourceName, "access_key"),
					resource.TestCheckResourceAttrSet(resourceName, "secret_key"),
				),
			},
		},
	})
}

func TestAccAccount_customAttributes(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_account.test"

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
    env  = "test"
    team = "platform"
  }
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "custom_attributes.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "custom_attributes.team", "platform"),
				),
			},
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
  custom_attributes = {
    env    = "staging"
    region = "us-east-1"
  }
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "custom_attributes.env", "staging"),
					resource.TestCheckResourceAttr(resourceName, "custom_attributes.region", "us-east-1"),
				),
			},
		},
	})
}
