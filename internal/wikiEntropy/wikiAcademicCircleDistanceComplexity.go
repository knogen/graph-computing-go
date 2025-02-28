package wikientropy

import (
	"fmt"
	"strings"
	"sync"

	distanceComplexity "graph-computing-go/internal/distanceComplexity"
	extractwikipediadump "graph-computing-go/internal/extractWikipediadump"

	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

// 使用学术圈, 和学术圈中每篇文章的距离, 计算改进过的距离复杂度
func AcandemicDistanceComplexity() {
	subjectList := []string{"Mathematics", "Physics", "Computer science", "Engineering disciplines", "Medicine",
		"Biology", "Chemistry", "Materials science", "Geology", "Geography", "Environmental science",
		"Economics", "Sociology", "Psychology", "Political science", "Philosophy", "Business", "Art",
		"History"}

	mongoClient := extractwikipediadump.NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	pool, _ := ants.NewPool(20)
	defer pool.Release()
	wg := sync.WaitGroup{}
	for year := 2004; year <= 2024; year += 1 {
		for level := 1; level <= 3; level++ {
			wg.Add(1)
			pool.Submit(func() {

				dcg := distanceComplexity.NewDistanceGraph()
				nodeIDSet := hashset.New[int64]()
				tags := []string{}
				for _, subjectTitle := range subjectList {
					tag := fmt.Sprintf("lv%d-%s-%d", level, subjectTitle, year)
					tags = append(tags, tag)
				}

				revisionChan, err := mongoClient.Get_pages_subject_cats(tags, 0)
				if err != nil {
					log.Fatal().Err(err).Msg("failed to get pages by year")
				}
				// 设置节点的 category
				for item := range revisionChan {
					if item.Redirect != nil {
						continue
					}
					nodeIDSet.Add(item.PageID)

					// 过滤掉多余的 tag 信息
					coreTags := []string{}
					lvStart := fmt.Sprintf("lv%d-", level)
					for _, tag := range item.CoreSubjectTag {
						if !strings.HasPrefix(tag, lvStart) {
							continue
						}

						parts := strings.Split(tag, "-")
						filteredTag := strings.Join(parts[1:len(parts)-1], "-")

						for _, subjectTitle := range subjectList {
							if filteredTag == subjectTitle {
								coreTags = append(coreTags, subjectTitle)
							}
						}
					}
					dcg.SetNodeCategory(item.PageID, coreTags)
				}

				distanceChan, err := mongoClient.GetGoogleDistanceByYear(year)
				if err != nil {
					log.Fatal().Err(err).Msg("failed to get pages by year")
				}

				for item := range distanceChan {
					if !nodeIDSet.Contains(item.A) || !nodeIDSet.Contains(item.B) {
						continue
					}
					dcg.SetEdge(item.A, item.B, item.Distance)
				}

				complexityVal := dcg.ProgressDistanceComplexity()
				log.Info().Any("len", len(dcg.NodesMap)).Int("year", year).Float64("lv", float64(level)).Float64("BigDegreeEntropy", complexityVal.BigComplexity).Float64("LittleStructuralEntropy", complexityVal.LittlComplexity).Msg("graph entropy complete")
				mongoClient.InsertDistanceComplexity(year, level, complexityVal)

				// log.Info().Msg("graph entropy complete")
				wg.Done()

			})

		}
	}
	wg.Wait()
}
