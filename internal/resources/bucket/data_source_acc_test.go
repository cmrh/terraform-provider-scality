package bucket_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func TestAccBucketDataSource_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	dataSourceName := "data.scality_bucket.lookup"
	bucketName := name + "-bucket"

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
  bucket             = %[1]q
  versioning         = true
  tags = {
    Environment = "test"
    Project     = "acctest"
  }
  depends_on = [scality_user_policy.test]
}

data "scality_bucket" "lookup" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
}
`, bucketName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "bucket", bucketName),
					resource.TestCheckResourceAttr(dataSourceName, "id", bucketName),
					resource.TestCheckResourceAttr(dataSourceName, "arn", "arn:aws:s3:::"+bucketName),
					resource.TestCheckResourceAttr(dataSourceName, "versioning", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "object_lock_enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "tags.Environment", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "tags.Project", "acctest"),
				),
			},
		},
	})
}

func TestAccBucketDataSource_objectLock(t *testing.T) {
	name := acctest.RandomName("acctest")
	dataSourceName := "data.scality_bucket.lookup"
	bucketName := name + "-lock"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "test" {
  account_access_key  = scality_user_access_key.test.access_key_id
  account_secret_key  = scality_user_access_key.test.secret_access_key
  bucket              = %[1]q
  object_lock_enabled = true
  depends_on          = [scality_user_policy.test]
}

data "scality_bucket" "lookup" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = scality_bucket.test.bucket
}
`, bucketName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "bucket", bucketName),
					resource.TestCheckResourceAttr(dataSourceName, "object_lock_enabled", "true"),
				),
			},
		},
	})
}

func TestAccBucketDataSource_notFound(t *testing.T) {
	name := acctest.RandomName("acctest")
	missing := name + "-does-not-exist"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
data "scality_bucket" "lookup" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = %[1]q
  depends_on         = [scality_user_policy.test]
}
`, missing),
				ExpectError: regexp.MustCompile(`Bucket Not Found`),
			},
		},
	})
}
