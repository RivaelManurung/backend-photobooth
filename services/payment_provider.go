package services

import (
	"backendphotobooth/config"
	"backendphotobooth/models"
	"context"
	"fmt"
)

type PaymentInstructions struct {
	Provider     string `json:"provider"`
	QRISImageURL string `json:"qris_image_url,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

type PaymentProvider interface {
	Name() string
	CreatePayment(ctx context.Context, order *models.Order) (*PaymentInstructions, error)
}

type ManualQRISProvider struct {
	imageURL     string
	instructions string
}

func NewManualQRISProvider(cfg *config.Config) *ManualQRISProvider {
	return &ManualQRISProvider{
		imageURL:     cfg.Payment.ManualQRISImageURL,
		instructions: cfg.Payment.ManualQRISInstructions,
	}
}

func (p *ManualQRISProvider) Name() string {
	return "manual_qris"
}

func (p *ManualQRISProvider) CreatePayment(ctx context.Context, order *models.Order) (*PaymentInstructions, error) {
	return &PaymentInstructions{
		Provider:     p.Name(),
		QRISImageURL: p.imageURL,
		Instructions: p.instructions,
	}, nil
}

func NewPaymentProvider(cfg *config.Config) (PaymentProvider, error) {
	switch cfg.Payment.Provider {
	case "", "manual_qris":
		return NewManualQRISProvider(cfg), nil
	case "gopay":
		return nil, fmt.Errorf("gopay provider uses existing GoPay QRIS handler")
	case "midtrans", "stripe":
		return nil, fmt.Errorf("%s provider is disabled unless configured explicitly", cfg.Payment.Provider)
	default:
		return nil, fmt.Errorf("unsupported payment provider %q", cfg.Payment.Provider)
	}
}
