package bucket_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
	"github.com/cmrh/terraform-provider-scality/internal/client"
)

// checkBucketVersioningAPI asserts the bucket's real S3 versioning status,
// reading the account credentials and bucket name out of state. expected is the
// raw status: "" (Unversioned), "Enabled", or "Suspended".
func checkBucketVersioningAPI(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		b, ok := s.RootModule().Resources["scality_bucket.test"]
		if !ok {
			return fmt.Errorf("scality_bucket.test not in state")
		}
		k, ok := s.RootModule().Resources["scality_user_access_key.test"]
		if !ok {
			return fmt.Errorf("scality_user_access_key.test not in state")
		}
		s3 := client.NewS3Client(os.Getenv("SCALITY_ENDPOINT"), true)
		status, err := s3.GetBucketVersioning(
			context.Background(),
			k.Primary.Attributes["access_key_id"],
			k.Primary.Attributes["secret_access_key"],
			b.Primary.Attributes["bucket"],
		)
		if err != nil {
			return fmt.Errorf("get bucket versioning: %w", err)
		}
		if status != expected {
			return fmt.Errorf("bucket versioning status = %q, want %q", status, expected)
		}
		return nil
	}
}

func testAccBucketVersioningConfig(name string, versioning bool) string {
	return testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  versioning         = %[2]t
  depends_on         = [scality_user_policy.test]
}
`, name, versioning)
}

func testAccBucketNoVersioningConfig(name string) string {
	return testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  depends_on         = [scality_user_policy.test]
}
`, name)
}

// TestAccBucket_versioningUnsetLeavesAsIs verifies that removing the versioning
// attribute means "stop managing" — an Enabled bucket is left Enabled, not
// suspended (which would also 409 on an Object Lock bucket).
func TestAccBucket_versioningUnsetLeavesAsIs(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{ // manage versioning = true -> Enabled
				Config: testAccBucketVersioningConfig(name, true),
				Check:  checkBucketVersioningAPI("Enabled"),
			},
			{ // drop the attribute -> unmanaged; bucket must stay Enabled
				Config: testAccBucketNoVersioningConfig(name),
				Check:  checkBucketVersioningAPI("Enabled"),
			},
		},
	})
}

// TestAccBucket_objectLockVersioningConflict verifies the plan-time guard:
// object_lock_enabled = true with versioning = false is rejected before apply.
func TestAccBucket_objectLockVersioningConflict(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key  = scality_user_access_key.test.access_key_id
  account_secret_key  = scality_user_access_key.test.secret_access_key
  bucket              = "%[1]s-bucket"
  object_lock_enabled = true
  versioning          = false
  depends_on          = [scality_user_policy.test]
}
`, name),
				ExpectError: regexp.MustCompile(`requires versioning to be enabled`),
			},
		},
	})
}

// TestAccBucket_versioningLifecycle pins the three-state contract: versioning =
// false on a new bucket leaves it Unversioned (NOT Suspended); enabling then
// disabling lands on Suspended, the only "off" state reachable once enabled.
func TestAccBucket_versioningLifecycle(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{ // created with false -> never enabled -> Unversioned
				Config: testAccBucketVersioningConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket.test", "versioning", "false"),
					checkBucketVersioningAPI(""),
				),
			},
			{ // true -> Enabled
				Config: testAccBucketVersioningConfig(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket.test", "versioning", "true"),
					checkBucketVersioningAPI("Enabled"),
				),
			},
			{ // back to false -> was enabled -> Suspended
				Config: testAccBucketVersioningConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket.test", "versioning", "false"),
					checkBucketVersioningAPI("Suspended"),
				),
			},
		},
	})
}

func testAccBucketBase(name string) string {
	return acctest.ProviderBlock() + fmt.Sprintf(`
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

resource "scality_user_policy" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = scality_user.test.username
  policy_name        = "full-access"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "*"
      Resource = "*"
    }]
  })
}

resource "scality_user_access_key" "test" {
  account_access_key = scality_console_account.test.access_key
  account_secret_key = scality_console_account.test.secret_key
  username           = scality_user.test.username
}
`, name)
}

func TestAccBucket_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  depends_on         = [scality_user_policy.test]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket.test", "bucket", name+"-bucket"),
				),
			},
			{
				ResourceName:                         "scality_bucket.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc("scality_bucket.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
			{
				ResourceName:                         "scality_bucket.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, "scality_bucket.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
		},
	})
}

func TestAccBucket_versioning(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  versioning         = true
  depends_on         = [scality_user_policy.test]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket.test", "versioning", "true"),
				),
			},
		},
	})
}

func TestAccBucket_tags(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  tags = {
    Environment = "test"
    Project     = "acctest"
  }
  depends_on = [scality_user_policy.test]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket.test", "tags.Environment", "test"),
					resource.TestCheckResourceAttr("scality_bucket.test", "tags.Project", "acctest"),
				),
			},
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  tags = {
    Environment = "production"
    Team        = "platform"
  }
  depends_on = [scality_user_policy.test]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket.test", "tags.Environment", "production"),
					resource.TestCheckResourceAttr("scality_bucket.test", "tags.Team", "platform"),
				),
			},
		},
	})
}

// TestAccBucket_tagsEmptyRoundTrip guards the empty-collection round-trip: a
// configured tags = {} must survive apply->read unchanged. Previously Read
// forced an empty map to null, producing a perpetual "+ tags = {}" diff.
func TestAccBucket_tagsEmptyRoundTrip(t *testing.T) {
	name := acctest.RandomName("acctest")
	cfg := testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  tags               = {}
  depends_on         = [scality_user_policy.test]
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check:  resource.TestCheckResourceAttr("scality_bucket.test", "tags.%", "0"),
			},
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
