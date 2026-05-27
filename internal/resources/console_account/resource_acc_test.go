package consoleaccount_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
	"github.com/scality/terraform-provider-scality/internal/client"
)

func TestAccConsoleAccount_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	resourceName := "scality_console_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_console_account"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "account_name", name),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "access_key"),
					resource.TestCheckResourceAttrSet(resourceName, "secret_key"),
					resource.TestCheckResourceAttrSet(resourceName, "password"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
		},
	})
}

// TestAccConsoleAccount_driftDetection verifies that out-of-band deletion via
// the Vault admin API is detected by Read and the resource is removed from
// state on refresh. Regression guard for the silent-drift behavior fixed in
// issue #54.
func TestAccConsoleAccount_driftDetection(t *testing.T) {
	name := acctest.RandomName("acctest")

	config := acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "test" {
  account_name             = %[1]q
  email                    = "%[1]s@test.local"
  generate_random_password = true
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_console_account"),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				PreConfig: func() {
					// Simulate "operator deletes the account via the Console UI."
					// Console's delete handles the paired IAM-user cleanup that
					// Vault admin's DeleteAccount would 409 on otherwise.
					consoleClient := client.NewConsoleClient(
						os.Getenv("SCALITY_CONSOLE_ENDPOINT"),
						os.Getenv("SCALITY_CONSOLE_USERNAME"),
						os.Getenv("SCALITY_CONSOLE_PASSWORD"),
						true,
					)
					if err := consoleClient.DeleteConsoleAccount(context.Background(), name); err != nil {
						t.Fatalf("out-of-band delete of %q failed: %v", name, err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
