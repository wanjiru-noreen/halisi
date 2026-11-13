package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"
)

type STKRequest struct {
	ShortCode        string
	Passkey          string
	TransactionType  string
	PhoneNumber      string
	Amount           int64
	AccountReference string
	Description      string
	CallbackURL      string
}

type STKResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
}

type STKClient interface {
	Initiate(context.Context, STKRequest) (STKResponse, error)
}

var ErrSTKRejected = errors.New("M-Pesa rejected the payment request")
var ErrSTKNotSubmitted = errors.New("M-Pesa payment prompt was not submitted")

type Daraja struct {
	BaseURL        string
	ConsumerKey    string
	ConsumerSecret string
	HTTPClient     *http.Client
}

var nairobiTime = func() *time.Location {
	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		return time.FixedZone("EAT", 3*60*60)
	}
	return location
}()

func darajaTimestamp(now time.Time) string {
	return now.In(nairobiTime).Format("20060102150405")
}

func NewDaraja(environment, key, secret string, client *http.Client) (*Daraja, error) {
	var baseURL string
	switch environment {
	case "sandbox":
		baseURL = "https://sandbox.safaricom.co.ke"
	case "production":
		baseURL = "https://api.safaricom.co.ke"
	default:
		return nil, errors.New("MPESA_ENV must be sandbox or production")
	}
	if strings.TrimSpace(key) == "" || strings.TrimSpace(secret) == "" {
		return nil, errors.New("M-Pesa consumer credentials are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Daraja{BaseURL: baseURL, ConsumerKey: key, ConsumerSecret: secret, HTTPClient: client}, nil
}

func (d *Daraja) Initiate(ctx context.Context, input STKRequest) (STKResponse, error) {
	token, err := d.accessToken(ctx)
	if err != nil {
		return STKResponse{}, fmt.Errorf("%w: %v", ErrSTKNotSubmitted, err)
	}
	timestamp := darajaTimestamp(time.Now())
	password := base64.StdEncoding.EncodeToString([]byte(input.ShortCode + input.Passkey + timestamp))
	body := map[string]any{
		"BusinessShortCode": input.ShortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   input.TransactionType,
		"Amount":            input.Amount,
		"PartyA":            input.PhoneNumber,
		"PartyB":            input.ShortCode,
		"PhoneNumber":       input.PhoneNumber,
		"CallBackURL":       input.CallbackURL,
		"AccountReference":  input.AccountReference,
		"TransactionDesc":   input.Description,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return STKResponse{}, fmt.Errorf("%w: %v", ErrSTKNotSubmitted, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.BaseURL+"/mpesa/stkpush/v1/processrequest", bytes.NewReader(encoded))
	if err != nil {
		return STKResponse{}, fmt.Errorf("%w: %v", ErrSTKNotSubmitted, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return STKResponse{}, err
	}
	defer resp.Body.Close()
	var result STKResponse
	if err := decodeDarajaResponse(resp, &result); err != nil {
		return STKResponse{}, err
	}
	if result.ResponseCode != "0" || result.CheckoutRequestID == "" || result.MerchantRequestID == "" {
		if result.ResponseCode != "0" {
			return STKResponse{}, ErrSTKRejected
		}
		return STKResponse{}, errors.New("M-Pesa returned an incomplete payment response")
	}
	return result, nil
}

func (d *Daraja) accessToken(ctx context.Context) (string, error) {
	endpoint := d.BaseURL + "/oauth/v1/generate?grant_type=client_credentials"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(d.ConsumerKey, d.ConsumerSecret)
	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := decodeDarajaResponse(resp, &result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", errors.New("M-Pesa authorization response did not include an access token")
	}
	return result.AccessToken, nil
}

func decodeDarajaResponse(resp *http.Response, target any) error {
	defer io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("M-Pesa request failed with HTTP status %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return errors.New("M-Pesa returned an invalid response")
	}
	return nil
}
