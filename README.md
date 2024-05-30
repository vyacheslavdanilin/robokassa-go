# robokassa-go

## Robokassa Payment Gateway

This package provides integration with the Robokassa payment system for Go applications.

## Installation

To install, use `go get`:

```sh
go get github.com/vyacheslavdanilin/robokassa-go
```

## Usage

### Initializing a Payment

```go
package main

import (
	"context"
	"fmt"
	"github.com/vyacheslavdanilin/robokassa-go"
)

func main() {
	// Initialize the payment
	payment := robokassa.NewPayment("your_login", "your_payment_password", "your_validation_password", true)

	// Set payment parameters
	payment.SetSum(100.50)
	payment.SetDescription("Test payment")
	payment.SetInvoiceId(12345)

	// Get the payment URL
	url, err := payment.GetPaymentUrl(context.Background(), "base")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Payment URL:", url)
	}
}
```