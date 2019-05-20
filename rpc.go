package leto

import (
	"math"
	"time"
)

type Response struct {
	Error string
}

type TagDetectionConfiguration struct {
	Family              string
	QuadDecimate        float64
	QuadSigma           float64
	RefineEdges         bool
	QuadMinClusterPixel int
	QuadMaxNMaxima      int
	QuadCriticalRadian  float64
	QuadMaxLineMSE      float64
	QuadMinBWDiff       int
	QuadDeglitch        bool
}

func NewTagDetectionConfig() TagDetectionConfiguration {
	return TagDetectionConfiguration{
		Family:              "36h11",
		QuadDecimate:        1.0,
		QuadSigma:           0.0,
		RefineEdges:         false,
		QuadMinClusterPixel: 5,
		QuadMaxNMaxima:      10,
		QuadCriticalRadian:  10.0 * math.Pi / 180.0,
		QuadMaxLineMSE:      10.0,
		QuadMinBWDiff:       5.0,
		QuadDeglitch:        false,
	}
}

type CameraConfiguration struct {
	StrobeDuration time.Duration
	StrobeDelay    time.Duration
	FPS            float64
}

type TrackingStart struct {
	ExperimentName      string
	NewAntOutputROISize int
	StreamHost          string
	BitRateKB           int
	Camera              CameraConfiguration
	Tag                 TagDetectionConfiguration
}

type SlaveTrackingStart struct {
	Stride int
	IDs    []int
	Remote string
	UUID   string
}

type TrackingStop struct {
}

type Link struct {
	Hostname string
}

type Unlink struct {
	Hostname string
}
