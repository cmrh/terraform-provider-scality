package bucketlifecycle_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

func testAccBucketLifecycleBase(name string) string {
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

resource "scality_bucket" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-bucket"
  depends_on         = [scality_user_policy.test]
}
`, name)
}

func testAccBucketLifecycleConfig(name string, days int) string {
	return testAccBucketLifecycleBase(name) + fmt.Sprintf(`
resource "scality_bucket_lifecycle" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket

  rule {
    id              = "expire-objects"
    status          = "Enabled"
    expiration_days = %d
  }

  depends_on = [scality_user_policy.test]
}
`, days)
}

func TestAccBucketLifecycle_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_lifecycle"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketLifecycleBase(name) + `
resource "scality_bucket_lifecycle" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket

  rule {
    id              = "expire-30d"
    status          = "Enabled"
    expiration_days = 30
  }

  depends_on = [scality_user_policy.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.id", "expire-30d"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.status", "Enabled"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.expiration_days", "30"),
				),
			},
			{
				ResourceName:                         "scality_bucket_lifecycle.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc("scality_bucket_lifecycle.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
			{
				ResourceName:                         "scality_bucket_lifecycle.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, "scality_bucket_lifecycle.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
		},
	})
}

func TestAccBucketLifecycle_update(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_lifecycle"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketLifecycleConfig(name, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.id", "expire-objects"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.status", "Enabled"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.expiration_days", "30"),
				),
			},
			{
				Config: testAccBucketLifecycleConfig(name, 90),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.id", "expire-objects"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.status", "Enabled"),
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.expiration_days", "90"),
				),
			},
		},
	})
}

// TestAccBucketLifecycle_emptyPrefixNoPhantomDiff guards against the
// null/empty round-trip mismatch where Read collapsed `prefix = ""` into a
// null state value, producing a phantom in-place update on the next plan. The
// framework's automatic post-apply plan check (ExpectNonEmptyPlan defaults to
// false) fails the test if the regression returns.
func TestAccBucketLifecycle_emptyPrefixNoPhantomDiff(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_lifecycle"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketLifecycleBase(name) + `
resource "scality_bucket_lifecycle" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket

  rule {
    id                                     = "expire-stuff"
    status                                 = "Enabled"
    prefix                                 = ""
    abort_incomplete_multipart_upload_days = 7
    noncurrent_version_expiration_days     = 30
  }

  depends_on = [scality_user_policy.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_lifecycle.test", "rule.0.prefix", ""),
				),
			},
		},
	})
}
