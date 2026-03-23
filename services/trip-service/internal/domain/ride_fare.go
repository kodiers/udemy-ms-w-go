package domain

import (
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideFareModel struct {
	ID                primitive.ObjectID
	UserID            string
	PackageSlug       string
	TotalPriceInCents float64
	Route             *tripTypes.OsrmApiResponse
}

func (r *RideFareModel) ToProto() *pb.RideFare {
	return &pb.RideFare{
		PackageSlug:       r.PackageSlug,
		TotalPriceInCents: r.TotalPriceInCents,
		Id:                r.ID.Hex(),
		UserID:            r.UserID,
	}
}

func ToRideFaresProto(fare []*RideFareModel) []*pb.RideFare {
	var protoFares []*pb.RideFare
	for _, f := range fare {
		protoFares = append(protoFares, f.ToProto())
	}
	return protoFares
}
