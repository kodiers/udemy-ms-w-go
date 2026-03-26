package main

import (
	"context"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcHandler struct {
	pb.UnimplementedDriverServiceServer
	Service *Service
}

func NewGrpcHandler(s *grpc.Server, service *Service) {
	handler := &grpcHandler{Service: service}
	pb.RegisterDriverServiceServer(s, handler)
}

func (h *grpcHandler) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	driver, err := h.Service.RegisterDriver(req.DriverID, req.PackageSlug)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register driver: %v", err)
	}
	return &pb.RegisterDriverResponse{
		Driver: driver,
	}, nil
}

func (h *grpcHandler) UnregisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	h.Service.UnregisterDriver(req.DriverID)
	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id:             req.DriverID,
			Name:           "",
			ProfilePicture: "",
			CarPlate:       "",
			Geohash:        "",
			PackageSlug:    "",
			Location:       nil,
		},
	}, nil
}
