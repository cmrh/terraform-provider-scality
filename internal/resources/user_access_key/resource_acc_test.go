package useraccesskey_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
	"github.com/cmrh/terraform-provider-scality/internal/client"
)

// deleteUserOutOfBand removes the parent user (and its key) directly via the API,
// simulating an out-of-band deletion.
func deleteUserOutOfBand(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not in state", resourceName)
		}
		ak := rs.Primary.Attributes["account_access_key"]
		sk := rs.Primary.Attributes["account_secret_key"]
		user := rs.Primary.Attributes["username"]
		keyID := rs.Primary.Attributes["access_key_id"]
		iam := client.NewIAMClient(os.Getenv("SCALITY_ENDPOINT"), "", "", true)
		ctx := context.Background()
		_ = iam.DeleteUserAccessKey(ctx, ak, sk, user, keyID) // a user with keys may be undeletable
		if err := iam.DeleteUser(ctx, ak, sk, user); err != nil {
			return fmt.Errorf("out-of-band delete user: %w", err)
		}
		return nil
	}
}

func TestAccUserAccessKey_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_user_access_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_user_access_key"),
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

resource "scality_user_access_key" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = scality_user.test.username
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "access_key_id"),
					resource.TestCheckResourceAttrSet(resourceName, "secret_access_key"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc(resourceName, "username", "access_key_id"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "access_key_id",
				ImportStateVerifyIgnore:              []string{"secret_access_key"},
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, resourceName, "username", "access_key_id"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "access_key_id",
				ImportStateVerifyIgnore:              []string{"secret_access_key"},
			},
		},
	})
}

// TestAccUserAccessKey_parentUserDeletedOutOfBand guards #95: when the parent
// user is deleted outside Terraform, refresh must drop the key from state (plan
// then recreates it) instead of erroring on NoSuchEntity and blocking all plans.
func TestAccUserAccessKey_parentUserDeletedOutOfBand(t *testing.T) {
	name := acctest.RandomName("acctest")
	cfg := acctest.ProviderBlock() + fmt.Sprintf(`
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

resource "scality_user_access_key" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = scality_user.test.username
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_user_access_key"),
		Steps: []resource.TestStep{
			{
				Config:             cfg,
				Check:              deleteUserOutOfBand("scality_user_access_key.test"),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
