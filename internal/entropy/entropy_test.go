package entropy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LayerGraph(t *testing.T) {
	nlg := NewLayerGraph()
	nlg.SetNode(1, []int64{2, 3}, []string{"a"})
	nlg.SetNode(2, []int64{3, 4, 5}, []string{"a"})
	nlg.SetNode(3, []int64{4, 6}, []string{"a"})
	nlg.SetNode(4, []int64{1, 6}, []string{"b"})
	nlg.SetNode(5, []int64{4, 6}, []string{"b"})
	nlg.SetNode(6, []int64{1, 2, 4}, []string{"b"})
	entropy := nlg.ProgressMultiLayerStructuralEntropy()
	assert.InDelta(t, entropy.LittleStructuralEntropy, 1.504751568, 1e-9, "Expected %v to be equal to %v within delta %v", entropy.LittleStructuralEntropy, 1.504751568, 1e-9)
}

func Test_LayerGraphV2(t *testing.T) {
	nlg := NewLayerGraph()
	nlg.SetNode(1, []int64{2, 3}, []string{"a"})
	nlg.SetNode(2, []int64{3, 4, 5}, []string{"a"})
	nlg.SetNode(3, []int64{4, 6}, []string{"a", "b"})
	nlg.SetNode(4, []int64{1, 6}, []string{"b"})
	nlg.SetNode(5, []int64{4, 6}, []string{"b"})
	nlg.SetNode(6, []int64{1, 2, 4}, []string{"b"})
	entropy := nlg.ProgressMultiLayerStructuralEntropy()
	assert.InDelta(t, entropy.LittleStructuralEntropy, 1.612197223, 1e-9, "Expected %v to be equal to %v within delta %v", entropy.LittleStructuralEntropy, 1.612197223, 1e-9)
	assert.InDelta(t, entropy.BigDegreeEntropy, 2.636056086, 1e-9, "Expected %v to be equal to %v within delta %v", entropy.BigDegreeEntropy, 2.278585108, 1e-9)
}
