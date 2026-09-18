package acctest

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/provider"
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
	return fmt.Sprintf("%s-%d", prefix, rand.Intn(99999)) // #nosec G404 -- math/rand suffices for unique acceptance-test resource names
}

// SkipIfServerAccessLoggingDisabled skips the test when the cluster has server
// access logging turned off (the API returns 501 NotImplemented). The feature
// gate fires after auth but before the bucket-existence check, so the probe needs
// valid account S3 credentials — the platform superadmin key is not an S3
// credential. It creates a throwaway account, probes a nonexistent bucket with
// the account key, and cleans up; a disabled cluster yields the sentinel error,
// an enabled one yields NoSuchBucket (or nil).
func SkipIfServerAccessLoggingDisabled(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	admin := client.NewIAMClient(
		os.Getenv("SCALITY_ENDPOINT"),
		os.Getenv("SCALITY_ACCESS_KEY"),
		os.Getenv("SCALITY_SECRET_KEY"),
		true,
	)
	if r := os.Getenv("SCALITY_REGION"); r != "" {
		admin.Region = r
	}

	name := RandomName("acctest-logprobe")
	if _, err := admin.CreateAccount(ctx, client.AccountCreateRequest{
		Name:         name,
		EmailAddress: name + "@test.local",
	}); err != nil {
		t.Fatalf("logging-feature probe: create account: %s", err)
	}
	defer func() {
		if err := admin.DeleteAccount(ctx, name); err != nil {
			t.Logf("logging-feature probe: cleanup delete account %s: %s", name, err)
		}
	}()

	key, err := admin.GenerateAccountAccessKey(ctx, name)
	if err != nil {
		t.Fatalf("logging-feature probe: generate access key: %s", err)
	}

	s3 := client.NewS3Client(os.Getenv("SCALITY_ENDPOINT"), true)
	if r := os.Getenv("SCALITY_REGION"); r != "" {
		s3.Region = r
	}
	_, err = s3.GetBucketLogging(ctx, key.Data.ID, key.Data.Value, "acctest-logging-probe-nonexistent")
	if errors.Is(err, client.ErrServerAccessLoggingDisabled) {
		t.Skip("server access logging is not enabled on this cluster; skipping")
	}
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

// ImportStateIdFuncIdentityOnly produces an env-mode import ID — the identity
// portion only, no leading account credentials — and stages
// SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY in the test's
// environment so ImportState picks up the env-gated path.
func ImportStateIdFuncIdentityOnly(t *testing.T, resourceName string, idAttrs ...string) resource.ImportStateIdFunc {
	t.Helper()
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		t.Setenv("SCALITY_ACCOUNT_ACCESS_KEY", rs.Primary.Attributes["account_access_key"])
		t.Setenv("SCALITY_ACCOUNT_SECRET_KEY", rs.Primary.Attributes["account_secret_key"])
		parts := make([]string, 0, len(idAttrs))
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
				if r := os.Getenv("SCALITY_REGION"); r != "" {
					iamClient.Region = r
				}
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
