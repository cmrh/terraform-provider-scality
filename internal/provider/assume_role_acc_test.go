package provider_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cmrh/terraform-provider-scality/internal/acctest"
	"github.com/cmrh/terraform-provider-scality/internal/client"
)

// TestAccProviderAssumeRole_perAccountResources verifies the provider-level
// assume_role flow end-to-end: base account credentials are exchanged for
// temporary role credentials via STS, and per-account resources that omit their
// own account_access_key/account_secret_key fall back to them. It drives both a
// bucket (the S3 signing path) and a user (the IAM signing path), so a success
// proves the session token is threaded through both.
//
// Setup is done out-of-band because the role must exist before the provider is
// configured with assume_role — it cannot be created in the same config that
// assumes it.
func TestAccProviderAssumeRole_perAccountResources(t *testing.T) {
	// Out-of-band setup runs before resource.Test, so guard TF_ACC explicitly
	// (resource.Test's own skip would come too late).
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC must be set for acceptance tests")
	}
	acctest.PreCheck(t)

	ctx := context.Background()
	endpoint := os.Getenv("SCALITY_ENDPOINT")
	region := os.Getenv("SCALITY_REGION")

	newIAM := func(ak, sk string) *client.IAMClient {
		c := client.NewIAMClient(endpoint, ak, sk, true)
		if region != "" {
			c.Region = region
		}
		return c
	}

	// Superadmin: create the target account (and delete it at the end).
	admin := newIAM(os.Getenv("SCALITY_ACCESS_KEY"), os.Getenv("SCALITY_SECRET_KEY"))

	accountName := acctest.RandomName("acctest-assumerole")
	createResp, err := admin.CreateAccount(ctx, client.AccountCreateRequest{
		Name:         accountName,
		EmailAddress: accountName + "@example.com",
	})
	if err != nil {
		t.Fatalf("setup: create account: %s", err)
	}
	accountID := createResp.Account.Data.ID
	t.Cleanup(func() {
		if err := admin.DeleteAccount(context.Background(), accountName); err != nil {
			t.Logf("cleanup: delete account %s: %s", accountName, err)
		}
	})

	// Account root credentials — the identity that assumes the role.
	keyResp, err := admin.GenerateAccountAccessKey(ctx, accountName)
	if err != nil {
		t.Fatalf("setup: generate account access key: %s", err)
	}
	accountAK := keyResp.Data.ID
	accountSK := keyResp.Data.Value

	acct := newIAM(accountAK, accountSK)

	// Permissions the assumed session needs to manage the test resources.
	policyName := acctest.RandomName("assumerole-perms")
	permsDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`
	policy, err := acct.CreateManagedPolicy(ctx, accountAK, accountSK, policyName, permsDoc)
	if err != nil {
		t.Fatalf("setup: create managed policy: %s", err)
	}
	policyArn := policy.Arn
	t.Cleanup(func() {
		if err := acct.DeleteManagedPolicy(context.Background(), accountAK, accountSK, policyArn); err != nil {
			t.Logf("cleanup: delete managed policy: %s", err)
		}
	})

	// Role whose trust policy lets the account root assume it.
	roleName := acctest.RandomName("assumerole")
	trustDoc := fmt.Sprintf(
		`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::%s:root"},"Action":"sts:AssumeRole"}]}`,
		accountID,
	)
	role, err := acct.CreateRole(ctx, accountAK, accountSK, roleName, trustDoc)
	if err != nil {
		t.Fatalf("setup: create role: %s", err)
	}
	roleArn := role.Arn
	t.Cleanup(func() {
		if err := acct.DeleteRole(context.Background(), accountAK, accountSK, roleName); err != nil {
			t.Logf("cleanup: delete role: %s", err)
		}
	})

	if err := acct.AttachRolePolicy(ctx, accountAK, accountSK, roleName, policyArn); err != nil {
		t.Fatalf("setup: attach role policy: %s", err)
	}
	t.Cleanup(func() {
		if err := acct.DetachRolePolicy(context.Background(), accountAK, accountSK, roleName, policyArn); err != nil {
			t.Logf("cleanup: detach role policy: %s", err)
		}
	})

	// The assuming identity must be an IAM user, not the account root ("Roles
	// may not be assumed by root accounts"). Create one, grant it sts:AssumeRole,
	// and use its key as the provider's base credentials.
	assumerName := acctest.RandomName("assumer")
	if _, err := acct.CreateUser(ctx, accountAK, accountSK, assumerName); err != nil {
		t.Fatalf("setup: create assumer user: %s", err)
	}
	t.Cleanup(func() {
		if err := acct.DeleteUser(context.Background(), accountAK, accountSK, assumerName); err != nil {
			t.Logf("cleanup: delete assumer user: %s", err)
		}
	})

	assumePolicyDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sts:AssumeRole","Resource":"*"}]}`
	if err := acct.PutUserPolicy(ctx, accountAK, accountSK, assumerName, "assume-role", assumePolicyDoc); err != nil {
		t.Fatalf("setup: put assumer user policy: %s", err)
	}
	t.Cleanup(func() {
		if err := acct.DeleteUserPolicy(context.Background(), accountAK, accountSK, assumerName, "assume-role"); err != nil {
			t.Logf("cleanup: delete assumer user policy: %s", err)
		}
	})

	assumerKey, err := acct.CreateUserAccessKey(ctx, accountAK, accountSK, assumerName)
	if err != nil {
		t.Fatalf("setup: create assumer access key: %s", err)
	}
	assumerAK := assumerKey.AccessKeyId
	assumerSK := assumerKey.SecretAccessKey
	t.Cleanup(func() {
		if err := acct.DeleteUserAccessKey(context.Background(), accountAK, accountSK, assumerName, assumerAK); err != nil {
			t.Logf("cleanup: delete assumer access key: %s", err)
		}
	})

	bucketName := acctest.RandomName("acctest-assumed")
	userName := acctest.RandomName("acctest-assumed-user")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAssumeRoleConfig(assumerAK, assumerSK, roleArn, bucketName, userName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Both resources were created without explicit credentials —
					// only possible via the assumed-role fallback.
					resource.TestCheckResourceAttr("scality_bucket.assumed", "bucket", bucketName),
					resource.TestCheckResourceAttr("scality_user.assumed", "username", userName),
					resource.TestCheckResourceAttrSet("scality_user.assumed", "arn"),
					// Credentials were omitted, so they stay null in state.
					resource.TestCheckNoResourceAttr("scality_bucket.assumed", "account_access_key"),
					resource.TestCheckNoResourceAttr("scality_user.assumed", "account_access_key"),
				),
			},
		},
	})
}

func testAccAssumeRoleConfig(accountAK, accountSK, roleArn, bucketName, userName string) string {
	return fmt.Sprintf(`
provider "scality" {
  access_key           = %q
  secret_key           = %q
  insecure_skip_verify = true

  assume_role {
    role_arn = %q
  }
}

resource "scality_bucket" "assumed" {
  bucket = %q
}

resource "scality_user" "assumed" {
  username = %q
}
`, accountAK, accountSK, roleArn, bucketName, userName)
}
