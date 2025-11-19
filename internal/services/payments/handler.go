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

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String("https://example.com/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:          stripe.String("https://example.com/cancel"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("eur"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("order-number"),
					},
					UnitAmount: stripe.Int64(2000),
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
		http.Error(w, "Webhook secret not configured", http.StatusInternalServerError)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, endpointSecret)
	if err != nil {
		slog.Error("⚠️  Webhook signature verification failed", "error", err)
		http.Error(w, "Signature verification failed", http.StatusBadRequest)
		return
	}

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		// Deserialize the event data into the proper struct
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			slog.Error("Error parsing webhook JSON", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// TODO: fulfill the purchase, e.g. mark order paid in DB
		slog.Info("Checkout session completed: session id=%s, payment_status=%s", sess.ID, sess.PaymentStatus)

	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			slog.Error("Error parsing payment_intent", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// TODO: mark order as paid (if using PaymentIntent)
		slog.Info("PaymentIntent succeeded: id=%s, amount=%d", pi.ID, pi.Amount)

	// handle other relevant events:
	// case "invoice.payment_succeeded": for subscriptions
	default:
		slog.Error("Unhandled event type", "event_type", event.Type)
	}

	// Return 200 to acknowledge receipt
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
