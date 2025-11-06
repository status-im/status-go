package settings

import (
	"encoding/json"
	"testing"
)

func TestUnmarshal_ThirdpartyServicesEnabled_DefaultsToTrueWhenMissing(t *testing.T) {
	var s Settings
	data := []byte(`{}`)

	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !s.ThirdpartyServicesEnabled {
		t.Fatalf("expected ThirdpartyServicesEnabled to be true when omitted, got false")
	}
}

func TestUnmarshal_ThirdpartyServicesEnabled_RespectsTrue(t *testing.T) {
	var s Settings
	data := []byte(`{"thirdparty_services_enabled": true}`)

	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !s.ThirdpartyServicesEnabled {
		t.Fatalf("expected ThirdpartyServicesEnabled to be true when explicitly set to true, got false")
	}
}

func TestUnmarshal_ThirdpartyServicesEnabled_RespectsFalse(t *testing.T) {
	var s Settings
	data := []byte(`{"thirdparty_services_enabled": false}`)

	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if s.ThirdpartyServicesEnabled {
		t.Fatalf("expected ThirdpartyServicesEnabled to be false when explicitly set to false, got true")
	}
}

func TestUnmarshal_ThirdpartyServicesEnabled_NullTreatedAsMissing_DefaultsToTrue(t *testing.T) {
	var s Settings
	data := []byte(`{"thirdparty_services_enabled": null}`)

	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !s.ThirdpartyServicesEnabled {
		t.Fatalf("expected ThirdpartyServicesEnabled to be true when null (treated as missing), got false")
	}
}

func TestMarshal_ThirdpartyServicesEnabled_IncludedWhenTrue(t *testing.T) {
	s := Settings{ThirdpartyServicesEnabled: true}

	b, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal of marshaled bytes failed: %v", err)
	}

	// When true we expect the key to be present and true
	v, ok := m["thirdparty_services_enabled"]
	if !ok {
		t.Fatalf("expected thirdparty_services_enabled key to be present when true")
	}
	if v != true {
		t.Fatalf("expected thirdparty_services_enabled to be true in JSON, got %v", v)
	}
}

func TestMarshal_ThirdpartyServicesEnabled_IncludedWhenFalse(t *testing.T) {
	s := Settings{ThirdpartyServicesEnabled: false}

	b, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal of marshaled bytes failed: %v", err)
	}

	// The field tag includes `omitempty`, so when false the key is omitted.
	if _, ok := m["thirdparty_services_enabled"]; ok {
		// If present, verify it's false
		if m["thirdparty_services_enabled"] != false {
			t.Fatalf("expected thirdparty_services_enabled to be false in JSON when present, got %v", m["thirdparty_services_enabled"])
		}
	} else {
		// Key is omitted, which is not acceptable for false
		t.Fatalf("expected thirdparty_services_enabled key to be present when false")
	}
}
