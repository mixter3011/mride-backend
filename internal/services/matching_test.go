package services

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHaversineDistance_MumbaiPune(t *testing.T) {
	// Mumbai: 19.0760, 72.8777  Pune: 18.5204, 73.8567
	// Haversine (great-circle) distance ≈ 120 km (road distance is ~150 km)
	distance := haversineDistance(19.0760, 72.8777, 18.5204, 73.8567)
	assert.InDelta(t, 120.0, distance, 5.0, "Mumbai-Pune great-circle distance should be ~120 km, got %.2f", distance)
}

func TestHaversineDistance_SamePoint(t *testing.T) {
	distance := haversineDistance(19.0760, 72.8777, 19.0760, 72.8777)
	assert.InDelta(t, 0.0, distance, 0.001, "Same point should return ~0 km, got %.6f", distance)
}

func TestHaversineDistance_Antipodal(t *testing.T) {
	// North pole to south pole ≈ 20015 km (half earth circumference)
	distance := haversineDistance(90, 0, -90, 0)
	assert.InDelta(t, 20015.0, distance, 100.0, "Pole-to-pole should be ~20015 km, got %.2f", distance)
}

func TestHaversineDistance_DelhiMumbai(t *testing.T) {
	// Delhi: 28.7041, 77.1025  Mumbai: 19.0760, 72.8777
	// Expected: ~1148 km
	distance := haversineDistance(28.7041, 77.1025, 19.0760, 72.8777)
	assert.InDelta(t, 1148.0, distance, 20.0, "Delhi-Mumbai should be ~1148 km, got %.2f", distance)
}

func TestPointToSegmentDistance_OnSegment(t *testing.T) {
	// Point is the midpoint of the segment => distance should be ~0.
	ax, ay := 19.0, 72.0
	bx, by := 19.0, 73.0
	// Midpoint
	px, py := 19.0, 72.5

	dist := pointToSegmentDistance(px, py, ax, ay, bx, by)
	assert.InDelta(t, 0.0, dist, 0.5, "Point on segment should have ~0 distance, got %.4f", dist)
}

func TestPointToSegmentDistance_FarOff(t *testing.T) {
	// Segment runs east-west at lat 19.0 from lng 72.0 to 73.0.
	// Point is 0.1 degrees north (~11 km).
	ax, ay := 19.0, 72.0
	bx, by := 19.0, 73.0
	px, py := 19.1, 72.5

	dist := pointToSegmentDistance(px, py, ax, ay, bx, by)
	// 0.1 degree latitude ≈ 11.1 km
	assert.InDelta(t, 11.1, dist, 1.5, "Point 0.1° north of E-W segment should be ~11 km, got %.2f", dist)
}

func TestPointToSegmentDistance_BeyondEndpoint(t *testing.T) {
	// Segment from A(19.0, 72.0) to B(19.0, 73.0).
	// Point is at (19.0, 74.0) — 1 degree east beyond B.
	// Should measure to B (nearest endpoint), not to infinite line extension.
	ax, ay := 19.0, 72.0
	bx, by := 19.0, 73.0
	px, py := 19.0, 74.0

	dist := pointToSegmentDistance(px, py, ax, ay, bx, by)
	// 1 degree longitude at lat 19 ≈ 111.32 * cos(19°) ≈ 105 km
	expectedDist := 111.32 * math.Cos(19.0*math.Pi/180)
	assert.InDelta(t, expectedDist, dist, 5.0, "Point beyond endpoint should measure to nearest endpoint, got %.2f", dist)
}

func TestPointToSegmentDistance_BeforeStartpoint(t *testing.T) {
	// Segment from A(19.0, 72.0) to B(19.0, 73.0).
	// Point is at (19.0, 71.0) — 1 degree west before A.
	ax, ay := 19.0, 72.0
	bx, by := 19.0, 73.0
	px, py := 19.0, 71.0

	dist := pointToSegmentDistance(px, py, ax, ay, bx, by)
	expectedDist := 111.32 * math.Cos(19.0*math.Pi/180)
	assert.InDelta(t, expectedDist, dist, 5.0, "Point before start should measure to start endpoint, got %.2f", dist)
}

func TestCalculateMatchScore_CloserBetter(t *testing.T) {
	// Requester: from (19.08, 72.88) to (18.52, 73.86)
	reqFromLat, reqFromLng := 19.08, 72.88
	reqToLat, reqToLng := 18.52, 73.86

	// Candidate A: very close to requester's from/to
	rideAFromLat, rideAFromLng := 19.076, 72.877
	rideAToLat, rideAToLng := 18.520, 73.856

	// Candidate B: farther away from requester
	rideBFromLat, rideBFromLng := 19.15, 72.95
	rideBToLat, rideBToLng := 18.60, 73.95

	scoreA := CalculateMatchScore(reqFromLat, reqFromLng, reqToLat, reqToLng,
		rideAFromLat, rideAFromLng, rideAToLat, rideAToLng)
	scoreB := CalculateMatchScore(reqFromLat, reqFromLng, reqToLat, reqToLng,
		rideBFromLat, rideBFromLng, rideBToLat, rideBToLng)

	assert.Less(t, scoreA.Score, scoreB.Score,
		"Closer candidate A (score %.4f) should have lower score than farther candidate B (score %.4f)",
		scoreA.Score, scoreB.Score)
	assert.Less(t, scoreA.PickupDistance, scoreB.PickupDistance)
	assert.Less(t, scoreA.DropoffDistance, scoreB.DropoffDistance)
}

func TestCalculateMatchScore_ScoreComponents(t *testing.T) {
	ms := CalculateMatchScore(19.08, 72.88, 18.52, 73.86,
		19.08, 72.88, 18.52, 73.86)

	// Identical from/to → pickup and dropoff should be ~0, score ~0
	assert.InDelta(t, 0.0, ms.PickupDistance, 0.01)
	assert.InDelta(t, 0.0, ms.DropoffDistance, 0.01)
	assert.InDelta(t, 0.0, ms.Score, 0.1)
}
