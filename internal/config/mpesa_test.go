package config

import "testing"

func clearMPesaEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"MPESA_ENV", "MPESA_CONSUMER_KEY", "MPESA_CONSUMER_SECRET", "MPESA_SHORT_CODE", "MPESA_PASSKEY", "MPESA_TRANSACTION_TYPE"} {
		t.Setenv(name, "")
	}
}

func TestLoadMPesaDisabledWhenUnconfigured(t *testing.T) {
	clearMPesaEnv(t)
	cfg, err := LoadMPesa()
	if err != nil || cfg != nil {
		t.Fatalf("LoadMPesa() = (%v, %v), want (nil, nil)", cfg, err)
	}
}

func TestLoadMPesaRequiresCompleteConfiguration(t *testing.T) {
	clearMPesaEnv(t)
	t.Setenv("MPESA_ENV", "sandbox")
	if _, err := LoadMPesa(); err == nil {
		t.Fatal("expected partial configuration to fail")
	}
}

func TestLoadMPesaAcceptsSandbox(t *testing.T) {
	clearMPesaEnv(t)
	for name, value := range map[string]string{
		"MPESA_ENV": "sandbox", "MPESA_CONSUMER_KEY": "key", "MPESA_CONSUMER_SECRET": "secret",
		"MPESA_SHORT_CODE": "174379", "MPESA_PASSKEY": "pass", "MPESA_TRANSACTION_TYPE": "CustomerPayBillOnline",
	} {
		t.Setenv(name, value)
	}
	cfg, err := LoadMPesa()
	if err != nil || cfg == nil || cfg.Environment != "sandbox" || cfg.ShortCode != "174379" {
		t.Fatalf("LoadMPesa() = (%+v, %v)", cfg, err)
	}
}
