package bucketpolicy_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func testAccBucketPolicyBase(name string) string {
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

func testAccBucketPolicyConfig(name, effect string) string {
	return testAccBucketPolicyBase(name) + fmt.Sprintf(`
resource "scality_bucket_policy" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
  policy             = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "PublicAccess"
      Effect    = %q
      Principal = "*"
      Action    = "s3:GetObject"
      Resource  = "arn:aws:s3:::%[2]s-bucket/*"
    }]
  })
  depends_on = [scality_user_policy.test]
}
`, effect, name)
}

func TestAccBucketPolicy_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_policy"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketPolicyBase(name) + fmt.Sprintf(`
resource "scality_bucket_policy" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
  policy             = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = "*"
      Action    = "s3:GetObject"
      Resource  = "arn:aws:s3:::%[1]s-bucket/*"
    }]
  })
  depends_on = [scality_user_policy.test]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_policy.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttrSet("scality_bucket_policy.test", "policy"),
				),
			},
			{
				ResourceName:                         "scality_bucket_policy.test",
				ImportState:                          true,
				ImportStateIdFunc:                     acctest.ImportStateIdFunc("scality_bucket_policy.test", "bucket"),
				ImportStateVerify:                     true,
				ImportStateVerifyIdentifierAttribute:  "bucket",
				ImportStateVerifyIgnore:               []string{"policy"},
			},
		},
	})
}

func TestAccBucketPolicy_update(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_policy"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketPolicyConfig(name, "Allow"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_policy.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttrSet("scality_bucket_policy.test", "policy"),
				),
			},
			{
				Config: testAccBucketPolicyConfig(name, "Deny"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_policy.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttrSet("scality_bucket_policy.test", "policy"),
				),
			},
		},
	})
}
