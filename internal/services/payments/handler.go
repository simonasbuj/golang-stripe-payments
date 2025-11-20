package payments

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type handler struct {
	paymentProvider PaymentProvider
}

func NewHandler(paymentProvider PaymentProvider) *handler {
	return &handler{
		paymentProvider: paymentProvider,
	}
}

func (h *handler) CreateCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("running CreateCheckoutSessionHandler")

	var body PaymentRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	checkoutID, err := h.paymentProvider.CreateCheckoutSession(body)
	if err != nil {
		slog.Error("creating new checkout session", "error", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id":"%s"}`, checkoutID)
}

func (h *handler) PaymentSuccessWebhook(w http.ResponseWriter, r *http.Request) {
	slog.Info("running WebhookHandler")

	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error reading request body", "error", err)
		http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")

	resp, err := h.paymentProvider.HandlePaymentSuccess(payload, sigHeader)
	if err != nil {
		if errors.Is(err, errUnknownWebhookEventType) {
			slog.Error("unknown wehbook event type", "error", err)
			return
		}
	}

	slog.Info("successful payment handled", "payment", resp)

	w.WriteHeader(http.StatusOK)
}

func (h *handler) CreatePaymentIntentHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("running CreatePaymentIntentHandler")
	var body PaymentRequest

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

	
	clientSecret, err := h.paymentProvider.CreatePaymentIntent(body)
	if err != nil {
		slog.Error("failed to create payment intent", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to create payment intent: " + err.Error(),
		})
		return
	}

	resp := PaymentIntentResponse{clientSecret}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
