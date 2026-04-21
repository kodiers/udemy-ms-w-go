package domain

import (
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideFareModel struct {
	ID                primitive.ObjectID         `bson:"_id,omitempty"`
	UserID            string                     `bson:"user_id"`
	PackageSlug       string                     `bson:"package_slug"`
	TotalPriceInCents float64                    `bson:"total_price_in_cents"`
	Route             *tripTypes.OsrmApiResponse `bson:"route"`
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
