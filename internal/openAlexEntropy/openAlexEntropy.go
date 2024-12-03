package openalexentropy

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/emirpasic/gods/v2/queues/circularbuffer"
	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/ider-zh/graph-entropy-go/graph"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/sethvargo/go-envconfig"
)

type defaultConfig struct {
	OpenAlex_Version string `env:"OPENALEX_VERSION, default=v0.0.1"`
	MongoUrl         string `env:"Mongo_Url, default=mongo://localhost:27017"`
}

var (
	ctx  = context.Background()
	conf defaultConfig
)

func init() {
	if err := envconfig.Process(ctx, &conf); err != nil {
		log.Fatal().Err(err).Msg("failed to process env var")
	}
}

type percentRange struct {
	Start int
	End   int
}

type graphTask struct {
	PR       percentRange
	RankType string
}

type YearTasks struct {
	Year      int
	GraphTask []graphTask
}

// 对 task 进行初始化, 过滤已经完成的 task
func taskGenerate(mongoClient *mongoDataBase) <-chan *YearTasks {

	// 添加检测范围
	percentPlan := []percentRange{}
	for _, stepEnd := range []int{1, 5, 10, 20, 40, 60, 80} {
		stepStart := 0
		percentPlan = append(percentPlan, percentRange{
			Start: stepStart,
			End:   stepEnd,
		})
	}

	for _, stepEnd := range []int{10, 20, 30, 40, 50, 60, 70, 80} {
		stepStart := stepEnd - 10
		percentPlan = append(percentPlan, percentRange{
			Start: stepStart,
			End:   stepEnd,
		})
	}

	for _, stepEnd := range []int{20, 40, 60, 80} {
		stepStart := stepEnd - 20
		percentPlan = append(percentPlan, percentRange{
			Start: stepStart,
			End:   stepEnd,
		})
	}
	yearStart := 2024
	yearEnd := 1940

	outChan := make(chan *YearTasks)
	go func() {
		for year := yearStart; year >= yearEnd; year -= 1 {
			oneYearTask := YearTasks{
				Year: year,
			}
			for _, plan := range percentPlan {
				for _, rankType := range []string{"total", "current"} {

					if !mongoClient.IsEntropyComplete(year, plan.Start, plan.End, rankType) {
						oneYearTask.GraphTask = append(oneYearTask.GraphTask, graphTask{
							PR:       plan,
							RankType: rankType,
						})
					}
				}
			}
			outChan <- &oneYearTask
		}
		close(outChan)
	}()
	return outChan
}

func maxUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0 // or an appropriate error value
	}
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}

// openalex 的熵计算
// 定义质量好: linksout 高
// 1. 总体 linksin 高, 按总体 linksin 排序
// 2. 当时 linksin 高, 按截至当年的 linksin 排序

// 画 19条曲线, 点数少于 1000 的不计算
// top 1%, 5% , 10% ,20%, 40%, 60%, 80% 100%

// 不用关系子图的连通性, 计算熵只数边数, 不计算连通性

// 时间从 1920 年开始, 到 2024 年结束

// 存储参数: year, percent: [1,2,3..100], graphSize: int, rank: string, entropyMode [struct, degress], entropy: Struct
func MainExt() {
	log.Info().Msg("start")

	// init pool
	const (
		initialPoolSize   = 2                     // 初始线程池大小
		maxPoolSize       = 8                     // 初始线程池大小
		checkInterval     = 10 * 60 * time.Second // 检测间隔
		resetPeakInterval = 60 * time.Minute      // 重置峰值间隔
	)
	pool, _ := ants.NewPool(initialPoolSize)
	defer pool.Release()
	// 线程控制
	go func() {
		// 可用内存不足 10% 时, 减小线程池
		// 可用内存不足 20% 时, 不改变
		// 峰值 alloc 除以线程数, 仍小于剩余内存的 1/2 时, 加大线程池
		allocQueue := circularbuffer.New[uint64](30) // empty (max size is 3)

		ticker := time.NewTicker(checkInterval)
		resetTicker := time.NewTicker(resetPeakInterval)
		defer ticker.Stop()
		defer resetTicker.Stop()

		for {
			select {
			case <-ticker.C:
				// 检测当前内存使用
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				allocQueue.Enqueue(m.Alloc / uint64(pool.Cap()))

			case <-resetTicker.C:
				maxAlloc := maxUint64(allocQueue.Values())

				var info syscall.Sysinfo_t
				err := syscall.Sysinfo(&info)
				if err != nil {
					fmt.Println(err)
				}
				memTotal := info.Totalram * uint64(info.Unit)
				memFree := info.Freeram * uint64(info.Unit)

				percentFree := float64(memFree) / float64(memTotal) * 100

				if percentFree < 10 && pool.Cap() > 1 {
					pool.Tune(pool.Cap() - 1)
					log.Info().Any("poolCap", pool.Cap()).Msg("deduce pool size")
				} else if percentFree < 30 {
					log.Info().Any("poolCap", pool.Cap()).Any("percentFree", percentFree).Msg("keep pool size")
				} else {
					if maxAlloc < (memFree / 2) {
						if pool.Cap() < maxPoolSize {
							pool.Tune(pool.Cap() + 1)
							log.Info().Any("poolCap", pool.Cap()).Any("percentFree", percentFree).Msg("additional pool size")
						}
					} else {
						log.Info().Any("maxAlloc", maxAlloc).Any("memFree", memFree).Any("memTotal", memTotal).Msg("keep pool size")
					}
				}
			}
		}
	}()

	mongoClient := newMongoDataBase(conf.MongoUrl, conf.OpenAlex_Version)
	defer mongoClient.close()

	worksMap := make(map[int64]*worksMongo)

	// 全图的 linksin 数量排序
	totalLinksInCountMap := make(map[int64]int32)

	worksChan := mongoClient.Get_works()

	bar := progressbar.Default(-1, "")
	for item := range worksChan {
		worksMap[item.ID] = item
		totalLinksInCountMap[item.ID] = item.LinksInWorksCount
		bar.Add(1)
	}
	bar.Close()
	log.Info().Int("totalWorks", len(worksMap)).Msg("totalWorks")

	// time.Sleep(10 * time.Second)

	// 全图的 linksin 数量排序
	for task := range taskGenerate(mongoClient) {
		year := task.Year

		if len(task.GraphTask) < 1 {
			log.Info().Any("year", year).Msg("no task")
			continue
		}

		// 完成年份过滤
		log.Info().Any("year", year).Msg("start works filter by year")
		newWorksMap := make(map[int64]*worksMongo)
		for key, item := range worksMap {
			if item.PublicationYear <= int32(year) {
				newWorksMap[key] = worksMap[key]
			}
		}
		log.Info().Msg("works filter by year complete")

		// 过滤掉大于时间点的边
		log.Info().Msg("start works shrink")
		var currentLinksInCountMap map[int64]int32
		worksMap, currentLinksInCountMap = worksShrink(newWorksMap)
		log.Info().Msg("works shrink complete")

		log.Info().Msg("works rank start")
		var worksSliceCurrentRank []*worksMongo
		var worksSliceTotalRank []*worksMongo
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			worksSliceCurrentRank = sortWorksByLinksInMap(worksMap, currentLinksInCountMap)
			wg.Done()
		}()
		go func() {
			worksSliceTotalRank = sortWorksByLinksInMap(worksMap, totalLinksInCountMap)
			wg.Done()
		}()
		wg.Wait()
		log.Info().Msg("works rank  complete")

		for _, plan := range task.GraphTask {
			wg.Add(1)
			pool.Submit(func() {
				log.Info().Any("year", year).Str("rank", plan.RankType).Int("start", plan.PR.Start).Int("end", plan.PR.End).Msg("start")
				var subgraphMap []*worksMongo
				switch plan.RankType {
				case "total":
					subgraphMap = sliceWorksMongoByPercent(worksSliceTotalRank, plan.PR.Start, plan.PR.End)
				case "current":
					subgraphMap = sliceWorksMongoByPercent(worksSliceCurrentRank, plan.PR.Start, plan.PR.End)
				default:
					log.Fatal().Msg("rankType error")
				}

				subGraph := getWorksGraph(subgraphMap)
				subWg := sync.WaitGroup{}
				subWg.Add(2)
				go func() {
					entropy1 := subGraph.DegreeEntropy()
					mongoClient.InsertEntropy(year, plan.PR.Start, plan.PR.End, len(subGraph.Nodes), plan.RankType, "degree", entropy1)
					subWg.Done()
				}()
				go func() {
					entropy2 := subGraph.StructEntropy()
					mongoClient.InsertEntropy(year, plan.PR.Start, plan.PR.End, len(subGraph.Nodes), plan.RankType, "struct", entropy2)
					subWg.Done()
				}()
				subWg.Wait()
				wg.Done()
				log.Info().Any("year", year).Str("rank", plan.RankType).Int("start", plan.PR.Start).Int("end", plan.PR.End).Msg("complete")
			})

		}

		wg.Wait()
		log.Info().Any("year", year).Msg("complete")
	}

}

// 现在是网络子图, 没有子图外的 edge
func getWorksGraph(worksMap []*worksMongo) *graph.Graph[int64] {

	IDSet := hashset.New[int64]()
	for _, item := range worksMap {
		IDSet.Add(item.ID)
	}

	edgeChan := make(chan *graph.Edge[int64], 1024)
	go func() {
		for _, item := range worksMap {
			for _, linksOut := range item.ReferencedWorks {
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

func worksShrink(worksMap map[int64]*worksMongo) (map[int64]*worksMongo, map[int64]int32) {
	// 网络收缩, 正常情况下不用这么处理, 但是有引用时间穿越情况, 需要确认子网
	// 另一种情况是网络子集, 重新计算网络的子图
	newWorksMap := make(map[int64]*worksMongo)
	newLinksInCountMap := make(map[int64]int32)
	for key, item := range worksMap {
		newItem := &worksMongo{
			ID:              item.ID,
			PublicationYear: item.PublicationYear,
			ReferencedWorks: []int64{},
		}

		for _, linksOut := range item.ReferencedWorks {
			if _, ok := worksMap[linksOut]; ok {
				newLinksInCountMap[linksOut] += 1
				newItem.ReferencedWorks = append(newItem.ReferencedWorks, linksOut)
			}
		}
		newWorksMap[key] = newItem

	}
	return newWorksMap, newLinksInCountMap
}

func sortWorksByLinksInMap(worksMap map[int64]*worksMongo, linksInMap map[int64]int32) []*worksMongo {
	// 根据总的 links 长度进行排序
	worksMongoList := []*worksMongo{}
	for key := range worksMap {
		worksMongoList = append(worksMongoList, worksMap[key])
	}

	// 降序
	slices.SortFunc(worksMongoList, func(a, b *worksMongo) int {
		return int(linksInMap[b.ID] - linksInMap[a.ID])
	})
	return worksMongoList
}

func sliceWorksMongoByPercent(worksSlice []*worksMongo, startPercent, endPercent int) []*worksMongo {

	if startPercent == 0 && endPercent == 100 {
		return worksSlice
	}

	startKeyPosition := math.Ceil(float64(len(worksSlice)) * float64(startPercent) / float64(100))
	endKeyPosition := math.Ceil(float64(len(worksSlice)) * float64(endPercent) / float64(100))

	return worksSlice[int(startKeyPosition):int(endKeyPosition)]
}
