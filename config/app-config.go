// Package config holds the application's configuration settings.
package config

// AppConfig defines environment-based configuration for the application.
type AppConfig struct {
	Http   HttpConfig
	Stripe StripeConfig
	PayPal PaypalConfig
}

type HttpConfig struct {
	Addr	string `env:"PAYMENTS_HTTP_ADDR"`
}

type StripeConfig struct {
	SecretKey 		string `env:"STRIPE_SECRET_KEY"`
	WebhookSecret 	string `env:"STRIPE_WEBHOOK_SECRET"`
}

type PaypalConfig struct {
	ClientID 	string 	`env:"PAYPAL_CLIENT_ID"`
	SecretKey 	string 	`env:"PAYPAL_SECRET_KEY"`
}
