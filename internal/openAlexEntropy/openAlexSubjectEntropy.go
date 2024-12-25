package openalexentropy

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/emirpasic/gods/v2/queues/circularbuffer"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
)

// openalex 的熵计算
// 定义质量好: linksout 高
// 1. 总体 linksin 高, 按总体 linksin 排序
// 2. 当时 linksin 高, 按截至当年的 linksin 排序
// 按照学科分类计算, 只计算学科子图的熵

// 不用关系子图的连通性, 计算熵只数边数, 不计算连通性

// 时间从 1920 年开始, 到 2024 年结束

// 存储参数: year, percent: [1,2,3..100], graphSize: int, rank: string, entropyMode [struct, degress], entropy: Struct
func MainSubjectExt() {
	log.Info().Msg("start")

	// 大于 2 的度 最大约 60%
	GatherLinksInCount := 2
	// init pool
	const (
		initialPoolSize   = 8                     // 初始线程池大小
		maxPoolSize       = 12                    // 初始线程池大小
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

	subjectWorksMap := make(map[string]map[int64]*worksMongo)

	// 全图的 linksin 数量排序
	totalLinksInCountMap := make(map[int64]int32)

	worksChan := mongoClient.Get_works()

	bar := progressbar.Default(-1, "")
	for item := range worksChan {

		// filter useless nodes
		if item.LinksInWorksCount == 0 && len(item.ReferencedWorks) == 0 {
			continue
		}

		for _, lvSubject := range item.ConceptsLv0 {
			if _, ok := subjectWorksMap[lvSubject]; !ok {
				subjectWorksMap[lvSubject] = make(map[int64]*worksMongo)
			}
			subjectWorksMap[lvSubject][item.ID] = item
		}
		totalLinksInCountMap[item.ID] = item.LinksInWorksCount
		bar.Add(1)
	}
	bar.Close()
	log.Info().Int("subject", len(subjectWorksMap)).Msg("subject Count")

	// time.Sleep(10 * time.Second)

	// 全图的 linksin 数量排序
	for task := range taskGenerate(mongoClient) {
		year := task.Year

		if len(task.GraphTask) < 1 {
			log.Info().Any("year", year).Msg("no task")
			continue
		}

		currentSubjectLinksInCountMap := make(map[string]map[int64]int32)
		// 完成年份过滤
		log.Info().Any("year", year).Msg("start works filter by year")
		for subject, worksMap := range subjectWorksMap {
			log.Info().Any("subject", subject).Any("year", year).Msg("start works filter by year")
			newWorksMap := make(map[int64]*worksMongo)
			for key, item := range worksMap {
				if item.PublicationYear <= int32(year) {
					newWorksMap[key] = worksMap[key]
				}
			}

			log.Info().Msg("start works shrink")
			var currentLinksInCountMap map[int64]int32
			newWorksMap, currentLinksInCountMap = worksShrink(newWorksMap)

			// 当前学科的 linksin 排名
			currentSubjectLinksInCountMap[subject] = currentLinksInCountMap

			// 当前学科收缩后的图
			subjectWorksMap[subject] = newWorksMap
		}

		log.Info().Msg("works shrink complete")

		log.Info().Msg("works rank start")

		wg := sync.WaitGroup{}
		for subject := range subjectWorksMap {
			worksMap := subjectWorksMap[subject]
			currentLinksInCountMap := currentSubjectLinksInCountMap[subject]

			var worksSliceCurrentRank []*worksMongo
			var worksSliceTotalRank []*worksMongo
			wg.Add(2)
			worksList := filterWorksByLinksIn(worksMap, currentLinksInCountMap, int32(GatherLinksInCount))
			worksListNew := make([]*worksMongo, len(worksList))
			copy(worksListNew, worksList)
			go func() {
				worksSliceCurrentRank = sortWorksByLinksInMap(worksListNew, currentLinksInCountMap)
				wg.Done()
			}()
			go func() {
				worksSliceTotalRank = sortWorksByLinksInMap(worksList, totalLinksInCountMap)
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
						mongoClient.InsertSubjectEntropy(subject, year, plan.PR.Start, plan.PR.End, len(subGraph.Nodes), subGraph.EdgeCount, plan.RankType, "degree", entropy1)
						subWg.Done()
					}()
					go func() {
						entropy2 := subGraph.StructEntropy()
						mongoClient.InsertSubjectEntropy(subject, year, plan.PR.Start, plan.PR.End, len(subGraph.Nodes), subGraph.EdgeCount, plan.RankType, "struct", entropy2)
						subWg.Done()
					}()
					subWg.Wait()
					wg.Done()
					log.Info().Any("year", year).Str("rank", plan.RankType).Int("start", plan.PR.Start).Int("end", plan.PR.End).Msg("complete")
				})

			}
		}

		wg.Wait()
		log.Info().Any("year", year).Msg("complete")
	}
}
