package client

import "os"

// ImportAccountCreds returns the per-account credentials from
// SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY. ok is true only
// when both are set, gating the secret-free ImportState path.
func ImportAccountCreds() (ak, sk string, ok bool) {
	ak = os.Getenv("SCALITY_ACCOUNT_ACCESS_KEY")
	sk = os.Getenv("SCALITY_ACCOUNT_SECRET_KEY")
	return ak, sk, ak != "" && sk != ""
}
