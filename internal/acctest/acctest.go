package acctest

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
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

func ImportStateIdFunc(resourceName string, idAttrs ...string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		parts := []string{
			rs.Primary.Attributes["account_access_key"],
			rs.Primary.Attributes["account_secret_key"],
		}
		for _, attr := range idAttrs {
			parts = append(parts, rs.Primary.Attributes[attr])
		}
		return strings.Join(parts, ":"), nil
	}
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
				if err := checkConsoleAccountDestroyed(ctx, rs.Primary.Attributes["account_name"]); err != nil {
					return err
				}

			default:
				if err := checkParentAccountsDestroyed(ctx, s); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func checkConsoleAccountDestroyed(ctx context.Context, name string) error {
	consoleClient := client.NewConsoleClient(
		os.Getenv("SCALITY_CONSOLE_ENDPOINT"),
		os.Getenv("SCALITY_CONSOLE_USERNAME"),
		os.Getenv("SCALITY_CONSOLE_PASSWORD"),
		true,
	)
	acct, err := consoleClient.GetConsoleAccount(ctx, name)
	if err != nil {
		return fmt.Errorf("error checking console account %s: %w", name, err)
	}
	if acct != nil {
		return fmt.Errorf("console account %s still exists after destroy", name)
	}
	return nil
}

func checkParentAccountsDestroyed(ctx context.Context, s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "scality_console_account":
			if err := checkConsoleAccountDestroyed(ctx, rs.Primary.Attributes["account_name"]); err != nil {
				return err
			}
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
		}
	}
	return nil
}
