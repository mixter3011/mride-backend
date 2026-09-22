package services

import "math"

// MatchScore represents how well a ride/subscription matches a requester's search.
type MatchScore struct {
	Score          float64 // lower is better
	PickupDistance float64 // km
	DropoffDistance float64 // km
	RouteDeviation float64 // km — perpendicular distance of requester's pickup from driver's route line
}

// haversineDistance computes the great-circle distance in km between two lat/lng
// points using the Haversine formula.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// CalculateMatchScore scores a candidate ride against a requester's from/to coordinates.
// Weights: pickup proximity matters most, then dropoff proximity, then route deviation.
func CalculateMatchScore(reqFromLat, reqFromLng, reqToLat, reqToLng float64,
	rideFromLat, rideFromLng, rideToLat, rideToLng float64) MatchScore {

	pickupDist := haversineDistance(reqFromLat, reqFromLng, rideFromLat, rideFromLng)
	dropoffDist := haversineDistance(reqToLat, reqToLng, rideToLat, rideToLng)

	// Route deviation: perpendicular distance from the requester's pickup point
	// to the line segment between the driver's from/to points.
	routeDeviation := pointToSegmentDistance(reqFromLat, reqFromLng,
		rideFromLat, rideFromLng, rideToLat, rideToLng)

	weightPickup := 0.5
	weightDropoff := 0.3
	weightDeviation := 0.2

	score := weightPickup*pickupDist + weightDropoff*dropoffDist + weightDeviation*routeDeviation

	return MatchScore{
		Score:          score,
		PickupDistance: pickupDist,
		DropoffDistance: dropoffDist,
		RouteDeviation: routeDeviation,
	}
}

// pointToSegmentDistance computes the distance in km from a point (px, py) to
// the line segment defined by (ax, ay)→(bx, by), where coordinates are
// latitude/longitude.
//
// It uses an equirectangular projection (multiply lng delta by cos(avg lat) to
// correct for lat/lng not being equal-distance) to project into a locally-flat
// plane, computes the perpendicular distance there, then converts back to km
// using ~111.32 km per degree of latitude.
func pointToSegmentDistance(px, py, ax, ay, bx, by float64) float64 {
	// Convert to a locally-flat coordinate system (units ≈ degrees scaled to km).
	avgLat := (px + ax + bx) / 3.0
	cosLat := math.Cos(avgLat * math.Pi / 180)

	// Scale all coordinates: x = lng * cosLat, y = lat (both in degrees)
	// then multiply by 111.32 at the end to get km.
	pxFlat := py * cosLat // lng of point
	pyFlat := px          // lat of point
	axFlat := ay * cosLat // lng of segment start
	ayFlat := ax          // lat of segment start
	bxFlat := by * cosLat // lng of segment end
	byFlat := bx          // lat of segment end

	dx := bxFlat - axFlat
	dy := byFlat - ayFlat

	// If segment is a single point, return distance to that point.
	lenSq := dx*dx + dy*dy
	if lenSq == 0 {
		ddx := pxFlat - axFlat
		ddy := pyFlat - ayFlat
		return math.Sqrt(ddx*ddx+ddy*ddy) * 111.32
	}

	// Parameter t of the projection of P onto line AB, clamped to [0,1].
	t := ((pxFlat-axFlat)*dx + (pyFlat-ayFlat)*dy) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	// Closest point on the segment.
	closestX := axFlat + t*dx
	closestY := ayFlat + t*dy

	ddx := pxFlat - closestX
	ddy := pyFlat - closestY

	return math.Sqrt(ddx*ddx+ddy*ddy) * 111.32
}
