package bucketlogging_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
)

func testAccBucketLoggingBase(name string) string {
	return acctest.ProviderBlock() + fmt.Sprintf(`
resource "scality_account" "test" {
  name          = %[1]q
  email_address = "%[1]s@test.local"
}

resource "scality_user" "test" {
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
  username           = "%[1]s-user"
}

resource "scality_user_policy" "test" {
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
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
  account_access_key = scality_account.test.access_key
  account_secret_key = scality_account.test.secret_key
  username           = scality_user.test.username
}

resource "scality_bucket" "source" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-src"
  depends_on         = [scality_user_policy.test]
}

resource "scality_bucket" "target" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-logs"
  depends_on         = [scality_user_policy.test]
}
`, name)
}

func testAccBucketLoggingConfig(name, prefix string) string {
	return testAccBucketLoggingBase(name) + fmt.Sprintf(`
resource "scality_bucket_logging" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.source.bucket
  target_bucket      = scality_bucket.target.bucket
  target_prefix      = %q
  depends_on         = [scality_user_policy.test]
}
`, prefix)
}

func TestAccBucketLogging_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t); acctest.SkipIfServerAccessLoggingDisabled(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_logging"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketLoggingConfig(name, name+"/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_logging.test", "bucket", name+"-src"),
					resource.TestCheckResourceAttr("scality_bucket_logging.test", "target_bucket", name+"-logs"),
					resource.TestCheckResourceAttr("scality_bucket_logging.test", "target_prefix", name+"/"),
				),
			},
			{
				ResourceName:                         "scality_bucket_logging.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc("scality_bucket_logging.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
			{
				ResourceName:                         "scality_bucket_logging.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, "scality_bucket_logging.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
		},
	})
}

func TestAccBucketLogging_update(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t); acctest.SkipIfServerAccessLoggingDisabled(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_logging"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketLoggingConfig(name, "logs/"),
				Check:  resource.TestCheckResourceAttr("scality_bucket_logging.test", "target_prefix", "logs/"),
			},
			{
				Config: testAccBucketLoggingConfig(name, "changed/"),
				Check:  resource.TestCheckResourceAttr("scality_bucket_logging.test", "target_prefix", "changed/"),
			},
		},
	})
}
