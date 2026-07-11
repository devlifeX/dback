package operation_test

import (
	"encoding/json"
	"testing"

	"dback/internal/operation"
)

func TestDecodeRestoreParams(t *testing.T) {
	raw := json.RawMessage(`{"record_id":"rec1","destination_profile_id":"host2"}`)
	p, err := operation.DecodeParams(operation.KindRestore, raw)
	if err != nil {
		t.Fatal(err)
	}
	rp := p.(operation.RestoreParams)
	if rp.RecordID != "rec1" || rp.DestinationProfileID != "host2" {
		t.Fatalf("unexpected params: %+v", rp)
	}
}

func TestDecodeDeepVerifyParams(t *testing.T) {
	raw := json.RawMessage(`{"record_id":"rec1","destination_profile_id":"host2"}`)
	p, err := operation.DecodeParams(operation.KindDeepVerify, raw)
	if err != nil {
		t.Fatal(err)
	}
	dp := p.(operation.DeepVerifyParams)
	if dp.RecordID != "rec1" {
		t.Fatalf("unexpected params: %+v", dp)
	}
}
