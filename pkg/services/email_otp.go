package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"os"
	"sync"
	"time"
)

// OTPEntry stores OTP data with expiry and attempt tracking
type OTPEntry struct {
	Code      string
	ExpiresAt time.Time
	Attempts  int
}

var (
	otpStore = make(map[string]*OTPEntry)
	otpMutex sync.RWMutex
)

// GenerateOTP creates a 6-digit OTP and stores it for the given email
func GenerateOTP(email string) (string, error) {
	// Generate 6-digit numeric OTP
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %v", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())

	// Store OTP with 5-minute expiry
	otpMutex.Lock()
	otpStore[email] = &OTPEntry{
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Attempts:  0,
	}
	otpMutex.Unlock()

	return code, nil
}

// VerifyOTP checks the OTP for the given email
func VerifyOTP(email, code string) (bool, string) {
	otpMutex.Lock()
	defer otpMutex.Unlock()

	entry, exists := otpStore[email]
	if !exists {
		return false, "No OTP found. Please request a new one."
	}

	// Check expiry
	if time.Now().After(entry.ExpiresAt) {
		delete(otpStore, email)
		return false, "OTP has expired. Please request a new one."
	}

	// Check attempts
	entry.Attempts++
	if entry.Attempts > 3 {
		delete(otpStore, email)
		return false, "Too many attempts. Please request a new OTP."
	}

	// Compare
	if entry.Code != code {
		return false, fmt.Sprintf("Invalid OTP. %d attempts remaining.", 3-entry.Attempts)
	}

	// Success - remove OTP
	delete(otpStore, email)
	return true, ""
}

// SendOTPEmail sends the OTP to the user's email via SMTP
func SendOTPEmail(email, code string) error {
	smtpEmail := os.Getenv("SMTP_EMAIL")
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	if smtpEmail == "" || smtpPassword == "" {
		// Log the OTP to console for development/testing
		fmt.Printf("📧 [DEV MODE] OTP for %s: %s\n", email, code)
		return nil
	}

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	subject := "Your Quick Byte Login OTP"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px; background: #1a1a2e; color: #ffffff;">
			<div style="max-width: 400px; margin: 0 auto; background: #16213e; border-radius: 16px; padding: 32px; text-align: center;">
				<h2 style="color: #FF6B35; margin-bottom: 8px;">Quick Byte</h2>
				<p style="color: #a0a0b0; margin-bottom: 24px;">Your one-time login code</p>
				<div style="background: #0f3460; border-radius: 12px; padding: 20px; margin-bottom: 24px;">
					<span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #FF6B35;">%s</span>
				</div>
				<p style="color: #a0a0b0; font-size: 14px;">This code expires in 5 minutes.</p>
				<p style="color: #a0a0b0; font-size: 12px; margin-top: 16px;">If you didn't request this, please ignore this email.</p>
			</div>
		</body>
		</html>
	`, code)

	message := fmt.Sprintf("From: Quick Byte <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		smtpEmail, email, subject, body)

	auth := smtp.PlainAuth("", smtpEmail, smtpPassword, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpEmail, []string{email}, []byte(message))
	if err != nil {
		// Fallback: log OTP to console
		fmt.Printf("📧 [EMAIL FAILED] OTP for %s: %s (error: %v)\n", email, code, err)
		return nil // Don't fail the request, just log
	}

	fmt.Printf("📧 OTP sent to %s\n", email)
	return nil
}

// CleanupExpiredOTPs removes expired OTP entries (call periodically)
func CleanupExpiredOTPs() {
	otpMutex.Lock()
	defer otpMutex.Unlock()

	now := time.Now()
	for email, entry := range otpStore {
		if now.After(entry.ExpiresAt) {
			delete(otpStore, email)
		}
	}
}
