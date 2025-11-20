package types

type PaymentRequest struct {
	Amount   	int64  	`json:"amount"`
	Currency 	string 	`json:"currency"`
	SuccessUrl 	string 	`json:"successUrl"`
	CancelUrl 	string 	`json:"cancelUrl"`
	OrderID 	string  `json:"orderId"`
	StoreID 	string  `json:"storeId"`
}

type CheckoutSessionResponse struct {
	ID 	string 	`json:"id"`
}

type PaymentIntentResponse struct {
	ClientSecret 	string 	`json:"clientSecret"`
}

type PaymentSuccessWebhookResponse struct {
	ID 			string
	Amount 		int64
	Currency 	string
	OrderID 	string
}
