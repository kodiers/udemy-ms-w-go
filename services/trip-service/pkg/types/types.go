package types

import pb "ride-sharing/shared/proto/trip"

type OsrmApiResponse struct {
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
}

func (o *OsrmApiResponse) ToProto() *pb.Route {
	route := o.Routes[0]
	geometry := route.Geometry.Coordinates
	coordinates := make([]*pb.Coordinate, len(geometry))
	for i, c := range geometry {
		coordinates[i] = &pb.Coordinate{Latitude: c[0], Longitude: c[1]}
	}
	return &pb.Route{
		Distance: route.Distance,
		Duration: route.Duration,
		Geometry: []*pb.Geometry{
			{
				Coordinates: coordinates,
			},
		},
	}
}

type PricingConfig struct {
	PricePerKm     float64
	PricePerMinute float64
}

func DefaultPricingConfig() *PricingConfig {
	return &PricingConfig{
		PricePerKm:     1.5,
		PricePerMinute: 0.25,
	}
}
