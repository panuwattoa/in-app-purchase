# in-app-purchase

[![License](https://img.shields.io/badge/license-WTFPL-brightgreen)](https://github.com/panuwattoa/in-app-purchase/blob/master/LICENSE)
#### in-app purchase library verify (in-app billing) for Apple, Google Play validation easy.

You should always validate receipts on the server, in [Apple's words](https://developer.apple.com/documentation/storekit/original_api_for_in-app_purchase/validating_receipts_with_the_app_store):
> Do not call the App Store server verifyReceipt endpoint from your app. You can't build a trusted connection between a user’s device and the App Store directly, because you don’t control either end of that connection, which makes it susceptible to a machine-in-the-middle attack.


## Installation

```go
go get -u github.com/panuwattoa/in-app-purchase
```

## Usage

##### Basic Usage:

```
you can use function in `iap` package or see example on `validation` package
```


##### Sample Apple JSON Response Schema not all return field
```json

{
  "status": 0,
  "environment": "Sandbox",
  "receipt": {
    "receipt_type": "ProductionSandbox",
    "adam_id": 0,
    "app_item_id": 0,
    "bundle_id": "your_product_id",
    "application_version": "58",
    "download_id": 0,
    "version_external_identifier": 0,
    "receipt_creation_date": "2016-06-17 01:54:26 Etc/GMT",
    "receipt_creation_date_ms": "1466128466000",
    "receipt_creation_date_pst": "2016-06-16 18:54:26 America/Los_Angeles",
    "request_date": "2016-06-17 17:34:41 Etc/GMT",
    "request_date_ms": "1466184881174",
    "request_date_pst": "2016-06-17 10:34:41 America/Los_Angeles",
    "original_purchase_date": "2013-08-01 07:00:00 Etc/GMT",
    "original_purchase_date_ms": "1375340400000",
    "original_purchase_date_pst": "2013-08-01 00:00:00 America/Los_Angeles",
    "original_application_version": "1.0",
    "in_app": [
      {
        "quantity": "1",
        "product_id": "product_id",
        "transaction_id": "1000000218147651",
        "original_transaction_id": "1000000218147500",
        "purchase_date": "2016-06-17 01:32:28 Etc/GMT",
        "purchase_date_ms": "1466127148000",
        "purchase_date_pst": "2016-06-16 18:32:28 America/Los_Angeles",
        "original_purchase_date": "2016-06-17 01:30:33 Etc/GMT",
        "original_purchase_date_ms": "1466127033000",
        "original_purchase_date_pst": "2016-06-16 18:30:33 America/Los_Angeles",
        "expires_date": "2016-06-17 01:37:28 Etc/GMT",
        "expires_date_ms": "1466127448000",
        "expires_date_pst": "2016-06-16 18:37:28 America/Los_Angeles",
        "web_order_line_item_id": "1000000032727764",
        "is_trial_period": "false"
      }
    ]
  },
}

```

##### Sample Google JSON Response Schema not all return field
```json
{
    "acknowledgementState": 0,
    "consumptionState": 0,
    "developerPayload": "",
    "kind": "androidpublisher#productPurchase",
    "orderId": "GPA.3332-9537-2114-58804",
    "purchaseState": 0,
    "purchaseTimeMillis": "1627715546799",
    "purchaseType": 0,
    "regionCode": "TH"
}
```

##### Sample Google Subscription JSON Response Schema not all return field
```json
{
    "startTimeMillis": "1623061338874",
    "expiryTimeMillis": "1623061756809",
    "autoRenewing": true,
    "priceCurrencyCode": "THB",
    "priceAmountMicros": "119000000",
    "countryCode": "TH",
    "developerPayload": "{\"developerPayload\":\"\",\"is_free_trial\":false,\"has_introductory_price_trial\":false,\"is_updated\":false,\"accountId\":\"\"}",
    "paymentState": 1,
    "orderId": "GPA.3329-6601-9591-xxxx",
    "purchaseType": 0,
    "acknowledgementState": 1,
    "kind": "androidpublisher#subscriptionPurchase"
}
```