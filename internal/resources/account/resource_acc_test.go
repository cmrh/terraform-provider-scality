package account_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

// TestAccAccount_emailCasePreserved verifies a mixed-case email does not force a
// perpetual replace: the server lowercases it, but state keeps the configured
// casing, so the post-apply plan is empty.
func TestAccAccount_emailCasePreserved(t *testing.T) {
	name := acctest.RandomName("acctest")
	email := fmt.Sprintf("Mixed.Case.%s@Example.com", name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = %[2]q
}
`, name, email),
			Check: resource.TestCheckResourceAttr("scality_account.test", "email_address", email),
		}},
	})
}

func TestAccAccount_tooManyCustomAttributes(t *testing.T) {
	name := acctest.RandomName("acctest")
	var b strings.Builder
	for i := 0; i < 11; i++ {
		fmt.Fprintf(&b, "    k%d = \"v\"\n", i)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
  custom_attributes = {
%[2]s  }
}
`, name, b.String()),
			ExpectError: regexp.MustCompile(`at most 10 custom attributes`),
		}},
	})
}

// TestAccAccount_quotaUpdateInPlace verifies that changing quota_max updates the
// account in place (ResourceActionUpdate) rather than destroying+recreating it.
func TestAccAccount_quotaUpdateInPlace(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_account.test"

	cfg := func(quota int64) string {
		return acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
  quota_max     = %[2]d
}
`, name, quota)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_account"),
		Steps: []resource.TestStep{
			{
				Config: cfg(1000000000),
				Check:  resource.TestCheckResourceAttr(resourceName, "quota_max", "1000000000"),
			},
			{
				Config: cfg(5000000000),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "quota_max", "5000000000"),
			},
		},
	})
}

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
