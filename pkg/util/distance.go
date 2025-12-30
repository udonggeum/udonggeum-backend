package util

import (
	"math"
)

// CalculateDistance calculates the distance between two geographic points using the Haversine formula
// Parameters: lat1, lon1, lat2, lon2 in degrees
// Returns: distance in kilometers
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := degToRad(lat1)
	lon1Rad := degToRad(lon1)
	lat2Rad := degToRad(lat2)
	lon2Rad := degToRad(lon2)

	// Differences
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusKm * c

	return distance
}

// degToRad converts degrees to radians
func degToRad(deg float64) float64 {
	return deg * (math.Pi / 180.0)
}
