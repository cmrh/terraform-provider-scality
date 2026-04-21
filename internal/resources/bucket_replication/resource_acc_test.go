package bucketreplication_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/scality/terraform-provider-scality/internal/acctest"
)

func testAccCRRConfig(name string) string {
	return acctest.ProviderBlock() + acctest.DestProviderBlock() + fmt.Sprintf(`
resource "scality_console_account" "source" {
  account_name             = "%[1]s-src"
  email                    = "%[1]s-src@test.local"
  generate_random_password = true
}

resource "scality_user" "source" {
  account_access_key = scality_console_account.source.access_key
  account_secret_key = scality_console_account.source.secret_key
  username           = "operator"
}

resource "scality_user_policy" "source" {
  account_access_key = scality_console_account.source.access_key
  account_secret_key = scality_console_account.source.secret_key
  username           = scality_user.source.username
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

resource "scality_user_access_key" "source" {
  account_access_key = scality_console_account.source.access_key
  account_secret_key = scality_console_account.source.secret_key
  username           = scality_user.source.username
}

resource "scality_console_account" "dest" {
  provider                 = scality.dest
  account_name             = "%[1]s-dst"
  email                    = "%[1]s-dst@test.local"
  generate_random_password = true
}

resource "scality_user" "dest" {
  provider           = scality.dest
  account_access_key = scality_console_account.dest.access_key
  account_secret_key = scality_console_account.dest.secret_key
  username           = "operator"
}

resource "scality_user_policy" "dest" {
  provider           = scality.dest
  account_access_key = scality_console_account.dest.access_key
  account_secret_key = scality_console_account.dest.secret_key
  username           = scality_user.dest.username
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

resource "scality_user_access_key" "dest" {
  provider           = scality.dest
  account_access_key = scality_console_account.dest.access_key
  account_secret_key = scality_console_account.dest.secret_key
  username           = scality_user.dest.username
}

resource "scality_bucket" "source" {
  account_access_key = scality_user_access_key.source.access_key_id
  account_secret_key = scality_user_access_key.source.secret_access_key
  bucket             = "%[1]s-src"
  versioning         = true

  depends_on = [scality_user_policy.source]
}

resource "scality_bucket" "dest" {
  provider           = scality.dest
  account_access_key = scality_user_access_key.dest.access_key_id
  account_secret_key = scality_user_access_key.dest.secret_access_key
  bucket             = "%[1]s-dst"
  versioning         = true

  depends_on = [scality_user_policy.dest]
}

resource "scality_iam_policy" "source" {
  account_access_key = scality_console_account.source.access_key
  account_secret_key = scality_console_account.source.secret_key
  policy_name        = "crr-policy"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:GetObjectVersion", "s3:GetObjectVersionAcl", "s3:GetObjectTagging", "s3:ReplicateObject", "s3:ReplicateTags", "s3:ReplicateDelete"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket", "s3:GetReplicationConfiguration"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ReplicateObject", "s3:ReplicateDelete", "s3:ReplicateTags", "s3:PutObject"]
        Resource = "arn:aws:s3:::${scality_bucket.dest.bucket}/*"
      },
    ]
  })
}

resource "scality_iam_policy" "dest" {
  provider           = scality.dest
  account_access_key = scality_console_account.dest.access_key
  account_secret_key = scality_console_account.dest.secret_key
  policy_name        = "crr-policy"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:GetObjectVersion", "s3:GetObjectVersionAcl", "s3:GetObjectTagging", "s3:ReplicateObject", "s3:ReplicateTags", "s3:ReplicateDelete"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket", "s3:GetReplicationConfiguration"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ReplicateObject", "s3:ReplicateDelete", "s3:ReplicateTags", "s3:PutObject"]
        Resource = "arn:aws:s3:::${scality_bucket.dest.bucket}/*"
      },
    ]
  })
}

resource "scality_iam_role" "source" {
  account_access_key = scality_console_account.source.access_key
  account_secret_key = scality_console_account.source.secret_key
  role_name          = "crr-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "backbeat" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "scality_iam_role" "dest" {
  provider           = scality.dest
  account_access_key = scality_console_account.dest.access_key
  account_secret_key = scality_console_account.dest.secret_key
  role_name          = "crr-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "backbeat" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "scality_iam_role_policy_attachment" "source" {
  account_access_key = scality_console_account.source.access_key
  account_secret_key = scality_console_account.source.secret_key
  role_name          = scality_iam_role.source.role_name
  policy_arn         = scality_iam_policy.source.arn
}

resource "scality_iam_role_policy_attachment" "dest" {
  provider           = scality.dest
  account_access_key = scality_console_account.dest.access_key
  account_secret_key = scality_console_account.dest.secret_key
  role_name          = scality_iam_role.dest.role_name
  policy_arn         = scality_iam_policy.dest.arn
}

resource "scality_bucket_replication" "test" {
  account_access_key = scality_user_access_key.source.access_key_id
  account_secret_key = scality_user_access_key.source.secret_access_key
  bucket             = scality_bucket.source.bucket
  role               = "${scality_iam_role.source.arn},${scality_iam_role.dest.arn}"

  rule {
    status             = "Enabled"
    prefix             = ""
    destination_bucket = "arn:aws:s3:::${scality_bucket.dest.bucket}"
  }

  depends_on = [
    scality_iam_role_policy_attachment.source,
    scality_iam_role_policy_attachment.dest,
  ]
}
`, name)
}

func TestAccBucketReplication_crr(t *testing.T) {
	name := acctest.RandomName("acctest")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckCRR(t) },
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		CheckDestroy:             acctest.CheckResourceDestroyed("scality_bucket_replication"),
		Steps: []resource.TestStep{
			{
				Config: testAccCRRConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("scality_bucket_replication.test", "bucket"),
					resource.TestCheckResourceAttrSet("scality_bucket_replication.test", "role"),
					resource.TestCheckResourceAttr("scality_bucket_replication.test", "rule.0.status", "Enabled"),
					resource.TestCheckResourceAttrSet("scality_bucket_replication.test", "rule.0.destination_bucket"),
					resource.TestCheckResourceAttrSet("scality_console_account.source", "access_key"),
					resource.TestCheckResourceAttrSet("scality_console_account.dest", "access_key"),
					resource.TestCheckResourceAttrSet("scality_iam_role.source", "arn"),
					resource.TestCheckResourceAttrSet("scality_iam_role.dest", "arn"),
				),
			},
		},
	})
}
