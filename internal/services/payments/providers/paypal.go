package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang-stripe-payments/internal/services/payments/types"
	"io"
	"net/http"
	"strconv"
)

type PayPalProvider struct {
	clientID 	string
	secretKey 	string
}

func NewPayPalProvider(clientID, secretKey string) *PayPalProvider {
	return &PayPalProvider{
		clientID: clientID,
		secretKey: secretKey,
	}
}

func (p *PayPalProvider) CreateCheckoutSession(req types.PaymentRequest) (string, error) {
    // 1. Get access token
    tokenReq, _ := http.NewRequest("POST", "https://api-m.sandbox.paypal.com/v1/oauth2/token", bytes.NewBufferString("grant_type=client_credentials"))
    tokenReq.SetBasicAuth(p.clientID, p.secretKey)
    tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    tokenResp, err := client.Do(tokenReq)
	fmt.Println(tokenResp)
    if err != nil {
        return "", err
    }
    defer tokenResp.Body.Close()

    var tokenData struct {
        AccessToken string `json:"access_token"`
    }
    json.NewDecoder(tokenResp.Body).Decode(&tokenData)
	fmt.Printf("token: %+v\n", tokenData)

    // 2. Create order
    orderBody := map[string]interface{}{
        "intent": "CAPTURE",
		"payment_source": map[string]interface{}{
			"paypal": map[string]interface{}{
				"experience_context": map[string]interface{}{
					"payment_method_preference": "IMMEDIATE_PAYMENT_REQUIRED",
					"landing_page":              "LOGIN",
					"shipping_preference":       "GET_FROM_FILE",
					"user_action":               "PAY_NOW",
					"return_url":                req.SuccessUrl,
					"cancel_url":                req.CancelUrl,
				},
			},
		},
        "purchase_units": []map[string]interface{}{
            {
                "amount": map[string]string{
                    "currency_code": req.Currency,
                    "value":         strconv.FormatInt(req.Amount, 10),
                },
            },
        },
    }

    bodyJson, _ := json.Marshal(orderBody)
	fmt.Printf("bodyJson: %+v\n", bodyJson)

    call, _ := http.NewRequest("POST", "https://api-m.sandbox.paypal.com/v2/checkout/orders", bytes.NewBuffer(bodyJson))
    call.Header.Set("Content-Type", "application/json")
    call.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

    resp, err := client.Do(call)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Print the raw response
	fmt.Println("Response body:", string(bodyBytes))
    var orderResp struct {
        ID    string `json:"id"`
        Links []struct {
            Href   string `json:"href"`
            Rel    string `json:"rel"`
            Method string `json:"method"`
        } `json:"links"`
    }
    json.NewDecoder(resp.Body).Decode(&orderResp)

    // Find approval link
    for _, link := range orderResp.Links {
        if link.Rel == "approve" || link.Rel == "payer-action" {
            return link.Href, nil
        }
    }

    return "", nil
}

func (p *PayPalProvider) CreatePaymentIntent(req types.PaymentRequest) (string, error){
	return "", fmt.Errorf("unimplemented method CreatePaymentIntent in paypal provider")
}

func (p *PayPalProvider) HandlePaymentSuccess(payload []byte, sigHeader string) (*types.PaymentSuccessWebhookResponse, error) {
	return nil, fmt.Errorf("unimplemented method HandlePaymentSuccess in paypal provider")
}
