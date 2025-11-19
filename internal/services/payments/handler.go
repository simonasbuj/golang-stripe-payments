package payments

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"

	"github.com/stripe/stripe-go/v84/paymentintent"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/webhook"
)

type handler struct {
	stripeWebhookSecret string
}

func NewHandler(stripeSecretKey, stripeWebhookSecret string) *handler {
	stripe.Key = stripeSecretKey
	return &handler{
		stripeWebhookSecret: stripeWebhookSecret,
	}
}

func (h *handler) CreateCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("running CreateCheckoutSessionHandler")
	var body struct {
		Amount   int64  	`json:"amount"`
		Currency string 	`json:"currency"`
		SuccessUrl string 	`json:"successUrl"`
		CancelUrl string 	`json:"cancelUrl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String(body.SuccessUrl),
		CancelURL:          stripe.String(body.CancelUrl),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"order_id": "this-gon-be-custom-order-id-PI",
				"store_id": "custom-store-id-PI",
			},
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(body.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("order-number"),
					},
					UnitAmount: stripe.Int64(body.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
	}

	s, err := session.New(params)
	if err != nil {
		slog.Error("session.New error", "error", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id":"%s"}`, s.ID)
}

func (h *handler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("running WebhookHandler")

	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error reading request body", "error", err)
		http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
		return
	}

	endpointSecret := h.stripeWebhookSecret
	if endpointSecret == "" {
		slog.Error("stripeWebhookSecret not set")
		http.Error(w, "webhook secret not configured", http.StatusInternalServerError)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, endpointSecret)
	if err != nil {
		slog.Error("webhook signature verification failed", "error", err)
		http.Error(w, "signature verification failed", http.StatusBadRequest)
		return
	}

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			slog.Error("error parsing webhook JSON", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// TODO: fulfill the purchase, e.g. mark order paid in DB
		slog.Info("checkout session completed: session id=%s, payment_status=%s", sess.ID, sess.PaymentStatus)

	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			slog.Error("Error parsing payment_intent", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// TODO: mark order as paid (if using PaymentIntent)
		slog.Info("PaymentIntent succeeded: id=%s, amount=%d", pi.ID, pi.Amount)

	default:
		slog.Error("Unhandled event type", "event_type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) CreatePaymentIntentHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("running CreatePaymentIntentHandler")
	var body struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Amount <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	if body.Currency == "" {
		body.Currency = "eur"
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(body.Amount),
		Currency: stripe.String(body.Currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, "Failed to create payment intent", http.StatusInternalServerError)
		return
	}

	resp := struct {
		ClientSecret string `json:"clientSecret"`
	}{
		ClientSecret: pi.ClientSecret,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
