package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

type MPesa struct {
	pool            *pgxpool.Pool
	auth            *Auth
	client          services.STKClient
	publicBaseURL   string
	shortCode       string
	passkey         string
	transactionType string
}

func NewMPesa(pool *pgxpool.Pool, auth *Auth, client services.STKClient, publicBaseURL, shortCode, passkey, transactionType string) *MPesa {
	return &MPesa{pool: pool, auth: auth, client: client, publicBaseURL: strings.TrimRight(publicBaseURL, "/"), shortCode: shortCode, passkey: passkey, transactionType: transactionType}
}

var mpesaPhonePattern = regexp.MustCompile(`^254[17][0-9]{8}$`)
var requestKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

func NormalizeMPesaPhone(raw string) (string, error) {
	phone := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(phone, "+254"):
		phone = phone[1:]
	case strings.HasPrefix(phone, "0") && len(phone) == 10:
		phone = "254" + phone[1:]
	case strings.HasPrefix(phone, "254"):
	default:
		return "", errors.New("enter a valid Kenyan M-Pesa phone number")
	}
	if !mpesaPhonePattern.MatchString(phone) {
		return "", errors.New("enter a valid Kenyan M-Pesa phone number")
	}
	return phone, nil
}

func (h *MPesa) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/mpesa/availability", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": h.client != nil})
	})
	mux.Handle("POST /api/customer/orders/{id}/payments/mpesa", middleware.RequireAuth(h.auth.service, h.auth.cookieName, middleware.RequirePermission(models.CustomerResources, http.HandlerFunc(h.start))))
	mux.HandleFunc("POST /api/mpesa/callback/{token}", h.callback)
}

type mpesaInput struct {
	PhoneNumber string `json:"phone_number"`
}

func (h *MPesa) start(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "M-Pesa payments are not configured."})
		return
	}
	var input mpesaInput
	if !decodeJSON(w, r, &input) {
		return
	}
	phone, err := NormalizeMPesaPhone(input.PhoneNumber)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	requestKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if !requestKeyPattern.MatchString(requestKey) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "A valid Idempotency-Key header is required."})
		return
	}
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orderID <= 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Order not found."})
		return
	}
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Payment is temporarily unavailable."})
		return
	}
	defer tx.Rollback(ctx)
	var orderTotal, orderStatus string
	err = tx.QueryRow(ctx, `SELECT total::text,status FROM orders WHERE id=$1 AND customer_id=$2 FOR UPDATE`, orderID, user.ID).Scan(&orderTotal, &orderStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Order not found."})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Payment is temporarily unavailable."})
		return
	}
	var existingID, existingOrder int64
	var existingStatus, existingPhone string
	err = tx.QueryRow(ctx, `SELECT id, order_id, status, phone_number FROM payments WHERE request_key=$1`, requestKey).Scan(&existingID, &existingOrder, &existingStatus, &existingPhone)
	if err == nil {
		if existingOrder != orderID || existingPhone != phone {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "This Idempotency-Key was already used with different payment details."})
			return
		}
		writeJSON(w, http.StatusAccepted, paymentJSON(existingID, orderID, existingStatus))
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Payment is temporarily unavailable."})
		return
	}
	if orderStatus != "pending" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Only pending orders can start a payment."})
		return
	}
	var activeID int64
	var activeStatus string
	err = tx.QueryRow(ctx, `SELECT id,status FROM payments WHERE order_id=$1 AND provider='mpesa' AND status IN ('pending','processing')`, orderID).Scan(&activeID, &activeStatus)
	if err == nil {
		writeJSON(w, http.StatusAccepted, paymentJSON(activeID, orderID, activeStatus))
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Payment is temporarily unavailable."})
		return
	}
	cents, err := decimalCents(orderTotal)
	if err != nil || cents == 0 || cents%100 != 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "M-Pesa payments require a positive whole-shilling order total."})
		return
	}
	var tokenBytes [32]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Payment is temporarily unavailable."})
		return
	}
	token := hex.EncodeToString(tokenBytes[:])
	tokenHash := sha256.Sum256([]byte(token))
	var paymentID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO payments(order_id,amount,currency,status,provider,request_key,phone_number,callback_token_hash)
		SELECT id,total,currency,'processing','mpesa',$3,$4,$5 FROM orders WHERE id=$1 AND customer_id=$2
		RETURNING id`, orderID, user.ID, requestKey, phone, hex.EncodeToString(tokenHash[:])).Scan(&paymentID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "A payment for this order is already being processed."})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Payment is temporarily unavailable."})
		return
	}
	callbackURL := h.publicBaseURL + "/api/mpesa/callback/" + token
	requestCtx, requestCancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
	defer requestCancel()
	response, err := h.client.Initiate(requestCtx, services.STKRequest{
		ShortCode: h.shortCode, Passkey: h.passkey, TransactionType: h.transactionType,
		PhoneNumber: phone, Amount: cents / 100, AccountReference: "H" + strconv.FormatInt(orderID, 36),
		Description: "HALISI Order", CallbackURL: callbackURL,
	})
	if err != nil {
		if errors.Is(err, services.ErrSTKRejected) || errors.Is(err, services.ErrSTKNotSubmitted) {
			reason := "Payment request was rejected by M-Pesa"
			if errors.Is(err, services.ErrSTKNotSubmitted) {
				reason = "Payment prompt was not submitted to M-Pesa"
			}
			if cleanupErr := h.markPaymentFailed(paymentID, reason); cleanupErr != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"payment": paymentJSON(paymentID, orderID, "processing")["payment"],
					"error":   "We couldn't confirm the payment attempt's status. Do not submit another payment yet; refresh this order shortly or contact support.",
				})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]any{"payment": map[string]any{"id": paymentID, "order_id": orderID, "status": "failed"}, "error": "M-Pesa could not start this payment. You can try again."})
			return
		}
		// A transport timeout can happen after Safaricom accepted the prompt. Keep the
		// attempt processing so a retry cannot charge the customer a second time.
		writeJSON(w, http.StatusAccepted, paymentJSON(paymentID, orderID, "processing"))
		return
	}
	_, _ = h.pool.Exec(requestCtx, `UPDATE payments SET merchant_request_id=$2,checkout_request_id=$3 WHERE id=$1 AND status='processing'`, paymentID, response.MerchantRequestID, response.CheckoutRequestID)
	writeJSON(w, http.StatusAccepted, paymentJSON(paymentID, orderID, "processing"))
}

func (h *MPesa) markPaymentFailed(paymentID int64, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		tag, err := h.pool.Exec(ctx, `UPDATE payments SET status='failed',result_description=$2 WHERE id=$1 AND status='processing'`, paymentID, reason)
		if err == nil {
			if tag.RowsAffected() != 1 {
				return errors.New("payment attempt was not in a failed-transition state")
			}
			return nil
		}
		lastErr = err
		if attempt < 2 {
			timer := time.NewTimer(time.Duration(attempt+1) * 200 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return lastErr
			case <-timer.C:
			}
		}
	}
	return lastErr
}

func paymentJSON(id, orderID int64, status string) map[string]any {
	return map[string]any{"payment": map[string]any{"id": id, "order_id": orderID, "status": status}}
}

type stkCallbackEnvelope struct {
	Body struct {
		STKCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        *int   `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			Metadata          struct {
				Items []struct {
					Name  string          `json:"Name"`
					Value json.RawMessage `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

func (h *MPesa) callback(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid callback."})
		return
	}
	defer r.Body.Close()
	var callback stkCallbackEnvelope
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&callback); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid callback."})
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid callback."})
		return
	}
	data := callback.Body.STKCallback
	if data.ResultCode == nil || data.CheckoutRequestID == "" || len(data.CheckoutRequestID) > 128 || data.MerchantRequestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid callback."})
		return
	}
	tokenHash := sha256.Sum256([]byte(r.PathValue("token")))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Callback could not be processed."})
		return
	}
	defer tx.Rollback(ctx)
	var paymentID, orderID int64
	var paymentStatus, storedCheckoutID, storedMerchantID, storedPhone, amountText, storedTokenHash string
	err = tx.QueryRow(ctx, `
		SELECT id,order_id,status,COALESCE(checkout_request_id,''),COALESCE(merchant_request_id,''),phone_number,amount::text,callback_token_hash
		FROM payments WHERE provider='mpesa' AND callback_token_hash=$1 FOR UPDATE`, hex.EncodeToString(tokenHash[:])).
		Scan(&paymentID, &orderID, &paymentStatus, &storedCheckoutID, &storedMerchantID, &storedPhone, &amountText, &storedTokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Callback not recognized."})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Callback could not be processed."})
		return
	}
	if subtle.ConstantTimeCompare([]byte(storedTokenHash), []byte(hex.EncodeToString(tokenHash[:]))) != 1 || (storedCheckoutID != "" && storedCheckoutID != data.CheckoutRequestID) || (storedMerchantID != "" && storedMerchantID != data.MerchantRequestID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Callback did not match the payment attempt."})
		return
	}
	if paymentStatus == "completed" || paymentStatus == "failed" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
		return
	}
	var receipt, callbackPhone string
	var callbackAmount int64
	for _, item := range data.Metadata.Items {
		switch item.Name {
		case "MpesaReceiptNumber":
			_ = json.Unmarshal(item.Value, &receipt)
		case "PhoneNumber":
			if err := json.Unmarshal(item.Value, &callbackPhone); err != nil {
				var numeric json.Number
				if err := json.Unmarshal(item.Value, &numeric); err == nil {
					callbackPhone = numeric.String()
				}
			}
		case "Amount":
			var raw json.Number
			if err := json.Unmarshal(item.Value, &raw); err == nil {
				callbackAmount, _ = raw.Int64()
			}
		}
	}
	newStatus := "failed"
	var receiptValue any
	if *data.ResultCode == 0 {
		cents, amountErr := decimalCents(amountText)
		if amountErr != nil || cents%100 != 0 || callbackAmount != cents/100 || callbackPhone != storedPhone || receipt == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Callback payment details did not match the order."})
			return
		}
		newStatus = "completed"
		receiptValue = receipt
	}
	_, err = tx.Exec(ctx, `UPDATE payments SET status=$2,reference=$3,result_code=$4,result_description=$5,checkout_request_id=$6,merchant_request_id=$7 WHERE id=$1 AND status IN ('pending','processing')`, paymentID, newStatus, receiptValue, *data.ResultCode, truncate(data.ResultDesc, 500), data.CheckoutRequestID, data.MerchantRequestID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Callback could not be processed."})
		return
	}
	if newStatus == "completed" {
		if _, err := tx.Exec(ctx, `UPDATE orders SET status='confirmed' WHERE id=$1 AND status='pending'`, orderID); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Callback could not be processed."})
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Callback could not be processed."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func truncate(value string, max int) string {
	if len(value) > max {
		return strings.ToValidUTF8(value[:max], "�")
	}
	return value
}
