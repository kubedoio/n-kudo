package controlplane

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"
)

// EmailService handles sending emails
type EmailService struct {
	enabled  bool
	host     string
	port     int
	user     string
	password string
	from     string
	baseURL  string
}

// NewEmailService creates a new email service
func NewEmailService(cfg Config) *EmailService {
	return &EmailService{
		enabled:  cfg.SMTPHost != "",
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		user:     cfg.SMTPUser,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
		baseURL:  cfg.AppBaseURL,
	}
}

// IsEnabled returns whether email is configured
func (s *EmailService) IsEnabled() bool {
	return s.enabled
}

// SendVerificationEmail sends an email verification link
func (s *EmailService) SendVerificationEmail(to, token string) error {
	if !s.enabled {
		return nil // Silently skip if email not configured
	}

	subject := "Verify your email address - N-Kudo"
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)

	body := fmt.Sprintf(`
Hello,

Thank you for registering with N-Kudo. Please verify your email address by clicking the link below:

%s

This link will expire in 24 hours.

If you didn't create an account, you can safely ignore this email.

Best regards,
The N-Kudo Team
`, verifyURL)

	return s.sendEmail(to, subject, body)
}

// SendPasswordResetEmail sends a password reset link
func (s *EmailService) SendPasswordResetEmail(to, token string) error {
	if !s.enabled {
		return nil // Silently skip if email not configured
	}

	subject := "Reset your password - N-Kudo"
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)

	body := fmt.Sprintf(`
Hello,

You requested to reset your password. Click the link below to set a new password:

%s

This link will expire in 1 hour.

If you didn't request this, you can safely ignore this email. Your password will not be changed.

Best regards,
The N-Kudo Team
`, resetURL)

	return s.sendEmail(to, subject, body)
}

// SendInvitationEmail sends a project invitation email
func (s *EmailService) SendInvitationEmail(to, inviterName, projectName, token string) error {
	if !s.enabled {
		return nil // Silently skip if email not configured
	}

	subject := fmt.Sprintf("You've been invited to join %s on N-Kudo", projectName)
	inviteURL := fmt.Sprintf("%s/accept-invitation?token=%s", s.baseURL, token)

	body := fmt.Sprintf(`
Hello,

%s has invited you to join the project "%s" on N-Kudo.

Click the link below to accept the invitation:

%s

This link will expire in 7 days.

Best regards,
The N-Kudo Team
`, inviterName, projectName, inviteURL)

	return s.sendEmail(to, subject, body)
}

// sendEmail sends a plain text email
func (s *EmailService) sendEmail(to, subject, body string) error {
	if !s.enabled {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body))

	var auth smtp.Auth
	if s.user != "" && s.password != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
}

// Email template cache
var emailTemplates = map[string]*template.Template{}

// GetEmailTemplate returns a cached email template
func GetEmailTemplate(name string) (*template.Template, error) {
	if tmpl, ok := emailTemplates[name]; ok {
		return tmpl, nil
	}

	var tmplStr string
	switch name {
	case "verification":
		tmplStr = verificationEmailTemplate
	case "password-reset":
		tmplStr = passwordResetEmailTemplate
	case "invitation":
		tmplStr = invitationEmailTemplate
	default:
		return nil, fmt.Errorf("unknown template: %s", name)
	}

	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return nil, err
	}

	emailTemplates[name] = tmpl
	return tmpl, nil
}

// Email template strings
const verificationEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify your email</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">Welcome to N-Kudo!</h2>
        <p>Thank you for registering. Please verify your email address by clicking the button below:</p>
        <a href="{{.URL}}" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0;">Verify Email Address</a>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.URL}}</p>
        <p>This link will expire in 24 hours.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">If you didn't create an account, you can safely ignore this email.</p>
    </div>
</body>
</html>
`

const passwordResetEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Reset your password</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">Password Reset Request</h2>
        <p>You requested to reset your password. Click the button below to set a new password:</p>
        <a href="{{.URL}}" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0;">Reset Password</a>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.URL}}</p>
        <p>This link will expire in 1 hour.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">If you didn't request this, you can safely ignore this email.</p>
    </div>
</body>
</html>
`

const invitationEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Project Invitation</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">You've Been Invited!</h2>
        <p><strong>{{.InviterName}}</strong> has invited you to join the project "<strong>{{.ProjectName}}</strong>" on N-Kudo.</p>
        <a href="{{.URL}}" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0;">Accept Invitation</a>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.URL}}</p>
        <p>This invitation will expire in 7 days.</p>
    </div>
</body>
</html>
`

// RenderEmailTemplate renders an email template with data
func RenderEmailTemplate(tmplName string, data interface{}) (string, error) {
	tmpl, err := GetEmailTemplate(tmplName)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
