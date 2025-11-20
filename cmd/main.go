package main

import (
	"fmt"
	"golang-stripe-payments/config"
	"golang-stripe-payments/internal/services/payments"
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

	stripePaymentProvider := payments.NewStripeProvider(cfg.Stripe.SecretKey, cfg.Stripe.WebhookSecret)
	handler := payments.NewHandler(stripePaymentProvider)

	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	http.HandleFunc("/create-checkout-session", handler.CreateCheckoutSessionHandler)
	http.HandleFunc("/create-payment-intent", handler.CreatePaymentIntentHandler)
	http.HandleFunc("/webhook/stripe/payment-success", handler.PaymentSuccessWebhook)

	slog.Info(fmt.Sprintf("Server running on %s", cfg.Http.Addr))

	err = http.ListenAndServe(cfg.Http.Addr, nil)
	if err != nil {
		slog.Error("failed to serve server", "error", err)
	}
}
