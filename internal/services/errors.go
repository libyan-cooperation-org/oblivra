package services

import (
	"errors"
	"fmt"
)

var (
	ErrNotReady         = errors.New("application not ready")
	ErrNotAuthenticated = errors.New("vault not unlocked")
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotFound         = errors.New("resource not found")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrConnectionFailed = errors.New("connection failed")
	ErrTimeout          = errors.New("operation timed out")
	ErrPermissionDenied = errors.New("permission denied")
)

type RecoveryAction string

const (
	ActionNone     RecoveryAction = "none"
	ActionRetry    RecoveryAction = "retry"
	ActionReboot   RecoveryAction = "reboot"
	ActionUnlock   RecoveryAction = "unlock"
	ActionRelogin  RecoveryAction = "relogin"
	ActionCheckLog RecoveryAction = "check_log"
)

type AppError struct {
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Detail   string         `json:"detail,omitempty"`
	Recovery RecoveryAction `json:"recovery,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Detail)
}

func NewAppError(code string, message string, detail string, recovery RecoveryAction) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		Detail:   detail,
		Recovery: recovery,
	}
}
