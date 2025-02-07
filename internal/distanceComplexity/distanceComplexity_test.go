package distanceComplexity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DistanceGraph(t *testing.T) {
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
	assert.InDelta(t, entropy.LittlComplexity, 2.028320834, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.LittlComplexity, 2.028320834, 1e-9)
	assert.InDelta(t, entropy.BigComplexity, 1.4999999999999998, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.BigComplexity, 1.4999999999999998, 1e-9)
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
	assert.InDelta(t, entropy.LittlComplexity, 2.310637912413301, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.LittlComplexity, 2.310637912413301, 1e-9)
	assert.InDelta(t, entropy.BigComplexity, 2.2438900508, 1e-9,
		"Expected %v to be equal to %v within delta %v",
		entropy.BigComplexity, 2.2438900508, 1e-9)
}
