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

type MailServer struct {
	mailpb.UnimplementedMailServiceServer
	userClient userpb.UserServiceClient
}

func NewMailServer(userClient userpb.UserServiceClient) *MailServer {
	return &MailServer{userClient: userClient}
}

func (s *MailServer) VerifyAccount(ctx context.Context, req *userpb.VerifyAccountRequest) (*userpb.VerifyAccountResponse, error) {
	if req.GetEmail() == "" || req.GetVerificationUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and verification url are required fields")
	}

	tmpl, err := template.New("verification").Parse(verificationEmailTemplate)
	if err != nil {
		log.Printf("[MailService] Failed to parse internal template: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to parse template: %v", err)
	}

	var bodyBuffer bytes.Buffer
	if err := tmpl.Execute(&bodyBuffer, req); err != nil {
		log.Printf("[MailService] Failed to execute template payload: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to inject template data: %v", err)
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	fromAddr := os.Getenv("SMTP_FROM")
	toAddr := req.GetEmail()

	var rawMessage bytes.Buffer
	rawMessage.WriteString(fmt.Sprintf("From: %s\r\n", fromAddr))
	rawMessage.WriteString(fmt.Sprintf("To: %s\r\n", toAddr))
	rawMessage.WriteString("Subject: Verify Your FileSify Account\r\n")
	rawMessage.WriteString("MIME-Version: 1.0\r\n")
	rawMessage.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	rawMessage.WriteString("\r\n")
	rawMessage.Write(bodyBuffer.Bytes())

	var auth smtp.Auth
	if smtpUser != "" && smtpPass != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	}

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	err = smtp.SendMail(addr, auth, fromAddr, []string{toAddr}, rawMessage.Bytes())
	if err != nil {
		log.Printf("[MailService] Native SMTP delivery breakdown to %s: %v", toAddr, err)
		return nil, status.Errorf(codes.Internal, "failed to deliver email package: %v", err)
	}

	log.Printf("[MailService] Verification successfully delivered natively to %s", toAddr)

	return &userpb.VerifyAccountResponse{
		Message: "Verification email sent successfully natively.",
	}, nil
}
