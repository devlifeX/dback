package notify

import "testing"

func TestValidateHTTPSURLBlocksLocalhost(t *testing.T) {
	if _, err := ValidateHTTPSURL("https://localhost/hook"); err == nil {
		t.Fatal("expected localhost to be blocked")
	}
}

func TestValidateHTTPSURLBlocksPrivateIPLiteral(t *testing.T) {
	if _, err := ValidateHTTPSURL("https://127.0.0.1/hook"); err == nil {
		t.Fatal("expected loopback ip to be blocked")
	}
	if _, err := ValidateHTTPSURL("https://10.0.0.1/hook"); err == nil {
		t.Fatal("expected private ip to be blocked")
	}
}

func TestValidateHTTPSURLRequiresHTTPS(t *testing.T) {
	if _, err := ValidateHTTPSURL("http://example.com/hook"); err == nil {
		t.Fatal("expected http to be rejected")
	}
}

func TestValidateHTTPMethodAllowlist(t *testing.T) {
	if err := ValidateHTTPMethod("POST"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHTTPMethod("DELETE"); err == nil {
		t.Fatal("expected DELETE to be rejected")
	}
}
