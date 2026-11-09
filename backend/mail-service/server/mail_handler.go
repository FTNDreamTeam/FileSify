package server

import (
	"bytes"
	"context"
	mailpb "gen/mail-service"
	userpb "gen/user-service-gen"
	"html/template"
	"log"

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
}

func NewMailServer() *MailServer {
	return &MailServer{}
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
}
