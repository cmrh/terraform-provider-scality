package bucket_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func TestAccBucketsDataSource_basic(t *testing.T) {
	name := acctest.RandomName("acctest")
	dataSourceName := "data.scality_buckets.all"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket"),
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + fmt.Sprintf(`
resource "scality_bucket" "alpha" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-alpha"
  depends_on         = [scality_user_policy.test]
}

resource "scality_bucket" "beta" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-beta"
  depends_on         = [scality_user_policy.test]
}

resource "scality_bucket" "gamma" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  bucket             = "%[1]s-gamma"
  depends_on         = [scality_user_policy.test]
}

data "scality_buckets" "all" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  depends_on         = [scality_bucket.alpha, scality_bucket.beta, scality_bucket.gamma]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "buckets.*", map[string]string{
						"name": name + "-alpha",
						"arn":  "arn:aws:s3:::" + name + "-alpha",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "buckets.*", map[string]string{
						"name": name + "-beta",
						"arn":  "arn:aws:s3:::" + name + "-beta",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "buckets.*", map[string]string{
						"name": name + "-gamma",
						"arn":  "arn:aws:s3:::" + name + "-gamma",
					}),
				),
			},
		},
	})
}

func TestAccBucketsDataSource_empty(t *testing.T) {
	name := acctest.RandomName("acctest")
	dataSourceName := "data.scality_buckets.all"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckConsole(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketBase(name) + `
data "scality_buckets" "all" {
  account_access_key = scality_user_access_key.test.access_key_id
  account_secret_key = scality_user_access_key.test.secret_access_key
  depends_on         = [scality_user_policy.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "0"),
				),
			},
		},
	})
}
