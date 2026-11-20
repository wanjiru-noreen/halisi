package config

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

type MPesa struct {
	Environment     string
	ConsumerKey     string
	ConsumerSecret  string
	ShortCode       string
	Passkey         string
	TransactionType string
}

var shortCodePattern = regexp.MustCompile(`^[0-9]{5,7}$`)

// LoadMPesa disables M-Pesa when no M-Pesa variables are present. Partial
// configuration is rejected so a deployment cannot silently misconfigure payments.
func LoadMPesa() (*MPesa, error) {
	values := []string{
		os.Getenv("MPESA_ENV"), os.Getenv("MPESA_CONSUMER_KEY"), os.Getenv("MPESA_CONSUMER_SECRET"),
		os.Getenv("MPESA_SHORT_CODE"), os.Getenv("MPESA_PASSKEY"), os.Getenv("MPESA_TRANSACTION_TYPE"),
	}
	configured := false
	for _, value := range values {
		configured = configured || strings.TrimSpace(value) != ""
	}
	if !configured {
		return nil, nil
	}
	cfg := &MPesa{
		Environment: strings.TrimSpace(values[0]), ConsumerKey: strings.TrimSpace(values[1]),
		ConsumerSecret: strings.TrimSpace(values[2]), ShortCode: strings.TrimSpace(values[3]),
		Passkey: strings.TrimSpace(values[4]), TransactionType: strings.TrimSpace(values[5]),
	}
	if cfg.Environment != "sandbox" && cfg.Environment != "production" {
		return nil, errors.New("MPESA_ENV must be sandbox or production")
	}
	if cfg.ConsumerKey == "" || cfg.ConsumerSecret == "" || cfg.Passkey == "" {
		return nil, errors.New("MPESA_CONSUMER_KEY, MPESA_CONSUMER_SECRET, and MPESA_PASSKEY are required")
	}
	if !shortCodePattern.MatchString(cfg.ShortCode) {
		return nil, errors.New("MPESA_SHORT_CODE must contain 5 to 7 digits")
	}
	if cfg.TransactionType != "CustomerPayBillOnline" && cfg.TransactionType != "CustomerBuyGoodsOnline" {
		return nil, errors.New("MPESA_TRANSACTION_TYPE must be CustomerPayBillOnline or CustomerBuyGoodsOnline")
	}
	return cfg, nil
}
