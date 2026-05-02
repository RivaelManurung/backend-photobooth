package services

import (
	"backendphotobooth/config"
	"backendphotobooth/models"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/skip2/go-qrcode"
)

type GoPayQRISService struct {
	config      *config.Config
	merchantID  string
	secretKey   string
	baseURL     string
	callbackURL string
}

// GoPayQRISRequest represents request to create QRIS
type GoPayQRISRequest struct {
	MerchantID    string  `json:"merchant_id"`
	TerminalID    string  `json:"terminal_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	OrderNumber   string  `json:"order_number"`
	CustomerName  string  `json:"customer_name"`
	CustomerPhone string  `json:"customer_phone"`
	CustomerEmail string  `json:"customer_email"`
	CallbackURL   string  `json:"callback_url"`
	ExpiryMinutes int     `json:"expiry_minutes"`
}

// GoPayQRISResponse represents response from GoPay
type GoPayQRISResponse struct {
	Success         bool   `json:"success"`
	TransactionID   string `json:"transaction_id"`
	QRISString      string `json:"qris_string"`
	QRISImageURL    string `json:"qris_image_url"`
	Amount          float64 `json:"amount"`
	ExpiresAt       string `json:"expires_at"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	ErrorCode       string `json:"error_code"`
}

// GoPayCallbackData represents callback from GoPay
type GoPayCallbackData struct {
	TransactionID   string  `json:"transaction_id"`
	OrderNumber     string  `json:"order_number"`
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"` // success, failed, expired
	PaymentMethod   string  `json:"payment_method"`
	PaidAt          string  `json:"paid_at"`
	CustomerName    string  `json:"customer_name"`
	Signature       string  `json:"signature"`
}

func NewGoPayQRISService(cfg *config.Config) *GoPayQRISService {
	return &GoPayQRISService{
		config:      cfg,
		merchantID:  cfg.Payment.GoPayMerchantID,
		secretKey:   cfg.Payment.GoPaySecretKey,
		baseURL:     cfg.Payment.GoPayBaseURL,
		callbackURL: cfg.Payment.GoPayCallbackURL,
	}
}

// CreateQRIS creates a new QRIS payment
func (s *GoPayQRISService) CreateQRIS(order *models.Order, user *models.User) (*models.QRISPayment, error) {
	// Prepare request
	request := GoPayQRISRequest{
		MerchantID:    s.merchantID,
		TerminalID:    "TERMINAL-001", // Your terminal ID
		Amount:        order.TotalAmount,
		Currency:      "IDR",
		OrderNumber:   order.OrderNumber,
		CustomerName:  user.Name,
		CustomerPhone: user.Phone,
		CustomerEmail: user.Email,
		CallbackURL:   s.callbackURL,
		ExpiryMinutes: 15, // QRIS expires in 15 minutes
	}

	// Generate signature
	signature := s.generateSignature(request)

	// Make API call to GoPay
	response, err := s.callGoPayAPI("/qris/create", request, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to call GoPay API: %w", err)
	}

	var goPayResponse GoPayQRISResponse
	if err := json.Unmarshal(response, &goPayResponse); err != nil {
		return nil, fmt.Errorf("failed to parse GoPay response: %w", err)
	}

	if !goPayResponse.Success {
		return nil, fmt.Errorf("GoPay error: %s (code: %s)", goPayResponse.Message, goPayResponse.ErrorCode)
	}

	// Generate QR code image
	qrImageURL, err := s.generateQRCodeImage(goPayResponse.QRISString, order.OrderNumber)
	if err != nil {
		// Log error but continue - we still have the QRIS string
		fmt.Printf("Failed to generate QR image: %v\n", err)
	}

	// Parse expiry time
	expiresAt, _ := time.Parse(time.RFC3339, goPayResponse.ExpiresAt)
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(15 * time.Minute)
	}

	// Create QRIS payment record
	qrisPayment := &models.QRISPayment{
		OrderID:            order.ID,
		QRISString:         goPayResponse.QRISString,
		QRISImageURL:       qrImageURL,
		GoPayTransactionID: goPayResponse.TransactionID,
		GoPayMerchantID:    s.merchantID,
		GoPayTerminalID:    request.TerminalID,
		Amount:             order.TotalAmount,
		Currency:           "IDR",
		Status:             "pending",
		ExpiresAt:          expiresAt,
		CustomerName:       user.Name,
		CustomerPhone:      user.Phone,
		CustomerEmail:      user.Email,
		CallbackURL:        s.callbackURL,
		RawResponse:        string(response),
	}

	return qrisPayment, nil
}

// CheckPaymentStatus checks payment status from GoPay
func (s *GoPayQRISService) CheckPaymentStatus(transactionID string) (string, error) {
	request := map[string]string{
		"merchant_id":    s.merchantID,
		"transaction_id": transactionID,
	}

	signature := s.generateSignatureFromMap(request)

	response, err := s.callGoPayAPI("/qris/status", request, signature)
	if err != nil {
		return "", err
	}

	var statusResponse struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(response, &statusResponse); err != nil {
		return "", err
	}

	if !statusResponse.Success {
		return "", fmt.Errorf("failed to check status: %s", statusResponse.Message)
	}

	return statusResponse.Status, nil
}

// VerifyCallback verifies callback signature from GoPay
func (s *GoPayQRISService) VerifyCallback(callback *GoPayCallbackData) bool {
	// Generate expected signature
	data := fmt.Sprintf("%s|%s|%.2f|%s",
		callback.TransactionID,
		callback.OrderNumber,
		callback.Amount,
		callback.Status,
	)

	expectedSignature := s.generateHMAC(data)
	return expectedSignature == callback.Signature
}

// CancelQRIS cancels a QRIS payment
func (s *GoPayQRISService) CancelQRIS(transactionID string) error {
	request := map[string]string{
		"merchant_id":    s.merchantID,
		"transaction_id": transactionID,
	}

	signature := s.generateSignatureFromMap(request)

	response, err := s.callGoPayAPI("/qris/cancel", request, signature)
	if err != nil {
		return err
	}

	var cancelResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(response, &cancelResponse); err != nil {
		return err
	}

	if !cancelResponse.Success {
		return fmt.Errorf("failed to cancel: %s", cancelResponse.Message)
	}

	return nil
}

// callGoPayAPI makes HTTP request to GoPay API
func (s *GoPayQRISService) callGoPayAPI(endpoint string, payload interface{}, signature string) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := s.baseURL + endpoint
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Merchant-ID", s.merchantID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GoPay API error: %s", string(body))
	}

	return body, nil
}

// generateSignature generates signature for request
func (s *GoPayQRISService) generateSignature(request GoPayQRISRequest) string {
	data := fmt.Sprintf("%s|%s|%.2f|%s",
		request.MerchantID,
		request.OrderNumber,
		request.Amount,
		request.Currency,
	)
	return s.generateHMAC(data)
}

// generateSignatureFromMap generates signature from map
func (s *GoPayQRISService) generateSignatureFromMap(data map[string]string) string {
	str := ""
	for k, v := range data {
		str += k + "=" + v + "&"
	}
	return s.generateHMAC(str)
}

// generateHMAC generates HMAC-SHA256 signature
func (s *GoPayQRISService) generateHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(s.secretKey))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// generateQRCodeImage generates QR code image from QRIS string
func (s *GoPayQRISService) generateQRCodeImage(qrisString, orderNumber string) (string, error) {
	// Generate QR code
	qr, err := qrcode.New(qrisString, qrcode.Medium)
	if err != nil {
		return "", err
	}

	// Save to file
	filename := fmt.Sprintf("qris-%s.png", orderNumber)
	filepath := fmt.Sprintf("./uploads/qris/%s", filename)

	if err := qr.WriteFile(256, filepath); err != nil {
		return "", err
	}

	return fmt.Sprintf("/uploads/qris/%s", filename), nil
}

// SimulatePayment simulates payment for testing (DEVELOPMENT ONLY)
func (s *GoPayQRISService) SimulatePayment(transactionID string) (*GoPayCallbackData, error) {
	// This is for testing only - DO NOT use in production
	return &GoPayCallbackData{
		TransactionID: transactionID,
		Status:        "success",
		PaidAt:        time.Now().Format(time.RFC3339),
	}, nil
}
