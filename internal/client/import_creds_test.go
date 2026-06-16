package client

import "testing"

func TestImportAccountCreds(t *testing.T) {
	cases := []struct {
		name   string
		ak, sk string
		wantAK string
		wantSK string
		wantOK bool
	}{
		{name: "both set", ak: "AKIA", sk: "secret", wantAK: "AKIA", wantSK: "secret", wantOK: true},
		{name: "only ak set", ak: "AKIA", sk: "", wantAK: "AKIA", wantSK: "", wantOK: false},
		{name: "only sk set", ak: "", sk: "secret", wantAK: "", wantSK: "secret", wantOK: false},
		{name: "neither set", ak: "", sk: "", wantAK: "", wantSK: "", wantOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SCALITY_ACCOUNT_ACCESS_KEY", tc.ak)
			t.Setenv("SCALITY_ACCOUNT_SECRET_KEY", tc.sk)
			ak, sk, ok := ImportAccountCreds()
			if ak != tc.wantAK || sk != tc.wantSK || ok != tc.wantOK {
				t.Fatalf("ImportAccountCreds() = (%q, %q, %v), want (%q, %q, %v)", ak, sk, ok, tc.wantAK, tc.wantSK, tc.wantOK)
			}
		})
	}
}
