package openalexentropy

import (
	"fmt"
	"graph-computing-go/internal/entropy"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/emirpasic/gods/v2/queues/circularbuffer"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
)

// openalex 多层次的结构熵计算
// 时间从 1920 年开始, 到 2024 年结束
// 存储参数: year, percent: [10, 20, 30..100], graphSize: int, rank: string, entropyMode [struct, degress], entropy: Struct
// 熵值计算思考, 这次计算的是一年的总熵, 从多个学科计算出一个总的熵, 只有时间维度的对比
// 传递到计算方法
// 学术圈网络
func MultilayerSubjectExt() {
	log.Info().Msg("start")

	// 大于 2 的度 最大约 60%, 每年都如此处理一遍, 处理的结果不影响下一年
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

			// 只取第一个 conceptsLv0 确保学科不交叉
			break
		}
		totalLinksInCountMap[item.ID] = item.LinksInWorksCount
		bar.Add(1)
	}
	bar.Close()
	log.Info().Int("subject", len(subjectWorksMap)).Msg("subject Count")

	// 开始按年 - percent 的百分比计算任务.
	for task := range taskGenerate(mongoClient) {
		year := task.Year

		if len(task.GraphTask) < 1 {
			log.Info().Any("year", year).Msg("no task")
			continue
		}

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
			subjectWorksMap[subject] = newWorksMap
		}

		academicCircleMap := make(map[int64]*worksMongo)
		// 因为是学术圈, 所以需要学术圈的网络
		for subject := range subjectWorksMap {
			for key, item := range subjectWorksMap[subject] {
				academicCircleMap[key] = item
			}
		}
		// 对学术圈网络进行收缩
		newWorksMap, currentLinksInCountMap := worksShrink(academicCircleMap)
		// 对整个网络进行过滤, 保留最小 currentLinksInCountMap 入度的 page
		// 这里不再对学科进行此类处理了
		worksList := filterWorksByLinksIn(newWorksMap, currentLinksInCountMap, int32(GatherLinksInCount))
		worksListMap := make(map[int64]*worksMongo)
		for _, item := range worksList {
			worksListMap[item.ID] = item
		}

		// 对 worksListMap 进行解包的各个学科
		// newSubjectWorksMap 是已经过滤了学术圈外的边, 去除了少于 GatherLinksInCount 的边的map
		newSubjectWorksMap := make(map[string]map[int64]*worksMongo)
		for subject := range subjectWorksMap {
			newSubjectWorksMap[subject] = make(map[int64]*worksMongo)
			for key, item := range subjectWorksMap[subject] {
				if _, ok := worksListMap[key]; ok {
					newSubjectWorksMap[subject][key] = item
				}
			}
		}

		// 到此为止, subjectWorksMap 是可以延续复用的
		// 开始分化计算, 执行不同的 plan, 对 subjectWorksMap 进行读取, 产生副本, 处理
		//

		// 这里要提前完成学术圈统计, 不然学科收缩后就丢失了学术圈的交叉边
		// todo
		// 这里要获取的数值有
		// 1. 某个学科到其他学科的边数, 这个数二维表格
		// 2. 学术圈总点数

		wg := sync.WaitGroup{}
		for _, plan := range task.GraphTask {
			wg.Add(1)
			pool.Submit(func() {
				defer wg.Done()
				log.Info().Any("plan", plan).Msg("start plan")
				process_plan(
					year,
					newSubjectWorksMap,
					plan,
					totalLinksInCountMap,
					currentLinksInCountMap,
					mongoClient,
				)
			})
		}
		wg.Wait()

	}
}

// 下面的数据处理是没有复用的
func process_plan(
	year int,
	subjectWorksMap map[string]map[int64]*worksMongo,
	plan graphTask,
	totalLinksInCountMap map[int64]int32,
	currentLinksInCountMap map[int64]int32,
	mongoClient *mongoDataBase,
) {

	// 对学科网络, 再次按照比例减小节点范围
	newSubjectWorksMap := make(map[string]map[int64]*worksMongo)
	for subjectName, subjectMap := range subjectWorksMap {
		newSubjectWorksMap[subjectName] = make(map[int64]*worksMongo)
		var worksList []*worksMongo
		for _, item := range subjectMap {
			worksList = append(worksList, item)
		}

		var subgraphLists []*worksMongo
		switch plan.RankType {
		case "total":
			worksSliceTotalRank := sortWorksByLinksInMap(worksList, totalLinksInCountMap)
			subgraphLists = sliceWorksMongoByPercent(worksSliceTotalRank, plan.PR.Start, plan.PR.End)
		case "current":
			worksSliceCurrentRank := sortWorksByLinksInMap(worksList, currentLinksInCountMap)
			subgraphLists = sliceWorksMongoByPercent(worksSliceCurrentRank, plan.PR.Start, plan.PR.End)
		default:
			log.Fatal().Msg("rankType error")
		}
		for _, item := range subgraphLists {
			newSubjectWorksMap[subjectName][item.ID] = item
		}
	}

	// 再次将缩效后的学科网络转换为学术圈, 因为 page 减小了, 所以边的范围超出了学术圈的范围
	pageIDSubjectMap := make(map[int64][]string)
	academicCircleMap := make(map[int64]*worksMongo)
	for subjectName, subjectMap := range newSubjectWorksMap {
		for key, item := range subjectMap {
			academicCircleMap[key] = item
			pageIDSubjectMap[key] = append(pageIDSubjectMap[key], subjectName)
		}
	}

	alg := entropy.NewLayerGraph()
	for _, item := range academicCircleMap {
		alg.SetNode(item.ID, item.ReferencedWorks, pageIDSubjectMap[item.ID])
	}

	entropyVal := alg.ProgressMultiLayerStructuralEntropy()
	mongoClient.InsertNewStructuralEntropy(year, plan.PR.Start, plan.PR.End, plan.RankType, entropyVal)

}
