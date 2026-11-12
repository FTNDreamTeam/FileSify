package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TwoFactorService struct {
	rdb        *redis.Client
	hmacSecret []byte
}

func (s *TwoFactorService) GenerateOTP(ctx context.Context, userId string) (string, error) {
	coolDownKey := fmt.Sprintf("otp:cooldown:%s", userId)
	ok, err := s.rdb.SetNX(ctx, coolDownKey, "1", 60*time.Second).Result()
	if err != nil {
		return "", status.Errorf(codes.Internal, "[user-service/two-factor] Failed to set Redis cooldown: %s", err)
	}
	if !ok {
		return "", status.Errorf(codes.PermissionDenied, "[user-service/two-factor] Please wait 60 seconds before requesting a new code %s", err)
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", status.Errorf(codes.Internal, "[user-service/two-factor] Failed random: %s", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())

	hashedCode := s.hashOTP(ctx, userId, code)

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, fmt.Sprintf("otp:code:%s", userId), hashedCode, 5*time.Minute)
	pipe.Del(ctx, fmt.Sprintf("otp:attempts:%s", userId))
	if _, err := pipe.Exec(ctx); err != nil {
		return "", status.Error(codes.Internal, "[user-service/two-factor] Failed saving otp")
	}

	return code, nil
}

func (s *TwoFactorService) hashOTP(ctx context.Context, userId string, code string) string {
	mac := hmac.New(sha256.New, s.hmacSecret)
	mac.Write([]byte(userId + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *TwoFactorService) VerifyOTP(ctx context.Context, userId string, submittedCode string) (bool, error) {
	codeKey := fmt.Sprintf("otp:code:%s", userId)
	attemptsKey := fmt.Sprintf("otp:attempts:%s", userId)

	attempts, err := s.rdb.Incr(ctx, attemptsKey).Result()
	if err != nil {
		return false, status.Errorf(codes.Internal, "[user-service/two-factor] Error loading failed attempts:%s", err)
	}
	if attempts == 1 {
		s.rdb.Expire(ctx, attemptsKey, 5*time.Minute)
	}
	if attempts > 5 {
		s.rdb.Del(ctx, codeKey) // Invalidate OTP immediately on brute-force
		return false, status.Errorf(codes.PermissionDenied, "[user-service/two-factor] Too many failed attempts, request a new codes:%s", err)
	}

	storedHash, err := s.rdb.Get(ctx, codeKey).Result()
	if err == redis.Nil {
		return false, status.Errorf(codes.Internal, "[user-service/two-factor] Code expired or not found:%s", err)
	}
	if err != nil {
		return false, err
	}

	submittedHash := s.hashOTP(ctx, userId, submittedCode)
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(submittedHash)) != 1 {
		return false, status.Errorf(codes.PermissionDenied, "[user-service/two-factor] Codes do not match:%s", err)
	}

	s.rdb.Del(ctx, codeKey, attemptsKey)
	return true, nil
}
