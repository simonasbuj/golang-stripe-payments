package payments

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/paymentintent"
	"github.com/stripe/stripe-go/v84/webhook"
)

var errUnknownWebhookEventType = errors.New("unhandled event type")

type PaymentProvider interface {
	CreateCheckoutSession(req PaymentRequest) (string, error)
	CreatePaymentIntent(req PaymentRequest) (string, error)
	HandlePaymentSuccess(payload []byte, sigHeader string) (*PaymentSuccessWebhookResponse, error)
}

type StripeProvider struct {
	webhookSecret string
}

func NewStripeProvider(secretKey, webhookSecret string) *StripeProvider {
	if secretKey == "" || webhookSecret == "" {
		panic("secretKey and webhookSecret required for StripePaymentProvider")
	}
	stripe.Key = secretKey

	return &StripeProvider{
		webhookSecret: webhookSecret,
	}
}

func (p *StripeProvider) CreateCheckoutSession(req PaymentRequest) (string, error) {
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String(req.SuccessUrl),
		CancelURL:          stripe.String(req.CancelUrl),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"order_id": req.OrderID,
				"store_id": req.StoreID,
			},
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(req.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("order-number"),
					},
					UnitAmount: stripe.Int64(req.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
	}

	s, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("creating checkout session: %w", err)
	}

	return s.ID, nil
}

func (p *StripeProvider) CreatePaymentIntent(req PaymentRequest) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: map[string]string{
			"order_id": req.OrderID,
			"store_id": req.StoreID,
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", fmt.Errorf("creating payment intent: %w", err)
	}

	return pi.ClientSecret, nil
}

func (p *StripeProvider) HandlePaymentSuccess(payload []byte, sigHeader string) (*PaymentSuccessWebhookResponse, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, p.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("veryfing stripe webhook signature: %w", err)
	}

	if event.Type != "payment_intent.succeeded" {
		return nil, fmt.Errorf("%w: %s", errUnknownWebhookEventType, event.Type)
	}

	var pi stripe.PaymentIntent

	err = json.Unmarshal(event.Data.Raw, &pi)
	if err != nil {
		return nil, fmt.Errorf("unmarhsaling payment_intent: %w", err)
	}

	return &PaymentSuccessWebhookResponse{
		ID: pi.ID,
		Amount: pi.Amount,
		Currency: string(pi.Currency),
		OrderID: pi.Metadata["order_id"],
	}, nil
}
