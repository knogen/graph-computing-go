package distanceComplexity

import (
	"math"

	"github.com/rs/zerolog/log"
)

// 存储无向网络的权重边
// 边的存储保持方向, 从大点到小点, 减少存储占用
type node struct {
	ID        int64
	OutDegree []int64
	InDegree  []int64
	Categroy  []string
}

type DistanceGraph struct {
	NodesMap    map[int64]*node
	distanceMap map[int64]map[int64]float64
	maxID       int64
	copyNodeMap map[int64][]int64 // 保存复制点的记录, 生成入度的时候生效
}

func NewDistanceGraph() *DistanceGraph {
	return &DistanceGraph{
		NodesMap:    make(map[int64]*node),
		copyNodeMap: make(map[int64][]int64),
		distanceMap: make(map[int64]map[int64]float64),
	}
}

// 所有的节点都会加入 graph 进行计算
func (c *DistanceGraph) SetEdge(ID, targetID int64, weight float64) {
	if _, ok := c.NodesMap[ID]; !ok {
		c.NodesMap[ID] = &node{
			ID:        ID,
			OutDegree: []int64{targetID},
		}
	} else {
		c.NodesMap[ID].OutDegree = append(c.NodesMap[ID].OutDegree, targetID)
	}

	if _, ok := c.NodesMap[targetID]; !ok {
		c.NodesMap[ID] = &node{
			ID: targetID,
		}
	}

	// 生成 distanceMap
	var a, b int64
	if ID > targetID {
		a = ID
		b = targetID
	} else {
		a = targetID
		b = ID
	}
	if _, ok := c.distanceMap[a]; !ok {
		c.distanceMap[a] = make(map[int64]float64)
	}
	c.distanceMap[a][b] = weight

	// 记录最大 ID
	if ID > c.maxID {
		c.maxID = ID
	}
	if targetID > c.maxID {
		c.maxID = targetID
	}
}

// set node category, 将范围内的点都 set category
func (c *DistanceGraph) SetNodeCategory(ID int64, Categroy []string) {
	if _, ok := c.NodesMap[ID]; ok {
		c.NodesMap[ID].Categroy = Categroy
	} else {
		c.NodesMap[ID] = &node{
			ID:       ID,
			Categroy: Categroy,
		}
	}

	// 记录最大 ID
	if ID > c.maxID {
		c.maxID = ID
	}
}

// 网络收缩, 对交叉学科进行复制, 并去除交叉性
func (c *DistanceGraph) cleanGraph() {

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

				// copy distance
				c.distanceMap[newNodeID] = c.distanceMap[item.ID]
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

type complexityResult struct {
	BigComplexity    float64
	LittlComplexity  float64
	ModuleComplexity map[string]float64
	ModuleNodeSize   map[string]int64
	ModuleEdgeCount  map[string]int64
}

// 接受有多个分区的有向图, 计算入度结构熵
func (c *DistanceGraph) ProgressDistanceComplexity() complexityResult {
	c.cleanGraph()

	// 统计模块内的距离, 模块间的距离 va * 2,ga
	// vall = SUM(va*2) + SUM(ga)
	var vall float64
	moduleInnerNodeDistanceTotalMap := make(map[string]map[int64][]float64)
	moduleOutterDistanceTotalMap := make(map[string]float64)
	moduleInnerDistanceTotalMap := make(map[string]float64)

	moduleEdgeCount := make(map[string]int64)
	// log.Debug().Any("distanceMap", c.distanceMap).Msg("test")

	for IDA, distanceMap := range c.distanceMap {
		// 只计算图内的节点
		if itemA, ok := c.NodesMap[IDA]; ok {
			moduleA := itemA.Categroy[0]
			for IDB, distance := range distanceMap {
				if itemB, ok := c.NodesMap[IDB]; ok {
					vall += distance * 2
					moduleB := itemB.Categroy[0]
					if moduleA == moduleB {
						if _, ok := moduleInnerNodeDistanceTotalMap[moduleA]; !ok {
							moduleInnerNodeDistanceTotalMap[moduleA] = make(map[int64][]float64)
						}
						moduleEdgeCount[moduleA] += 1
						moduleInnerNodeDistanceTotalMap[moduleA][IDA] = append(moduleInnerNodeDistanceTotalMap[moduleA][IDA], distance)
						moduleInnerNodeDistanceTotalMap[moduleA][IDB] = append(moduleInnerNodeDistanceTotalMap[moduleA][IDB], distance)
						moduleInnerDistanceTotalMap[moduleA] += distance * 2
					} else {
						moduleOutterDistanceTotalMap[moduleA] += distance
						moduleOutterDistanceTotalMap[moduleB] += distance
					}
				}
			}
		}
	}
	// log.Info().Any("vall", vall).Any("moduleInnerNodeDistanceTotalMap", moduleInnerNodeDistanceTotalMap).Any("moduleOutterDistanceTotalMap", moduleOutterDistanceTotalMap).Msg("test")

	var moduleComplexSum float64
	nodeEntropyMap := make(map[int64]float64)

	moduleNodeSize := make(map[string]int64)
	moduleComplexity := make(map[string]float64)
	for moduleName, distanceMap := range moduleInnerNodeDistanceTotalMap {

		// 计算模块内节点的距离复杂度
		for nodeID, distanceList := range distanceMap {
			moduleNodeSize[moduleName] += 1
			var sum float64 //某个点模块内连接路径总长度
			for _, distance := range distanceList {
				sum += distance
			}

			var entropyNode float64 //模块的距离复杂度
			for _, distance := range distanceList {
				entropyNode -= distance / sum * math.Log2(distance/sum)

				// pass
				log.Debug().Any("distance", distance).
					Any("sum", sum).
					Any("nodeID", nodeID).
					Msg("-- node Entropy detail")
			}

			// pass
			log.Debug().Any("entropyNode", entropyNode).
				Any("nodeID", nodeID).
				Msg("node Entropy")

			nodeEntropyMap[nodeID] = entropyNode
		}

		var hModule float64
		for nodeID, distanceList := range distanceMap {
			var sum float64 //某个点模块内连接路径总长度
			for _, distance := range distanceList {
				sum += distance
			}
			hModule += sum / moduleInnerDistanceTotalMap[moduleName] * nodeEntropyMap[nodeID]

		}
		moduleComplexity[moduleName] = hModule
		moduleEntropy := hModule*moduleInnerDistanceTotalMap[moduleName]/vall - math.Log2(moduleInnerDistanceTotalMap[moduleName]/vall)*moduleOutterDistanceTotalMap[moduleName]/vall
		moduleComplexSum += moduleEntropy

		log.Debug().Any("hmodule", hModule).
			Any("moduleName", moduleName).
			Msg("hModel")
	}

	// 计算 distance complexity

	// collect distacne
	distance_collect := make(map[int64][]float64)
	for nodeA, distanceList := range c.distanceMap {
		for nodeB, distance := range distanceList {
			distance_collect[nodeA] = append(distance_collect[nodeA], distance)
			distance_collect[nodeB] = append(distance_collect[nodeB], distance)
		}
	}
	sumRi := []float64{}
	Hi := []float64{}
	for _, distanceList := range distance_collect {
		var sumR float64 //路径总长度
		for _, distance := range distanceList {
			sumR += distance

		}
		sumRi = append(sumRi, sumR)

		var sumPi float64
		for _, distance := range distanceList {
			pi := distance / sumR
			sumPi += -pi * math.Log2(pi)

			if math.IsNaN(sumPi) {
				log.Info().Any("pi", pi).Any("distance", distance).Any("sumR", sumR).Msg("sumPi")
				break
			}
		}
		Hi = append(Hi, sumPi)
	}
	var G float64
	for _, value := range sumRi {
		G += value
	}
	var distanceComplex float64
	for index := range sumRi {
		distanceComplex += Hi[index] * sumRi[index] / G
	}
	log.Debug().Any("modelComplexity", moduleComplexity).
		Any("moduleNodeSize", moduleNodeSize).
		Any("moduleEdgeCount", moduleEdgeCount).
		Msg("moduleComplexity")
	return complexityResult{distanceComplex, moduleComplexSum, moduleComplexity, moduleNodeSize, moduleEdgeCount}
}
