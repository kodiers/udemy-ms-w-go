package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	"ride-sharing/shared/types"

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

func (s *TripService) GetRoute(ctx context.Context, pickup types.Coordinate, destination types.Coordinate) (*tripTypes.OsrmApiResponse, error) {
	url := fmt.Sprintf("http://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		pickup.Longitude, pickup.Latitude, destination.Longitude, destination.Latitude)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch route from osrm api: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	var routeResponse tripTypes.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}
	return &routeResponse, nil
}

func (s *TripService) EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*domain.RideFareModel {
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))
	for i, fare := range baseFares {
		estimatedFares[i] = estimateFareRoute(fare, route)
	}
	return estimatedFares
}

func (s *TripService) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string) ([]*domain.RideFareModel, error) {
	fares := make([]*domain.RideFareModel, len(rideFares))
	for i, f := range rideFares {
		fare := &domain.RideFareModel{
			PackageSlug:       f.PackageSlug,
			TotalPriceInCents: f.TotalPriceInCents,
			UserID:            userID,
			ID:                primitive.NewObjectID(),
		}
		if err := s.repository.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save ride fare: %v", err)
		}
		fares[i] = fare
	}
	return fares, nil
}

func estimateFareRoute(fare *domain.RideFareModel, route *tripTypes.OsrmApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := fare.TotalPriceInCents
	distanceKm := route.Routes[0].Distance
	durationMinutes := route.Routes[0].Duration
	distanceFare := distanceKm * pricingCfg.PricePerKm
	timeFare := durationMinutes * pricingCfg.PricePerMinute
	totalPrice := carPackagePrice + distanceFare + timeFare
	return &domain.RideFareModel{
		PackageSlug:       fare.PackageSlug,
		TotalPriceInCents: totalPrice,
	}
}

func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 200,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 350,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 400,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 1000,
		},
	}
}
