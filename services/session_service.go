package services

import (
	"backendphotobooth/models"
	"fmt"
)

type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

func (s *SessionService) CanTransition(current, next string) bool {
	switch current {
	case models.StatusDraft:
		return next == models.StatusCapturing || next == models.StatusCancelled
	case models.StatusCapturing:
		return next == models.StatusProcessing || next == models.StatusCancelled
	case models.StatusProcessing:
		return next == models.StatusWaitingPayment || next == models.StatusPaid || next == models.StatusCancelled
	case models.StatusWaitingPayment:
		return next == models.StatusPaid || next == models.StatusCancelled || next == models.StatusExpired
	case models.StatusPaid:
		return next == models.StatusCompleted
	case models.StatusCompleted, models.StatusExpired, models.StatusCancelled:
		return false // Terminal states
	default:
		return false
	}
}

func (s *SessionService) ValidateStateAction(session *models.Session, action string) error {
	if session.Status == models.StatusExpired || session.IsExpired() {
		return fmt.Errorf("session has expired")
	}

	switch action {
	case "upload":
		if session.Status != models.StatusCapturing {
			return fmt.Errorf("cannot upload photos in %s status", session.Status)
		}
	case "pay":
		if session.Status != models.StatusWaitingPayment {
			return fmt.Errorf("session is not waiting for payment")
		}
	case "download":
		if session.Status != models.StatusPaid && session.Status != models.StatusCompleted {
			return fmt.Errorf("payment required to download")
		}
	}
	return nil
}
