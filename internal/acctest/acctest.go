package acctest

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/scality/terraform-provider-scality/internal/client"
	"github.com/scality/terraform-provider-scality/internal/provider"
)

var TestProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"scality": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func PreCheck(t *testing.T) {
	t.Helper()
	required := []string{
		"SCALITY_ENDPOINT",
		"SCALITY_ACCESS_KEY",
		"SCALITY_SECRET_KEY",
	}
	for _, v := range required {
		if os.Getenv(v) == "" {
			t.Fatalf("%s must be set for acceptance tests", v)
		}
	}
}

func PreCheckConsole(t *testing.T) {
	t.Helper()
	PreCheck(t)
	required := []string{
		"SCALITY_CONSOLE_ENDPOINT",
		"SCALITY_CONSOLE_USERNAME",
		"SCALITY_CONSOLE_PASSWORD",
	}
	for _, v := range required {
		if os.Getenv(v) == "" {
			t.Fatalf("%s must be set for console acceptance tests", v)
		}
	}
}

func PreCheckCRR(t *testing.T) {
	t.Helper()
	PreCheck(t)
	PreCheckConsole(t)
	required := []string{
		"SCALITY_DEST_ENDPOINT",
		"SCALITY_DEST_ACCESS_KEY",
		"SCALITY_DEST_SECRET_KEY",
		"SCALITY_DEST_CONSOLE_ENDPOINT",
		"SCALITY_DEST_CONSOLE_USERNAME",
		"SCALITY_DEST_CONSOLE_PASSWORD",
	}
	for _, v := range required {
		if os.Getenv(v) == "" {
			t.Skipf("skipping CRR test: %s not set", v)
		}
	}
}

func RandomName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, rand.Intn(99999))
}

func ProviderBlock() string {
	return `
provider "scality" {
  insecure_skip_verify = true
}
`
}

func DestProviderBlock() string {
	return fmt.Sprintf(`
provider "scality" {
  alias                = "dest"
  endpoint             = "%s"
  access_key           = "%s"
  secret_key           = "%s"
  console_endpoint     = "%s"
  console_username     = "%s"
  console_password     = "%s"
  insecure_skip_verify = true
}
`,
		os.Getenv("SCALITY_DEST_ENDPOINT"),
		os.Getenv("SCALITY_DEST_ACCESS_KEY"),
		os.Getenv("SCALITY_DEST_SECRET_KEY"),
		os.Getenv("SCALITY_DEST_CONSOLE_ENDPOINT"),
		os.Getenv("SCALITY_DEST_CONSOLE_USERNAME"),
		os.Getenv("SCALITY_DEST_CONSOLE_PASSWORD"),
	)
}

func CheckResourceDestroyed(resourceType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}

			switch resourceType {
			case "scality_account":
				iamClient := client.NewIAMClient(
					os.Getenv("SCALITY_ENDPOINT"),
					os.Getenv("SCALITY_ACCESS_KEY"),
					os.Getenv("SCALITY_SECRET_KEY"),
					true,
				)
				name := rs.Primary.Attributes["name"]
				acct, err := iamClient.GetAccount(ctx, name)
				if err != nil {
					return fmt.Errorf("error checking account %s: %w", name, err)
				}
				if acct != nil {
					return fmt.Errorf("account %s still exists after destroy", name)
				}

			case "scality_console_account":
				consoleClient := client.NewConsoleClient(
					os.Getenv("SCALITY_CONSOLE_ENDPOINT"),
					os.Getenv("SCALITY_CONSOLE_USERNAME"),
					os.Getenv("SCALITY_CONSOLE_PASSWORD"),
					true,
				)
				name := rs.Primary.Attributes["name"]
				acct, err := consoleClient.GetConsoleAccount(ctx, name)
				if err != nil {
					return fmt.Errorf("error checking console account %s: %w", name, err)
				}
				if acct != nil {
					return fmt.Errorf("console account %s still exists after destroy", name)
				}

			default:
				// Sub-resources (users, buckets, policies, etc.) are destroyed
				// along with their parent account. We can't verify via API since
				// the account credentials are already gone by this point.
			}
		}
		return nil
	}
}
