package wikientropy

import (
	"fmt"
	"sync"

	"graph-computing-go/internal/entropy"
	extractwikipediadump "graph-computing-go/internal/extractWikipediadump"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

func MultilayerSubjectExt() {
	subjectList := []string{"Mathematics", "Physics", "Computer science", "Engineering disciplines", "Medicine",
		"Biology", "Chemistry", "Materials science", "Geology", "Geography", "Environmental science",
		"Economics", "Sociology", "Psychology", "Political science", "Philosophy", "Business", "Art",
		"History"}

	mongoClient := extractwikipediadump.NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	pool, _ := ants.NewPool(20)
	defer pool.Release()
	wg := sync.WaitGroup{}
	for year := 2003; year <= 2024; year += 1 {
		for level := 1; level <= 3; level++ {
			wg.Add(1)
			pool.Submit(func() {
				tags := []string{}
				for _, subjectTitle := range subjectList {
					tag := fmt.Sprintf("lv%d-%s-%d", level, subjectTitle, year)
					tags = append(tags, tag)
				}
				revisionChan, err := mongoClient.Get_pages_subject_cats(tags, 0)
				if err != nil {
					log.Fatal().Err(err).Msg("failed to get pages by year")
				}

				pageMap := pageLinkHandle(revisionChan)
				log.Info().Any("year", year).Any("lv", level).Any("page total", len(pageMap)).Msg("graph entropy start")
				alg := entropy.NewLayerGraph()
				for _, item := range pageMap {
					if item.Redirect != nil {
						continue
					}
					// log.Info().Any("PageID", item.PageID).Any("PageLinksOutIDs", item.PageLinksOutIDs).Any("CoreSubjectTag", item.CoreSubjectTag).Msg("detail")
					alg.SetNode(item.PageID, item.PageLinksOutIDs, item.CoreSubjectTag)
				}
				entropyVal := alg.ProgressMultiLayerStructuralEntropy()
				log.Info().Any("len", len(alg.NodesMap)).Int("year", year).Float64("lv", float64(level)).Float64("BigDegreeEntropy", entropyVal.BigDegreeEntropy).Float64("LittleStructuralEntropy", entropyVal.LittleStructuralEntropy).Msg("graph entropy complete")
				mongoClient.InsertNewStructuralEntropy(year, level, entropyVal)

				// log.Info().Msg("graph entropy complete")
				wg.Done()

			})

		}
	}
	wg.Wait()
}
