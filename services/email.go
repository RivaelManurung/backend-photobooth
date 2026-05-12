package services

import (
	"backendphotobooth/config"
	"backendphotobooth/models"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"strings"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{config: cfg}
}

// SendWelcomeEmail sends welcome email to new user
func (s *EmailService) SendWelcomeEmail(user *models.User) error {
	subject := "Welcome to Photo Booth!"
	body := fmt.Sprintf(`
		<h1>Welcome %s!</h1>
		<p>Thank you for registering at Photo Booth.</p>
		<p>You're currently on the <strong>%s</strong> plan.</p>
		<p>Start creating amazing photo strips now!</p>
		<br>
		<p>Best regards,<br>Photo Booth Team</p>
	`, user.Name, user.SubscriptionPlan)

	return s.sendEmail(user.Email, subject, body)
}

// SendVerificationEmail sends email verification link
func (s *EmailService) SendVerificationEmail(user *models.User, verificationURL string) error {
	subject := "Verify Your Email"
	body := fmt.Sprintf(`
		<h1>Email Verification</h1>
		<p>Hi %s,</p>
		<p>Please verify your email address by clicking the link below:</p>
		<p><a href="%s">Verify Email</a></p>
		<p>This link will expire in 24 hours.</p>
		<br>
		<p>If you didn't create this account, please ignore this email.</p>
	`, user.Name, verificationURL)

	return s.sendEmail(user.Email, subject, body)
}

// SendPasswordResetEmail sends password reset link
func (s *EmailService) SendPasswordResetEmail(user *models.User, resetURL string) error {
	subject := "Reset Your Password"
	body := fmt.Sprintf(`
		<h1>Password Reset Request</h1>
		<p>Hi %s,</p>
		<p>We received a request to reset your password. Click the link below to reset it:</p>
		<p><a href="%s">Reset Password</a></p>
		<p>This link will expire in 1 hour.</p>
		<br>
		<p>If you didn't request this, please ignore this email.</p>
	`, user.Name, resetURL)

	return s.sendEmail(user.Email, subject, body)
}

// SendOrderConfirmation sends order confirmation email
func (s *EmailService) SendOrderConfirmation(user *models.User, order *models.Order) error {
	subject := "Order Confirmation - " + order.OrderNumber
	body := fmt.Sprintf(`
		<h1>Order Confirmation</h1>
		<p>Hi %s,</p>
		<p>Thank you for your order!</p>
		<h3>Order Details:</h3>
		<ul>
			<li>Order Number: <strong>%s</strong></li>
			<li>Plan: <strong>%s</strong></li>
			<li>Amount: <strong>Rp %s</strong></li>
			<li>Status: <strong>%s</strong></li>
		</ul>
		<p>You can view your order details in your account dashboard.</p>
		<br>
		<p>Best regards,<br>Photo Booth Team</p>
	`, user.Name, order.OrderNumber, order.SubscriptionPlan,
		formatCurrency(order.TotalAmount), order.Status)

	return s.sendEmail(user.Email, subject, body)
}

// SendSubscriptionRenewalReminder sends renewal reminder
func (s *EmailService) SendSubscriptionRenewalReminder(user *models.User, daysLeft int) error {
	subject := "Subscription Renewal Reminder"
	body := fmt.Sprintf(`
		<h1>Subscription Renewal</h1>
		<p>Hi %s,</p>
		<p>Your <strong>%s</strong> subscription will expire in <strong>%d days</strong>.</p>
		<p>To continue enjoying premium features, please renew your subscription.</p>
		<p><a href="https://yourdomain.com/pricing">Renew Now</a></p>
		<br>
		<p>Best regards,<br>Photo Booth Team</p>
	`, user.Name, user.SubscriptionPlan, daysLeft)

	return s.sendEmail(user.Email, subject, body)
}

// SendPhotoReadyNotification sends notification when photo is ready
func (s *EmailService) SendPhotoReadyNotification(user *models.User, photo *models.Photo) error {
	subject := "Your Photo is Ready!"
	body := fmt.Sprintf(`
		<h1>Photo Ready!</h1>
		<p>Hi %s,</p>
		<p>Your photo has been processed and is ready to download!</p>
		<p><a href="https://yourdomain.com/photos/%d">View Photo</a></p>
		<br>
		<p>Best regards,<br>Photo Booth Team</p>
	`, user.Name, photo.ID)

	return s.sendEmail(user.Email, subject, body)
}

// sendEmail sends email using SMTP
func (s *EmailService) sendEmail(to, subject, body string) error {
	switch strings.ToLower(s.config.Email.Driver) {
	case "", "disabled":
		return nil
	case "log":
		log.Printf("[email:log] to=%s subject=%q body=%s", to, subject, body)
		return nil
	case "smtp":
	default:
		return fmt.Errorf("unsupported email driver %q", s.config.Email.Driver)
	}

	// Setup authentication
	auth := smtp.PlainAuth(
		"",
		s.config.Email.SMTPUser,
		s.config.Email.SMTPPassword,
		s.config.Email.SMTPHost,
	)

	// Compose message
	msg := []byte(fmt.Sprintf(
		"From: %s <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		s.config.Email.FromName,
		s.config.Email.FromEmail,
		to,
		subject,
		body,
	))

	// Send email
	addr := fmt.Sprintf("%s:%d", s.config.Email.SMTPHost, s.config.Email.SMTPPort)
	return smtp.SendMail(addr, auth, s.config.Email.FromEmail, []string{to}, msg)
}

// Helper function to format currency
func formatCurrency(amount float64) string {
	return fmt.Sprintf("%.0f", amount)
}

// SendBulkEmail sends email to multiple recipients
func (s *EmailService) SendBulkEmail(recipients []string, subject, body string) error {
	for _, recipient := range recipients {
		if err := s.sendEmail(recipient, subject, body); err != nil {
			// Log error but continue with other recipients
			fmt.Printf("Failed to send email to %s: %v\n", recipient, err)
		}
	}
	return nil
}

// Email templates
type EmailTemplate struct {
	Subject string
	Body    *template.Template
}

// LoadEmailTemplates loads email templates from files
func (s *EmailService) LoadEmailTemplates() map[string]*EmailTemplate {
	templates := make(map[string]*EmailTemplate)

	// Welcome email template
	welcomeTemplate := template.Must(template.New("welcome").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background: #4CAF50; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background: #f9f9f9; }
				.button { background: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; display: inline-block; margin: 10px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Welcome to Photo Booth!</h1>
				</div>
				<div class="content">
					<h2>Hi {{.Name}},</h2>
					<p>Thank you for joining Photo Booth! We're excited to have you.</p>
					<p>You're currently on the <strong>{{.Plan}}</strong> plan.</p>
					<a href="{{.DashboardURL}}" class="button">Go to Dashboard</a>
				</div>
			</div>
		</body>
		</html>
	`))

	templates["welcome"] = &EmailTemplate{
		Subject: "Welcome to Photo Booth!",
		Body:    welcomeTemplate,
	}

	return templates
}

// SendTemplatedEmail sends email using template
func (s *EmailService) SendTemplatedEmail(to, templateName string, data interface{}) error {
	templates := s.LoadEmailTemplates()
	tmpl, exists := templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	var body bytes.Buffer
	if err := tmpl.Body.Execute(&body, data); err != nil {
		return err
	}

	return s.sendEmail(to, tmpl.Subject, body.String())
}
