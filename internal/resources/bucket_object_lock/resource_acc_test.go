package bucketobjectlock_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func testAccBucketObjectLockBase(name string) string {
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
  account_access_key  = scality_user_access_key.test.access_key_id
  account_secret_key  = scality_user_access_key.test.secret_access_key
  bucket              = "%[1]s-bucket"
  object_lock_enabled = true
  depends_on          = [scality_user_policy.test]
}
`, name)
}

func testAccBucketObjectLockConfig(name string, days int) string {
	return testAccBucketObjectLockBase(name) + fmt.Sprintf(`
resource "scality_bucket_object_lock" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
  retention_mode     = "GOVERNANCE"
  retention_days     = %d
  depends_on         = [scality_user_policy.test]
}
`, days)
}

func TestAccBucketObjectLock_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_object_lock"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketObjectLockBase(name) + `
resource "scality_bucket_object_lock" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
  retention_mode     = "GOVERNANCE"
  retention_days     = 1
  depends_on         = [scality_user_policy.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "retention_mode", "GOVERNANCE"),
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "retention_days", "1"),
				),
			},
			{
				ResourceName:                         "scality_bucket_object_lock.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc("scality_bucket_object_lock.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
			{
				ResourceName:                         "scality_bucket_object_lock.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFuncIdentityOnly(t, "scality_bucket_object_lock.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
		},
	})
}

func TestAccBucketObjectLock_update(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_object_lock"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketObjectLockConfig(name, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "retention_mode", "GOVERNANCE"),
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "retention_days", "1"),
				),
			},
			{
				Config: testAccBucketObjectLockConfig(name, 7),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "retention_mode", "GOVERNANCE"),
					resource.TestCheckResourceAttr("scality_bucket_object_lock.test", "retention_days", "7"),
				),
			},
		},
	})
}
