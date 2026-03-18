package main

import (
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"
)

type previewTripRequest struct {
	UserId      string           `json:"userID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

func (p *previewTripRequest) ToProto() *pb.PreviewTripRequest {
	return &pb.PreviewTripRequest{
		UserID: p.UserId,
		StartLocation: &pb.Coordinate{
			Latitude:  p.Pickup.Latitude,
			Longitude: p.Pickup.Longitude,
		},
		EndLocation: &pb.Coordinate{
			Latitude:  p.Destination.Latitude,
			Longitude: p.Destination.Longitude,
		},
	}
}

type startTripRequest struct {
	RideFareId string `json:"rideFareId"`
	UserId     string `json:"userID"`
}

func (s *startTripRequest) ToProto() *pb.CreateTripRequest {
	return &pb.CreateTripRequest{
		RideFareId: s.RideFareId,
		UserID:     s.UserId,
	}
}
