package service

import (
	"context"
	"ride-sharing/services/trip-service/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripService struct {
	repository domain.TripRepository
}

func NewTripService(repo domain.TripRepository) *TripService {
	return &TripService{
		repository: repo,
	}
}

func (s *TripService) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
	}
	return s.repository.CreateTrip(ctx, t)
}
