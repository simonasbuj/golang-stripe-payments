package providers

import (
	"errors"
	"golang-stripe-payments/internal/services/payments/types"
)

var ErrUnknownWebhookEventType = errors.New("unhandled event type")

type PaymentProvider interface {
	CreateCheckoutSession(req types.PaymentRequest) (string, error)
	CreatePaymentIntent(req types.PaymentRequest) (string, error)
	HandlePaymentSuccess(payload []byte, sigHeader string) (*types.PaymentSuccessWebhookResponse, error)
}
