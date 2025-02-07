package openalexentropy

import (
	"graph-computing-go/internal/entropy"
	"sync"

	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
)

// 计算 Medicine, Biology, Computer science, Psychology 一级4个学科的结构熵
// 只过滤一次总入度大于0的节点, 然后收缩网络
func SubDispolieDistructuralEntropyDemo() {
	log.Info().Msg("start")

	// 大于 2 的度 最大约 60%, 每年都如此处理一遍, 处理的结果不影响下一年
	GatherLinksInCount := 2

	pool, _ := ants.NewPool(2)
	defer pool.Release()

	mongoClient := newMongoDataBase(conf.MongoUrl, conf.OpenAlex_Version)
	defer mongoClient.close()

	// for _, subjectName := range []string{"Medicine", "Biology", "Computer science", "Psychology"} {
	for _, subjectName := range []string{"Biology"} {
		subConceptList := mongoClient.GetSubConcepts(subjectName)

		subConceptStringList := []string{}
		for _, item := range subConceptList {
			subConceptStringList = append(subConceptStringList, item.DisplayName)
		}
		log.Info().Any("subConceptStringList", subConceptStringList).Msg("subConceptStringList")
		subConceptStringMap := hashset.New(subConceptStringList...)

		subjectWorksMap := make(map[string]map[int64]*worksMongo)

		// 找到二级学科的所有 works
		totalLinksInCountMap := make(map[int64]int32)
		worksChan := mongoClient.Get_concepts_works(subConceptStringList, 1)
		bar := progressbar.Default(-1, "")
		for item := range worksChan {

			// filter useless nodes
			if item.LinksInWorksCount == 0 && len(item.ReferencedWorks) == 0 {
				continue
			}

			// todo 这里特定的 concept1
			for _, lvSubject := range item.ConceptsLv1 {

				// 只保留有关的二级学科 tag
				if !subConceptStringMap.Contains(lvSubject) {
					continue
				}
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

		// 开始按年 - percent 的百分比计算任务.
		for task := range taskGenerate2() {
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
					process_subdispline_plan(
						year,
						newSubjectWorksMap,
						plan,
						totalLinksInCountMap,
						currentLinksInCountMap,
						mongoClient,
						subjectName,
					)
				})
			}
			wg.Wait()
		}

	}

}

func process_subdispline_plan(
	year int,
	subjectWorksMap map[string]map[int64]*worksMongo,
	plan graphTask,
	totalLinksInCountMap map[int64]int32,
	currentLinksInCountMap map[int64]int32,
	mongoClient *mongoDataBase,
	topSubject string,
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
	mongoClient.InsertNewStructuralEntropySubDispline(year, plan.PR.Start, plan.PR.End, plan.RankType, topSubject, entropyVal)

}

// 对 task 进行初始化, 过滤已经完成的 task
func taskGenerate2() <-chan *YearTasks {

	// 添加检测范围
	percentPlan := []percentRange{}
	for _, stepEnd := range []int{100, 40, 10, 20, 60, 80} {
		stepStart := 0
		percentPlan = append(percentPlan, percentRange{
			Start: stepStart,
			End:   stepEnd,
		})
	}

	// 废弃的方案, 分层的网络会出现以外的情况
	// for _, stepEnd := range []int{10, 20, 30, 40, 50, 60, 70, 80, 100} {
	// 	stepStart := stepEnd - 10
	// 	percentPlan = append(percentPlan, percentRange{
	// 		Start: stepStart,
	// 		End:   stepEnd,
	// 	})
	// }

	// for _, stepEnd := range []int{20, 40, 60, 80, 100} {
	// 	stepStart := stepEnd - 20
	// 	percentPlan = append(percentPlan, percentRange{
	// 		Start: stepStart,
	// 		End:   stepEnd,
	// 	})
	// }
	yearStart := 2024
	yearEnd := 1940
	// yearEnd := 2010

	outChan := make(chan *YearTasks)
	go func() {
		for year := yearStart; year >= yearEnd; year -= 1 {
			oneYearTask := YearTasks{
				Year: year,
			}
			for _, plan := range percentPlan {
				for _, rankType := range []string{"current"} {

					// if !mongoClient.IsEntropyComplete(year, plan.Start, plan.End, rankType) {
					oneYearTask.GraphTask = append(oneYearTask.GraphTask, graphTask{
						PR:       plan,
						RankType: rankType,
					})
					// }
				}
			}
			outChan <- &oneYearTask
		}
		close(outChan)
	}()
	return outChan
}
