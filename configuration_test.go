package leto

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ConfigurationSuite struct{}

var _ = Suite(&ConfigurationSuite{})

func (s *ConfigurationSuite) TestHasADefaultConfiguration(c *C) {
	config := RecommendedTrackingConfiguration()
	c.Check(config.CheckAllFieldAreSet(), IsNil)
}

func (s *ConfigurationSuite) TestCanBeMerged(c *C) {
	from := RecommendedTrackingConfiguration()

	to := &TrackingConfiguration{}

	expected := RecommendedTrackingConfiguration()

	to.NewAntRenewPeriod = new(time.Duration)
	*to.NewAntRenewPeriod = 10 * time.Minute
	*expected.NewAntRenewPeriod = 10 * time.Minute

	to.Stream.Host = new(string)
	*to.Stream.Host = "google.com"
	*expected.Stream.Host = "google.com"

	to.Camera.FPS = new(float64)
	*to.Camera.FPS = 10.0
	*expected.Camera.FPS = 10.0

	to.Detection.Family = new(string)
	*to.Detection.Family = "36ARTag"
	*expected.Detection.Family = "36ARTag"

	to.Detection.Quad.MinBWDiff = new(int)
	*to.Detection.Quad.MinBWDiff = 120
	*expected.Detection.Quad.MinBWDiff = 120

	c.Assert(from.Merge(to), IsNil)
	c.Check(from, DeepEquals, expected)

}
