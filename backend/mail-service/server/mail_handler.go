package server

import (
	"bytes"
	"context"
	"fmt"
	mailpb "gen/mail-service"
	userpb "gen/user-service-gen"
	"html/template"
	"log"
	"net/smtp"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const verificationEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<style>
		body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 0; }
		.container { max-width: 600px; margin: 20px auto; background: #ffffff; padding: 20px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
		h2 { color: #333333; }
		p { color: #555555; line-height: 1.6; }
		.button { display: inline-block; background-color: #4CAF50; color: white !important; padding: 12px 24px; text-decoration: none; border-radius: 4px; font-weight: bold; margin-top: 15px; }
		.footer { margin-top: 30px; font-size: 12px; color: #999999; border-top: 1px solid #eeeeee; padding-top: 10px; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Welcome to FileSify, {{.Username}}!</h2>
		<p>Thank you for registering. Please click the button below to verify your account and complete your registration:</p>
		<p>
			<a href="{{.VerificationUrl}}" class="button">Verify My Account</a>
		</p>
		<p>If the button doesn't work, copy and paste this link into your browser:</p>
		<p style="word-break: break-all;"><a href="{{.VerificationUrl}}">{{.VerificationUrl}}</a></p>
		<div class="footer">
			This verification link is secure and unique to your registration request.
		</div>
	</div>
</body>
</html>
`

const twoFAEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<style>
		body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 0; }
		.container { max-width: 600px; margin: 20px auto; background: #ffffff; padding: 20px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
		h2 { color: #333333; }
		p { color: #555555; line-height: 1.6; }
		.button { display: inline-block; background-color: #4CAF50; color: white !important; padding: 12px 24px; text-decoration: none; border-radius: 4px; font-weight: bold; margin-top: 15px; }
		.footer { margin-top: 30px; font-size: 12px; color: #999999; border-top: 1px solid #eeeeee; padding-top: 10px; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Welcome back to FileSify, {{.FirstName}}!</h2>
		<p>Please enter the code below to complete login:</p>
		<p>
			<b>{{.Code}}</b>
		</p>
		<div class="footer">
			This code is unique to your login request.
		</div>
	</div>
</body>
</html>
`

const magicLinkEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<style>
		body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 0; }
		.container { max-width: 600px; margin: 20px auto; background: #ffffff; padding: 20px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
		h2 { color: #333333; }
		p { color: #555555; line-height: 1.6; }
		.button { display: inline-block; background-color: #4f46e5; color: white !important; padding: 12px 24px; text-decoration: none; border-radius: 4px; font-weight: bold; margin-top: 15px; }
		.footer { margin-top: 30px; font-size: 12px; color: #999999; border-top: 1px solid #eeeeee; padding-top: 10px; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Hi {{.Username}},</h2>
		<p>Click the button below to sign in to your FileSify account. This link is valid for <strong>12 hours</strong> and can only be used once.</p>
		<p>
			<a href="{{.MagicLinkUrl}}" class="button">Sign In to FileSify</a>
		</p>
		<p>If the button doesn't work, copy and paste this link into your browser:</p>
		<p style="word-break: break-all;"><a href="{{.MagicLinkUrl}}">{{.MagicLinkUrl}}</a></p>
		<div class="footer">
			If you didn't request this email, you can safely ignore it — your account is not at risk.
		</div>
	</div>
</body>
</html>
`

type MailServer struct {
	mailpb.UnimplementedMailServiceServer
	userClient userpb.UserServiceClient
}

func NewMailServer(userClient userpb.UserServiceClient) *MailServer {
	return &MailServer{userClient: userClient}
}

type EmailPayload struct {
	To      []string
	Subject string
	Body    bytes.Buffer
}

type VerificationTemplateData struct {
	Username        string
	VerificationUrl string
}

type MagicLinkTemplateData struct {
	Username     string
	MagicLinkUrl string
}

type TwoFactorTemplateData struct {
	FirstName string
	Code      string
}

func SendMail(ctx context.Context, payload EmailPayload) error {
	smtpAddress := os.Getenv("SMTP_ADDRESS")
	if smtpAddress == "" {
		smtpAddress = "mailhog:1025"
	}

	err := smtp.SendMail(smtpAddress, nil, "noreply@filesify.com", payload.To, payload.Body.Bytes())
	if err != nil {
		log.Printf("Failed to route mail to MailHog: %v", err)
		return status.Errorf(codes.Unavailable, "failed to send email via mail relay: %v", err)
	}
	return nil
}

func (s *MailServer) SendVerification(ctx context.Context, req *userpb.SendVerificationRequest) (*userpb.SendVerificationResponse, error) {
	if req.GetEmail() == "" || req.GetVerificationUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and verification url are required fields")
	}

	tmpl, err := template.New("verification").Parse(verificationEmailTemplate)
	if err != nil {
		log.Printf("[MailService] Failed to parse internal template: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to parse template: %v", err)
	}

	data := VerificationTemplateData{
		Username:        req.GetUsername(),
		VerificationUrl: req.GetVerificationUrl(),
	}

	var bodyBuffer bytes.Buffer
	if err := tmpl.Execute(&bodyBuffer, data); err != nil {
		log.Printf("[MailService] Failed to execute template payload: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to inject template data: %v", err)
	}

	email := &EmailPayload{
		Subject: "Verify Mail",
	}

	from := "noreply@filesify.com"
	to := []string{req.GetEmail()}

	var rawMessage bytes.Buffer
	fmt.Fprintf(&rawMessage, "From: %s\r\n", from)
	fmt.Fprintf(&rawMessage, "To: %s\r\n", to)
	fmt.Fprintf(&rawMessage, "Subject: %s\r\n", email.Subject)
	rawMessage.WriteString("MIME-Version: 1.0\r\n")
	rawMessage.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	rawMessage.WriteString("\r\n")
	rawMessage.Write(bodyBuffer.Bytes())

	email.Body = rawMessage
	email.To = to
	if err := SendMail(ctx, *email); err != nil {
		return nil, err
	}

	return &userpb.SendVerificationResponse{
		Message: "Verification email sent successfully natively.",
	}, nil
}

func (s *MailServer) SendMagicLink(ctx context.Context, req *userpb.SendMagicLinkRequest) (*userpb.SendMagicLinkResponse, error) {
	if req.GetEmail() == "" || req.GetMagicLinkUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and magic link url are required fields")
	}

	tmpl, err := template.New("magic-link").Parse(magicLinkEmailTemplate)
	if err != nil {
		log.Printf("[MailService] Failed to parse magic-link template: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to parse template: %v", err)
	}

	data := MagicLinkTemplateData{
		Username:     req.GetUsername(),
		MagicLinkUrl: req.GetMagicLinkUrl(),
	}

	var bodyBuffer bytes.Buffer
	if err := tmpl.Execute(&bodyBuffer, data); err != nil {
		log.Printf("[MailService] Failed to execute magic-link template: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to inject template data: %v", err)
	}

	smtpAddress := os.Getenv("SMTP_ADDRESS")
	if smtpAddress == "" {
		smtpAddress = "mailhog:1025"
	}

	from := "noreply@filesify.com"
	to := req.GetEmail()

	var rawMessage bytes.Buffer
	fmt.Fprintf(&rawMessage, "From: %s\r\n", from)
	fmt.Fprintf(&rawMessage, "To: %s\r\n", to)
	fmt.Fprintf(&rawMessage, "Subject: %s\r\n", "Sign in to FileSify")
	rawMessage.WriteString("MIME-Version: 1.0\r\n")
	rawMessage.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	rawMessage.WriteString("\r\n")
	rawMessage.Write(bodyBuffer.Bytes())

	if err := smtp.SendMail(smtpAddress, nil, from, []string{to}, rawMessage.Bytes()); err != nil {
		log.Printf("[MailService] Failed to send magic-link email to %s: %v", to, err)
		return nil, status.Errorf(codes.Unavailable, "failed to send email via mail relay: %v", err)
	}

	log.Printf("[MailService] Magic link sent to %s", to)

	return &userpb.SendMagicLinkResponse{
		Message: "Magic link email sent successfully.",
	}, nil
}

func (s *MailServer) SendTwoFactor(ctx context.Context, req *userpb.SendTwoFactorRequest) (*userpb.SendTwoFactorResponse, error) {
	if req.GetCode() == "" || req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "[mail-service/send-2FA] Email and OTP are required fields")
	}

	tmpl, err := template.New("two-factor").Parse(twoFAEmailTemplate)
	if err != nil {
		log.Printf("[MailService] Failed to parse two-factor template: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to parse template: %v", err)
	}

	data := TwoFactorTemplateData{
		FirstName: req.Name,
		Code:      req.GetCode(),
	}

	var bodyBuffer bytes.Buffer
	if err := tmpl.Execute(&bodyBuffer, data); err != nil {
		return nil, status.Errorf(codes.Internal, "[mail-service] Failed to create email body")
	}

	email := EmailPayload{
		Subject: "Two Factor Authentication",
	}

	to := []string{req.GetEmail()}
	from := "noreply@filesify.com"

	var rawMessage bytes.Buffer
	fmt.Fprintf(&rawMessage, "From: %s\r\n", from)
	fmt.Fprintf(&rawMessage, "To: %s\r\n", to)
	fmt.Fprintf(&rawMessage, "Subject: %s\r\n", email.Subject)
	rawMessage.WriteString("MIME-Version: 1.0\r\n")
	rawMessage.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	rawMessage.WriteString("\r\n")
	rawMessage.Write(bodyBuffer.Bytes())

	email.Body = rawMessage
	email.To = to
	if err := SendMail(ctx, email); err != nil {
		return nil, err
	}

	return &userpb.SendTwoFactorResponse{
		Message: "OTP sent to mail",
	}, nil
}
