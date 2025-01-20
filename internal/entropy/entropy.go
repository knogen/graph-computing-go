package entropy

import (
	"math"
	"sync"

	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/ider-zh/graph-entropy-go/graph"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/atomic"
)

type node struct {
	ID        int64
	OutDegree []int64
	InDegree  []int64
	Categroy  []string
}

type layerGraph struct {
	NodesMap    map[int64]*node
	maxID       int64
	copyNodeMap map[int64][]int64 // 保存复制点的记录, 生成入度的时候生效
}

func NewLayerGraph() *layerGraph {
	return &layerGraph{
		NodesMap:    make(map[int64]*node),
		copyNodeMap: make(map[int64][]int64),
	}
}

// 允许OutDegree 包含范围外的点, clean的时候会进行过滤处理
func (c *layerGraph) SetNode(ID int64, OutDegree []int64, Categroy []string) {
	c.NodesMap[ID] = &node{
		ID:        ID,
		OutDegree: OutDegree,
		Categroy:  Categroy,
	}
	if ID > c.maxID {
		c.maxID = ID
	}
}

// 网络收缩, 对交叉学科进行复制, 并去除交叉性
func (c *layerGraph) cleanGraph() {

	// 复制交叉学科
	for _, item := range c.NodesMap {
		if len(item.Categroy) > 1 {
			for i := 1; i < len(item.Categroy); i += 1 {
				// 复制
				c.maxID += 1
				newNodeID := c.maxID
				c.copyNodeMap[item.ID] = append(c.copyNodeMap[item.ID], newNodeID)
				c.NodesMap[newNodeID] = &node{
					ID:       newNodeID,
					Categroy: []string{item.Categroy[i]},
				}
				c.NodesMap[newNodeID].OutDegree = make([]int64, len(item.OutDegree))
				copy(c.NodesMap[newNodeID].OutDegree, item.OutDegree)
			}
			item.Categroy = []string{item.Categroy[0]}
		}
	}

	// 网络收缩, 去除网络外的 linksout
	for _, item := range c.NodesMap {
		NewOutDegree := []int64{}
		for _, outDegree := range item.OutDegree {
			if _, ok := c.NodesMap[outDegree]; ok {
				// check 节点都在 graph 中
				// 重构 outDegree
				NewOutDegree = append(NewOutDegree, outDegree)
				// 生成 inDegree
				c.NodesMap[outDegree].InDegree = append(c.NodesMap[outDegree].InDegree, item.ID)

				// 生成复制点的记录
				if copNodeList, ok := c.copyNodeMap[outDegree]; ok {
					// 复制节点的 inDegree 也需要进行复制
					for _, copyNodeID := range copNodeList {
						NewOutDegree = append(NewOutDegree, copyNodeID)
						c.NodesMap[copyNodeID].InDegree = append(c.NodesMap[copyNodeID].InDegree, item.ID)
					}
				}

			}
		}
		item.OutDegree = NewOutDegree
	}

}

type entropyResult struct {
	BigDegreeEntropy        float64
	LittleStructuralEntropy float64
}

// 接受有多个分区的有向图, 计算入度结构熵
func (c *layerGraph) ProgressMultiLayerStructuralEntropy() entropyResult {
	c.cleanGraph()

	m := 0
	subjectSubGraph := make(map[string][]*node)
	topGraphNodeList := []*node{}
	// log.Info().Any("graph", c.NodesMap).Msg("test")
	for _, node := range c.NodesMap {
		m += len(node.InDegree)
		for _, categroy := range node.Categroy {
			subjectSubGraph[categroy] = append(subjectSubGraph[categroy], node)
		}
		topGraphNodeList = append(topGraphNodeList, node)
	}

	pool, _ := ants.NewPool(20)
	defer pool.Release()
	wg := sync.WaitGroup{}

	var TopStructEntropy float64
	wg.Add(1)
	pool.Submit(func() {
		subGraph := getSubsGraph(topGraphNodeList)
		entropy := subGraph.StructEntropy()
		TopStructEntropy = entropy.EntropyIN
		wg.Done()
	})

	var atomEntropyValue atomic.Float64
	for subjectTitle, nodeList := range subjectSubGraph {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			subGraph := getSubsGraph(nodeList)
			entropy := subGraph.StructEntropy()

			vol_Vj := subGraph.EdgeCount

			// 计算 g_j
			g_j := 0
			for _, node := range nodeList {
				for _, outDegreeID := range node.OutDegree {
					if outNode, ok := c.NodesMap[outDegreeID]; ok {
						outPartionFlag := true
						for _, outNodeCategory := range outNode.Categroy {
							if outNodeCategory == subjectTitle {
								outPartionFlag = false
							}
						}
						if outPartionFlag {
							g_j += 1
						}
					}
				}
			}
			if vol_Vj == 0 || m == 0 {
				return
			}
			atomEntropyValue.Add(entropy.EntropyIN*float64(vol_Vj)/float64(m) - math.Log2(float64(vol_Vj)/float64(m))*float64(g_j)/float64(m))

		})
	}
	wg.Wait()
	return entropyResult{TopStructEntropy, atomEntropyValue.Load()}
}

func getSubsGraph(nodeList []*node) *graph.Graph[int64] {

	IDSet := hashset.New[int64]()
	for _, item := range nodeList {
		IDSet.Add(item.ID)
	}

	edgeChan := make(chan *graph.Edge[int64], 1024)
	go func() {
		for _, item := range nodeList {
			for _, linksOut := range item.OutDegree {
				if IDSet.Contains(linksOut) {
					edgeChan <- &graph.Edge[int64]{
						From: item.ID,
						To:   linksOut,
					}
				}

			}
		}
		close(edgeChan)
	}()

	worksGraph := graph.NewGraphFromChan(edgeChan)
	return worksGraph
}
