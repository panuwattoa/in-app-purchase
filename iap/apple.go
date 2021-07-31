package iap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

const (
	AppleUrlSandbox    = "https://sandbox.itunes.apple.com/verifyReceipt"
	AppleUrlProduction = "https://buy.itunes.apple.com/verifyReceipt"
)

const (
	AppleReceiptIsValid   = 0
	AppleReceiptIsSandbox = 21007
)

var (
	ErrNon200Apple = errors.New("non 200 response from apple")
)

const (
	AppleSandboxEnv    = "Sandbox"
	AppleProductionEnv = "Production"
)

type ValidateReceiptAppleResponse struct {
	IsRetryable bool             `json:"is-retryable"` // If true, must be retried later.
	Status      int              `json:"status"`
	Receipt     *ResponseReceipt `json:"receipt"`
	Environment string           `json:"environment"` // possible values: 'Sandbox', 'Production'.
}

type ResponseReceipt struct {
	OriginalPurchaseDateMs string   `json:"original_purchase_date_ms"`
	InApp                  []*InApp `json:"in_app"`
}

type InApp struct {
	OriginalTransactionID string               `json:"original_transaction_id"`
	TransactionId         string               `json:"transaction_id"` // Different than OriginalTransactionId if the user Auto-renews subscription or restores a purchase.
	ProductID             string               `json:"product_id"`
	ExpiresDateMs         string               `json:"expires_date_ms"` // Only returned for Subscription expiration or renewal date.
	PurchaseDateMs        string               `json:"purchase_date_ms"`
	CancellationDateMs    string               `json:"cancellation_date_ms"` // canceled a transaction This field is only present for refunded transactions
	CancellationReason    string               `json:"cancellation_reason"`  // reason for a refunded transaction Possible values: 1, 0
	PendingRenewalInfo    []PendingRenewalInfo `json:"pending_renewal_info"` // Only returned for app receipts that contain auto-renewable subscriptions.
}

type PendingRenewalInfo struct {
	AutoRenewStatus string `json:"auto_renew_status"` // Possible values: 1, 0
}

// ValidateReceiptApple this function will check against both the production and sandbox Apple URLs follow by Apple suggestion.
// return response struct and raw data. Do what ever you want.
func ValidateReceiptApple(ctx context.Context, httpc *http.Client, receipt, password string) (*ValidateReceiptAppleResponse, []byte, error) {
	resp, raw, err := requestValidateWithUrl(ctx, httpc, AppleUrlProduction, receipt, password, false)
	if err != nil {
		return nil, nil, err
	}

	switch resp.Status {
	case AppleReceiptIsSandbox:
		// Receipt should be checked with the Apple sandbox.
		return requestValidateWithUrl(ctx, httpc, AppleUrlSandbox, receipt, password, false)
	}

	return resp, raw, nil
}

// ValidateSubscriptionReceiptApple this function for purchase subscription will check against both the production and sandbox Apple URLs follow by Apple suggestion.
// required password
// return response struct and raw data. Do what ever you want.
func ValidateSubscriptionReceiptApple(ctx context.Context, httpc *http.Client, receipt, password string) (*ValidateReceiptAppleResponse, []byte, error) {
	resp, raw, err := requestValidateWithUrl(ctx, httpc, AppleUrlProduction, receipt, password, true)
	if err != nil {
		return nil, nil, err
	}

	switch resp.Status {
	case AppleReceiptIsSandbox:
		// Receipt should be checked with the Apple sandbox.
		return requestValidateWithUrl(ctx, httpc, AppleUrlSandbox, receipt, password, true)
	}

	return resp, raw, nil
}

func requestValidateWithUrl(ctx context.Context, httpc *http.Client, url, receipt, password string, isSubscription bool) (*ValidateReceiptAppleResponse, []byte, error) {
	if len(url) < 1 {
		return nil, nil, errors.New("'url' is empty")
	}

	if len(receipt) < 1 {
		return nil, nil, errors.New("'receipt' is empty")
	}

	// optional use only subscription validation
	if len(password) < 1 && isSubscription {
		return nil, nil, errors.New("'password' is empty")
	}

	payload := map[string]interface{}{
		"receipt-data":             receipt,
		"exclude-old-transactions": true,
		"password":                 password,
	}

	var w bytes.Buffer
	if err := json.NewEncoder(&w).Encode(&payload); err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, &w)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}

		var out ValidateReceiptAppleResponse
		if err := json.Unmarshal(buf, &out); err != nil {
			return nil, nil, err
		}
		return &out, buf, nil
	default:
		return nil, nil, ErrNon200Apple
	}
}
