package main

import (
	"fmt"
	"golang-stripe-payments/config"
	"golang-stripe-payments/internal/services/payments/handler"
	"golang-stripe-payments/internal/services/payments/providers"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

func main() {
	var cfg config.AppConfig

	err := cleanenv.ReadEnv(&cfg)
		if err != nil {
		log.Panic("failed to load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	stripeProvider := providers.NewStripeProvider(cfg.Stripe.SecretKey, cfg.Stripe.WebhookSecret)
	paypalProvider := providers.NewPayPalProvider(cfg.PayPal.ClientID, cfg.PayPal.SecretKey)
	handler := handler.NewHandler(stripeProvider, paypalProvider)

	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	http.HandleFunc("/create-checkout-session", handler.CreateCheckoutSession)
	http.HandleFunc("/create-payment-intent", handler.CreatePaymentIntent)
	http.HandleFunc("/webhook/stripe/payment-success", handler.PaymentSuccessWebhook)

	http.HandleFunc("/create-paypal-order", handler.CreatePayPalOrder)
	
	slog.Info(fmt.Sprintf("Server running on %s", cfg.Http.Addr))

	err = http.ListenAndServe(cfg.Http.Addr, nil)
	if err != nil {
		slog.Error("failed to serve server", "error", err)
	}
}
