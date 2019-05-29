package leto

import (
	"errors"
	"math"
	"time"
)

type Response struct {
	Error string
}

type Status struct {
	Running        bool
	Since          time.Time
	ExperimentName string
}

func (r Response) ToError() error {
	if len(r.Error) == 0 {
		return nil
	}
	return errors.New(r.Error)
}

type TagDetectionConfiguration struct {
	Family              string  `long:"at-family" description:"tag family to use" default:"36h11"`
	QuadDecimate        float64 `long:"at-quad-decimate" description:"Decimate quads" default:"1.0"`
	QuadSigma           float64 `long:"at-quad-sigma" description:"Blur before finding quads" default:"0.0"`
	RefineEdges         bool    `long:"at-refine-edges" description:"Refine Edges once a quad is found"`
	QuadMinClusterPixel int     `long:"at-quad-min-cluster" description:"Minimum numbe rof pixel to consider a quad" default:"25"`
	QuadMaxNMaxima      int     `long:"at-quad-max-n-maxima" description:"Maximum number of corner to consider when fitting a quad" default:"10"`
	QuadCriticalRadian  float64 `long:"at-quad-critical-radian" description:"Minimal angle for a quad corner" default:"0.17453292519"`
	QuadMaxLineMSE      float64 `long:"at-quad-max-line-mse" description:"Maximal MSE for a line fit" default:"10.0"`
	QuadMinBWDiff       int     `long:"at-quad-min-bw-diff" description:"Maximal MSE for a line fit" default:"25"`
	QuadDeglitch        bool    `long:"at-quad-deglitch" description:"Quad deglitching"`
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
	StrobeDelay    time.Duration `long:"strobe-delay" description:"delay of the strobe signal" default:"0us"`
	StrobeDuration time.Duration `long:"strobe-duration" description:"duration of the strobe signal" default:"1500us"`
	FPS            float64       `long:"f" description:"FPS to use for the experiment" default:"8.0"`
}

type TrackingStart struct {
	ExperimentName      string `short:"e" long:"experiment" description:"Name of the experiment to run" required:"true"`
	NewAntOutputROISize int    `long:"new-ant-size" description:"Size of the image when a new ant is found" default:"600"`
	StreamHost          string `long:"stream-host" description:"host to stream to"`
	BitRateKB           int    `long:"cbr" description:"Constant encoding bitrate to use in kb/s" default:"2000"`
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
