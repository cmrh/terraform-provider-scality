package bucketencryption_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func testAccBucketEncryptionBase(name string) string {
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

func testAccBucketEncryptionConfig(name, algorithm, kmsKeyID string) string {
	kmsAttr := ""
	if kmsKeyID != "" {
		kmsAttr = fmt.Sprintf("\n  kms_master_key_id = %q", kmsKeyID)
	}

	return testAccBucketEncryptionBase(name) + fmt.Sprintf(`
resource "scality_bucket_encryption" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
  sse_algorithm      = %q%s
  depends_on         = [scality_user_policy.test]
}
`, algorithm, kmsAttr)
}

func TestAccBucketEncryption_basic(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_encryption"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketEncryptionBase(name) + `
resource "scality_bucket_encryption" "test" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
  sse_algorithm      = "AES256"
  depends_on         = [scality_user_policy.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_encryption.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_encryption.test", "sse_algorithm", "AES256"),
				),
			},
			{
				ResourceName:                         "scality_bucket_encryption.test",
				ImportState:                          true,
				ImportStateIdFunc:                    acctest.ImportStateIdFunc("scality_bucket_encryption.test", "bucket"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket",
			},
		},
	})
}

func TestAccBucketEncryption_update(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_encryption"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketEncryptionConfig(name, "AES256", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_encryption.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_encryption.test", "sse_algorithm", "AES256"),
				),
			},
			{
				Config: testAccBucketEncryptionConfig(name, "aws:kms", "arn:aws:kms:us-east-1:123456789012:key/test-key-id"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scality_bucket_encryption.test", "bucket", name+"-bucket"),
					resource.TestCheckResourceAttr("scality_bucket_encryption.test", "sse_algorithm", "aws:kms"),
					resource.TestCheckResourceAttr("scality_bucket_encryption.test", "kms_master_key_id", "arn:aws:kms:us-east-1:123456789012:key/test-key-id"),
				),
			},
		},
	})
}
