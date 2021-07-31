package validate

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/panuwattoa/in-app-purchase/iap"
)

// Validation Provider
type Store int32

const (
	// Apple App Store
	APPLE_APP_STORE Store = 0
	// Google Play Store
	GOOGLE_PLAY_STORE Store = 1
)

// Environment where the purchase took place
type Environment int32

const (
	// Unknown environment.
	UNKNOWN Environment = 0
	// Sandbox /test environment.
	SANDBOX Environment = 1
	// Production environment.
	PRODUCTION Environment = 2
)

var (
	ErrPurchasesListInvalidCursor = errors.New("purchases list cursor invalid")
	ErrUnavailableTryAgain        = errors.New("Apple IAP verification is currently unavailable")
	ErrFailedPrecondition         = errors.New("Invalid Receipt")
	ErrPurchaseReceiptAlreadySeen = errors.New("Purchase Receipt Already Seen")
)

type ValidatePurchaseResponse struct {
	// Newly seen validated purchases.
	ValidatedPurchases []*ValidatedPurchase `json:"validated_purchases,omitempty"`
}

type ValidatedPurchase struct {
	// Purchase Product ID.
	ProductId string `json:"product_id,omitempty"`
	// Purchase Transaction ID.
	TransactionId string `json:"transaction_id,omitempty"`
	// Store identifier
	Store Store `json:"store,omitempty"`
	// UNIX Timestamp when the purchase was done.
	PurchaseTime int64 `json:"purchase_time,omitempty"`
	// UNIX Timestamp when the receipt validation was stored in DB.
	CreateTime int64 `json:"create_time,omitempty"`
	// UNIX Timestamp when the receipt validation was updated in DB.
	UpdateTime int64 `json:"update_time,omitempty"`
	// Raw provider validation response.
	ProviderResponse string `json:"provider_response,omitempty"`
	// Whether the purchase was done in production or sandbox environment.
	Environment Environment `json:"environment,omitempty"`
}

type Purchase struct {
	userID        string
	store         Store
	productId     string
	transactionId string
	rawRequest    string
	rawResponse   string
	purchaseTime  time.Time
	createTime    time.Time // Set by storePurchases
	updateTime    time.Time // Set by storePurchases
	environment   Environment
}

type SubscriptionPurchase struct {
	Purchase
	AutoRenew   bool
	ExpiresTime time.Time
}

type Validate struct {
	Storage Storage
	// ApplePassword optional
	ApplePassword string
	GoogleConfig  IAPGoogleConfig
}

type IAPGoogleConfig struct {
	ClientEmail string `json:"client_email" usage:"Google Service Account client email."`
	PrivateKey  string `json:"private_key" usage:"Google Service Account private key."`
}

type Storage interface {
	StorePurchases(ctx context.Context, sp []*Purchase) ([]*Purchase, error)
	StoreSubscriptionPurchases(ctx context.Context, sp []*SubscriptionPurchase) ([]*SubscriptionPurchase, error)
}

func NewValidate(sg Storage, applePassword string, gc IAPGoogleConfig) *Validate {
	return &Validate{
		Storage:       sg,
		ApplePassword: applePassword,
		GoogleConfig:  gc,
	}
}

var httpc = &http.Client{Timeout: 5 * time.Second}

func (v *Validate) PurchasesApple(ctx context.Context, userID, receipt string) (*ValidatePurchaseResponse, error) {
	validation, raw, err := iap.ValidateReceiptApple(ctx, httpc, receipt, "")
	if err != nil {
		return nil, err
	}

	if validation.Status != iap.AppleReceiptIsValid {
		// TODO: log to DB
		if validation.IsRetryable == true {
			return nil, ErrUnavailableTryAgain
		}
		return nil, ErrFailedPrecondition
	}

	env := PRODUCTION
	if validation.Environment == iap.AppleSandboxEnv {
		env = SANDBOX
	}

	storagePurchases := make([]*Purchase, 0, len(validation.Receipt.InApp))
	for _, purchase := range validation.Receipt.InApp {
		pt, err := strconv.Atoi(purchase.PurchaseDateMs)
		if err != nil {
			return nil, err
		}

		storagePurchases = append(storagePurchases, &Purchase{
			userID:        userID,
			store:         APPLE_APP_STORE,
			productId:     purchase.ProductID,
			transactionId: purchase.TransactionId,
			rawResponse:   string(raw),
			rawRequest:    receipt,
			purchaseTime:  parseMillisecondUnixTimestamp(pt),
			environment:   env,
		})
	}

	purchases, err := v.Storage.StorePurchases(ctx, storagePurchases)
	if err != nil {
		return nil, err
	}

	if len(purchases) < 1 {
		return nil, ErrPurchaseReceiptAlreadySeen
	}

	validatedPurchases := make([]*ValidatedPurchase, 0, len(purchases))
	for _, p := range purchases {
		validatedPurchases = append(validatedPurchases, &ValidatedPurchase{
			ProductId:        p.productId,
			TransactionId:    p.transactionId,
			Store:            p.store,
			PurchaseTime:     p.purchaseTime.Unix(),
			CreateTime:       p.createTime.Unix(),
			UpdateTime:       p.updateTime.Unix(),
			ProviderResponse: string(raw),
			Environment:      p.environment,
		})
	}

	return &ValidatePurchaseResponse{
		ValidatedPurchases: validatedPurchases,
	}, nil
}

func (v *Validate) PurchaseGoogle(ctx context.Context, userID string, receipt string) (*ValidatePurchaseResponse, error) {
	_, gReceipt, raw, err := iap.ValidateReceiptGoogle(ctx, httpc, v.GoogleConfig.ClientEmail, v.GoogleConfig.PrivateKey, receipt)
	if err != nil {
		return nil, err
	}
	purchases, err := v.Storage.StorePurchases(ctx, []*Purchase{
		{
			userID:        userID,
			store:         GOOGLE_PLAY_STORE,
			productId:     gReceipt.ProductID,
			transactionId: gReceipt.PurchaseToken,
			rawRequest:    receipt,
			rawResponse:   string(raw),
			purchaseTime:  parseMillisecondUnixTimestamp(int(gReceipt.PurchaseTime)),
			environment:   UNKNOWN,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(purchases) < 1 {
		return nil, ErrPurchaseReceiptAlreadySeen
	}

	validatedPurchases := make([]*ValidatedPurchase, 0, len(purchases))
	for _, p := range purchases {
		validatedPurchases = append(validatedPurchases, &ValidatedPurchase{
			ProductId:        p.productId,
			TransactionId:    p.transactionId,
			Store:            p.store,
			PurchaseTime:     p.purchaseTime.Unix(),
			CreateTime:       p.createTime.Unix(),
			UpdateTime:       p.updateTime.Unix(),
			ProviderResponse: string(raw),
			Environment:      p.environment,
		})
	}

	return &ValidatePurchaseResponse{
		ValidatedPurchases: validatedPurchases,
	}, nil
}

func (v *Validate) PurchaseSubscriptionGoogle(ctx context.Context, userID string, receipt string) (*ValidatePurchaseResponse, error) {
	g, gReceipt, raw, err := iap.ValidateSubscriptionReceiptGoogle(ctx, httpc, v.GoogleConfig.ClientEmail, v.GoogleConfig.PrivateKey, receipt)
	if err != nil {
		return nil, err
	}
	purchases, err := v.Storage.StoreSubscriptionPurchases(ctx, []*SubscriptionPurchase{
		{
			Purchase: Purchase{
				userID:        userID,
				store:         GOOGLE_PLAY_STORE,
				productId:     gReceipt.ProductID,
				transactionId: gReceipt.PurchaseToken,
				rawRequest:    receipt,
				rawResponse:   string(raw),
				purchaseTime:  parseMillisecondUnixTimestamp(int(gReceipt.PurchaseTime)),
				environment:   UNKNOWN,
			},
			AutoRenew:   g.AutoRenewing,
			ExpiresTime: parseMillisecondUnixTimestamp(int(g.ExpirySubscriptionTimeMillis)),
		},
	})
	if err != nil {
		return nil, err
	}

	if len(purchases) < 1 {
		return nil, ErrPurchaseReceiptAlreadySeen
	}

	validatedPurchases := make([]*ValidatedPurchase, 0, len(purchases))
	for _, p := range purchases {
		validatedPurchases = append(validatedPurchases, &ValidatedPurchase{
			ProductId:        p.productId,
			TransactionId:    p.transactionId,
			Store:            p.store,
			PurchaseTime:     p.purchaseTime.Unix(),
			CreateTime:       p.createTime.Unix(),
			UpdateTime:       p.updateTime.Unix(),
			ProviderResponse: string(raw),
			Environment:      p.environment,
		})
	}

	return &ValidatePurchaseResponse{
		ValidatedPurchases: validatedPurchases,
	}, nil
}

func (v *Validate) PurchasesSubscriptionApple(ctx context.Context, userID, receipt string) (*ValidatePurchaseResponse, error) {
	validation, raw, err := iap.ValidateReceiptApple(ctx, httpc, receipt, v.ApplePassword)
	if err != nil {
		return nil, err
	}

	if validation.Status != iap.AppleReceiptIsValid {
		// TODO: log to DB
		if validation.IsRetryable == true {
			return nil, ErrUnavailableTryAgain
		}
		return nil, ErrFailedPrecondition
	}

	env := PRODUCTION
	if validation.Environment == iap.AppleSandboxEnv {
		env = SANDBOX
	}

	storagePurchases := make([]*SubscriptionPurchase, 0, len(validation.Receipt.InApp))
	for _, purchase := range validation.Receipt.InApp {
		pt, err := strconv.Atoi(purchase.PurchaseDateMs)
		if err != nil {
			return nil, err
		}

		exp, err := strconv.Atoi(purchase.ExpiresDateMs)
		if err != nil {
			return nil, err
		}
		isAutoRenew := false
		if len(purchase.PendingRenewalInfo) > 0 {
			isAutoRenew = purchase.PendingRenewalInfo[0].AutoRenewStatus == "1"
		}
		storagePurchases = append(storagePurchases, &SubscriptionPurchase{
			Purchase: Purchase{
				userID:        userID,
				store:         APPLE_APP_STORE,
				productId:     purchase.ProductID,
				transactionId: purchase.TransactionId,
				rawResponse:   string(raw),
				rawRequest:    receipt,
				purchaseTime:  parseMillisecondUnixTimestamp(pt),
				environment:   env,
			},
			AutoRenew:   isAutoRenew,
			ExpiresTime: parseMillisecondUnixTimestamp(exp),
		})
	}

	purchases, err := v.Storage.StoreSubscriptionPurchases(ctx, storagePurchases)
	if err != nil {
		return nil, err
	}

	if len(purchases) < 1 {
		return nil, ErrPurchaseReceiptAlreadySeen
	}

	validatedPurchases := make([]*ValidatedPurchase, 0, len(purchases))
	for _, p := range purchases {
		validatedPurchases = append(validatedPurchases, &ValidatedPurchase{
			ProductId:        p.productId,
			TransactionId:    p.transactionId,
			Store:            p.store,
			PurchaseTime:     p.purchaseTime.Unix(),
			CreateTime:       p.createTime.Unix(),
			UpdateTime:       p.updateTime.Unix(),
			ProviderResponse: string(raw),
			Environment:      p.environment,
		})
	}

	return &ValidatePurchaseResponse{
		ValidatedPurchases: validatedPurchases,
	}, nil
}

func parseMillisecondUnixTimestamp(t int) time.Time {
	return time.Unix(0, 0).Add(time.Duration(t) * time.Millisecond)
}
