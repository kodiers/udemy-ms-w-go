package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	pb.UnimplementedTripServiceServer
	service   domain.TripService
	publisher *events.TripEventPublisher
}

func NewGrpcHandler(server *grpc.Server, service domain.TripService, publisher *events.TripEventPublisher) *GrpcHandler {
	handler := &GrpcHandler{
		service:   service,
		publisher: publisher,
	}
	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *GrpcHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.StartLocation
	destination := req.EndLocation
	pickupCors := types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}
	destinationCors := types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}
	t, err := h.service.GetRoute(ctx, pickupCors, destinationCors)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}

	estimatedFares := h.service.EstimatePackagesPriceWithRoute(t)
	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, req.UserID, t)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate trip fares: %v", err)
	}
	return &pb.PreviewTripResponse{
		Route:     t.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}

func (h *GrpcHandler) CreateTrip(ctx context.Context, request *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fareId := request.RideFareId
	fare, err := h.service.GetAndValidateFare(ctx, fareId, request.UserID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ride fare: %v", err)
	}
	trip, err := h.service.CreateTrip(ctx, fare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create trip: %v", err)
	}
	err = h.publisher.PublishTripCreatedEvent(ctx, trip)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish trip created event: %v", err)
	}
	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}
