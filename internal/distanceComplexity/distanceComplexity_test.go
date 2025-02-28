package distanceComplexity

import (
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func Test_DistanceGraphV1(t *testing.T) {
	nlg := NewDistanceGraph()
	nlg.SetEdge(1, 2, float64(1))
	nlg.SetEdge(1, 3, float64(2))
	nlg.SetEdge(1, 4, float64(3))
	nlg.SetEdge(2, 3, float64(1))
	nlg.SetEdge(2, 4, float64(2))
	nlg.SetEdge(3, 4, float64(3))
	nlg.SetNodeCategory(1, []string{"a"})
	nlg.SetNodeCategory(2, []string{"a"})
	nlg.SetNodeCategory(3, []string{"b"})
	nlg.SetNodeCategory(4, []string{"b"})
	entropy := nlg.ProgressDistanceComplexity()
	assert.InDelta(t, entropy.LittlComplexity, 1.861654167, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.LittlComplexity, 1.861654167, 1e-9)
	assert.InDelta(t, entropy.BigComplexity, 1.4999999999999998, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.BigComplexity, 1.4999999999999998, 1e-9)
}

func Test_DistanceGraphV3(t *testing.T) {
	nlg := NewDistanceGraph()
	nlg.SetEdge(1, 2, float64(1))
	nlg.SetEdge(1, 3, float64(2))
	nlg.SetEdge(1, 4, float64(3))
	nlg.SetEdge(2, 3, float64(1))
	nlg.SetEdge(2, 4, float64(2))
	nlg.SetEdge(3, 4, float64(3))
	nlg.SetEdge(4, 5, float64(4))
	nlg.SetNodeCategory(1, []string{"a"})
	nlg.SetNodeCategory(2, []string{"a"})
	nlg.SetNodeCategory(3, []string{"b"})
	nlg.SetNodeCategory(4, []string{"b"})
	nlg.SetNodeCategory(5, []string{"b"})
	entropy := nlg.ProgressDistanceComplexity()
	assert.InDelta(t, entropy.LittlComplexity, 1.513679924, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.LittlComplexity, 1.513679924, 1e-9)
	assert.InDelta(t, entropy.BigComplexity, 1.4693609377704333, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.BigComplexity, 1.46936093777043338, 1e-9)
}

func Test_DistanceGraphV2(t *testing.T) {
	nlg := NewDistanceGraph()
	nlg.SetEdge(1, 2, float64(1))
	nlg.SetEdge(1, 3, float64(2))
	nlg.SetEdge(1, 4, float64(3))
	nlg.SetEdge(1, 5, float64(2))
	nlg.SetEdge(1, 6, float64(2))
	nlg.SetEdge(2, 3, float64(1))
	nlg.SetEdge(2, 4, float64(2))
	nlg.SetEdge(2, 5, float64(1))
	nlg.SetEdge(2, 6, float64(1))
	nlg.SetEdge(3, 4, float64(1))
	nlg.SetEdge(3, 5, float64(2))
	nlg.SetEdge(3, 6, float64(2))
	nlg.SetEdge(4, 5, float64(3))
	nlg.SetEdge(4, 6, float64(3))
	nlg.SetEdge(5, 6, float64(2))
	nlg.SetNodeCategory(1, []string{"a"})
	nlg.SetNodeCategory(2, []string{"a"})
	nlg.SetNodeCategory(3, []string{"b"})
	nlg.SetNodeCategory(4, []string{"b"})
	nlg.SetNodeCategory(5, []string{"b"})
	nlg.SetNodeCategory(6, []string{"b"})
	entropy := nlg.ProgressDistanceComplexity()
	assert.InDelta(t, entropy.LittlComplexity, 2.184720099868397, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.LittlComplexity, 2.184720099868397, 1e-9)
	assert.InDelta(t, entropy.BigComplexity, 2.2438900508, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.BigComplexity, 2.2438900508, 1e-9)
}
