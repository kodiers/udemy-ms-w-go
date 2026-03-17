package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"

	"google.golang.org/grpc"
)

type GrpcHandler struct {
	pb.UnimplementedTripServiceServer
	service domain.TripService
}

func NewGrpcHandler(server *grpc.Server, service domain.TripService) *GrpcHandler {
	handler := &GrpcHandler{
		service: service,
	}
	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *GrpcHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.StartLocation
	destination := req.EndLocation
	t, err := h.service.GetRoute(ctx, pickup, destination)
	if err != nil {
		log.Println(err)
	}
	return &pb.PreviewTripResponse{
		Route:
	}, nil
}
