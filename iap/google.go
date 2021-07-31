package iap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2/google"
	goJWT "golang.org/x/oauth2/jwt"
)

type ReceiptGoogle struct {
	OrderID       string `json:"orderId"`
	PackageName   string `json:"packageName"`
	ProductID     string `json:"productId"`
	PurchaseState int    `json:"purchaseState"`
	PurchaseTime  int64  `json:"purchaseTime"`
	PurchaseToken string `json:"purchaseToken"`
}

type ReceiptGoogleResponse struct {
	AcknowledgementState int    `json:"acknowledgementState"`
	ConsumptionState     int    `json:"consumptionState"`
	DeveloperPayload     string `json:"developerPayload"`
	Kind                 string `json:"kind"`
	OrderId              string `json:"orderId"`
	PurchaseState        int    `json:"purchaseState"`
	PurchaseTimeMillis   string `json:"purchaseTimeMillis"`
	PurchaseType         int    `json:"purchaseType"`
	RegionCode           string `json:"regionCode"`
}

type ReceiptSubscriptionGoogleResponse struct {
	AcknowledgementState int    `json:"acknowledgementState"`
	DeveloperPayload     string `json:"developerPayload"`
	Kind                 string `json:"kind"`
	OrderId              string `json:"orderId"`
	PurchaseType         int    `json:"purchaseType"`
	// This field is only set if this purchase was not made using the standard in-app billing flow.
	// Possible values are: 0. Test (i.e. purchased from a license testing account) 1. Promo (i.e. purchased using a promo code)
	AutoRenewing                 bool   `json:"autoRenewing"`
	StartSubscriptionTimeMillis  int64  `json:"startTimeMillis,string,omitempty"`
	ExpirySubscriptionTimeMillis int64  `json:"expiryTimeMillis,string,omitempty"`
	LinkedPurchaseToken          string `json:"linkedPurchaseToken"`
	CancelReason                 int    `json:"cancelReason"`
	//0 User canceled the subscription
	//1 Subscription was canceled by the system, for example because of a billing problem
	//2 Subscription was replaced with a new subscription
	//3 Subscription was canceled by the developer
	UserCancellationTimeMillis int `json:"userCancellationTimeMillis"`
	// Only present if cancelReason is 0.
	PaymentState int `json:"paymentState"`
	//0 Payment pending
	//1 Payment received
	//2 Free trial
	//3 Pending deferred upgrade/downgrade
}

var (
	ErrNon200ServiceGoogle = errors.New("non 200 response from Google service")
)

var conf *goJWT.Config

// ValidateReceiptGoogle validate an IAP receipt with the Android Publisher API and the Google credentials.
func ValidateReceiptGoogle(ctx context.Context, httpc *http.Client, clientEmail string, privateKey string, receipt string) (*ReceiptGoogleResponse, *ReceiptGoogle, []byte, error) {
	if len(receipt) < 1 {
		return nil, nil, nil, errors.New("'receipt' is empty")
	}

	token, err := getGoolgeAccessToken(ctx, clientEmail, privateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	return requestValidateReceiptGoogle(ctx, httpc, token, receipt)
}

// ValidateSubscriptionReceiptGoogle validate an IAP receipt with subscription type
func ValidateSubscriptionReceiptGoogle(ctx context.Context, httpc *http.Client, clientEmail string, privateKey string, receipt string) (*ReceiptSubscriptionGoogleResponse, *ReceiptGoogle, []byte, error) {
	if len(receipt) < 1 {
		return nil, nil, nil, errors.New("'receipt' is empty")
	}

	token, err := getGoolgeAccessToken(ctx, clientEmail, privateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	return requestValidateSubscriptionReceiptGoogle(ctx, httpc, token, receipt)
}

func requestValidateReceiptGoogle(ctx context.Context, httpc *http.Client, token string, receipt string) (*ReceiptGoogleResponse, *ReceiptGoogle, []byte, error) {

	gr, err := decodeReceipt(receipt)
	if err != nil {
		return nil, nil, nil, err
	}

	u := &url.URL{
		Host:     "androidpublisher.googleapis.com",
		Path:     fmt.Sprintf("androidpublisher/v3/applications/%s/purchases/products/%s/tokens/%s", gr.PackageName, gr.ProductID, gr.PurchaseToken),
		RawQuery: fmt.Sprintf("access_token=%s", token),
		Scheme:   "https",
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}

	defer resp.Body.Close()
	log.Printf("resp.StatusCode %v", resp.StatusCode)

	switch resp.StatusCode {

	case 200:
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, nil, err
		}

		out := &ReceiptGoogleResponse{}
		if err := json.Unmarshal(buf, &out); err != nil {
			return nil, nil, nil, err
		}

		return out, gr, buf, nil
	default:
		return nil, nil, nil, ErrNon200ServiceGoogle
	}
}

func requestValidateSubscriptionReceiptGoogle(ctx context.Context, httpc *http.Client, token string, receipt string) (*ReceiptSubscriptionGoogleResponse, *ReceiptGoogle, []byte, error) {
	if len(token) < 1 {
		return nil, nil, nil, errors.New("'token' is empty")
	}

	if len(receipt) < 1 {
		return nil, nil, nil, errors.New("'receipt' is empty")
	}

	gr, err := decodeReceipt(receipt)
	if err != nil {
		return nil, nil, nil, err
	}

	u := &url.URL{
		Host:     "androidpublisher.googleapis.com",
		Path:     fmt.Sprintf("androidpublisher/v3/applications/%s/purchases/subscriptions/%s/tokens/%s", gr.PackageName, gr.ProductID, gr.PurchaseToken),
		RawQuery: fmt.Sprintf("access_token=%s", token),
		Scheme:   "https",
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}

	defer resp.Body.Close()
	log.Printf("resp.StatusCode %v", resp.StatusCode)

	switch resp.StatusCode {

	case 200:
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, nil, err
		}

		out := &ReceiptSubscriptionGoogleResponse{}
		if err := json.Unmarshal(buf, &out); err != nil {
			return nil, nil, nil, err
		}

		return out, gr, buf, nil
	default:
		return nil, nil, nil, ErrNon200ServiceGoogle
	}
}

// getGoolgeAccessToken returns a TokenSource which repeatedly returns the
// same token as long as it's valid,
func getGoolgeAccessToken(ctx context.Context, clientEmail string, privateKey string) (string, error) {
	if len(clientEmail) < 1 {
		return "", errors.New("'clientEmail' is empty")
	}

	if len(privateKey) < 1 {
		return "", errors.New("'privateKey' is empty")
	}
	const authUrl = "https://accounts.google.com/o/oauth2/token"
	if conf == nil {
		now := time.Now()
		conf = &goJWT.Config{
			Email: clientEmail,
			// The contents of your RSA private key or your PEM file
			// that contains a private key.
			// If you have a p12 file instead, you
			// can use `openssl` to export the private key into a pem file.
			//
			//    $ openssl pkcs12 -in key.p12 -passin pass:notasecret -out key.pem -nodes
			//
			// The field only supports PEM containers with no passphrase.
			// The openssl command will convert p12 keys to passphrase-less PEM containers.
			PrivateKey: []byte(privateKey),
			Scopes: []string{
				"https://www.googleapis.com/auth/androidpublisher",
			},
			TokenURL: google.JWTTokenURL,
			Audience: authUrl,
			Expires:  time.Duration(now.Add(1 * time.Hour).Unix()),
		}
	}

	token, err := conf.TokenSource(ctx).Token()
	return token.AccessToken, err
}

// The standard google receipt structure:
//   "{\"json\":\"{\\\"orderId\\\":\\\"GPA.xxxx-xxxx-xxxx-xxxxx\\\",\\\"packageName\\\":\\\"com.xxx.xxx\\\",\\\"productId\\\":\\\"xxx.xxx.xx\\\",
//       \\\"purchaseTime\\\":1607721533824,\\\"purchaseState\\\":0,\\\"purchaseToken\\\":\\\"xxxx\\\",
//       \\\"acknowledged\\\":false}\",\"signature\":\"xxxxx\",\"skuDetails\":\"{\\\"productId\\\":\\\"xxx.xxx.xx\\\",
//       \\\"type\\\":\\\"inapp\\\",\\\"price\\\":\\\"\\u0e3f29.00\\\",\\\"price_amount_micros\\\":29000000,
//       \\\"price_currency_code\\\":\\\"THB\\\",\\\"title\\\":\\\"xxx\\\",\\\"description\\\":\\\"xxxxx\\\",
//       \\\"skuDetailsToken\\\":\\\"AEuhp4IhWdExxxxxxxxxxx\\\"}\"}"
func decodeReceipt(receipt string) (*ReceiptGoogle, error) {
	var wrapper map[string]interface{}
	if err := json.Unmarshal([]byte(receipt), &wrapper); err != nil {
		return nil, err
	}

	unwrapped, ok := wrapper["json"].(string)
	if !ok {
		return nil, errors.New("'json' field not found, receipt is malformed")
	}

	var gr ReceiptGoogle
	if err := json.Unmarshal([]byte(unwrapped), &gr); err != nil {
		return nil, err
	}
	return &gr, nil
}
