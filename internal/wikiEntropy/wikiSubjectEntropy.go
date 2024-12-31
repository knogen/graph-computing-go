package wikientropy

import (
	"fmt"
	"sync"

	extractwikipediadump "graph-computing-go/internal/extractWikipediadump"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

func MainSubject() {
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
			for _, subjectTitle := range subjectList {
				wg.Add(1)
				pool.Submit(func() {
					tag := fmt.Sprintf("lv%d-%s-%d", level, subjectTitle, year)
					log.Info().Any("subjectTag", tag).Msg("graph entropy start")
					revisionChan, err := mongoClient.Get_pages_subject_cat(tag, 0)
					if err != nil {
						log.Fatal().Err(err).Msg("failed to get pages by year")
					}

					pageMap := pageLinkHandle(revisionChan)

					var totalWikiItemSlice []*extractwikipediadump.PageInMongo
					for _, item := range pageMap {
						if item.Redirect != nil {
							continue
						}
						totalWikiItemSlice = append(totalWikiItemSlice, item)
					}

					log.Info().Any("subjectTag", tag).Any("page total", len(pageMap)).Msg("graph entropy start")

					graphSci := getWorksGraph(totalWikiItemSlice)

					log.Info().Any("subjectTag", tag).Int("total item:", len(graphSci.Nodes)).Msg("graph build finish")

					entropy1 := graphSci.DegreeEntropy()
					mongoClient.InsertSubjectEntropy(year, len(graphSci.Nodes), graphSci.EdgeCount, subjectTitle, "degree", level, entropy1)

					entropy2 := graphSci.StructEntropy()
					mongoClient.InsertSubjectEntropy(year, len(graphSci.Nodes), graphSci.EdgeCount, subjectTitle, "struct", level, entropy2)

					log.Info().Any("subjectTag", tag).Any("graph Nodes", len(graphSci.Nodes)).Msg("graph entropy complete")
					wg.Done()
				})
			}

			// academic circle
			wg.Add(1)
			pool.Submit(func() {
				var revisionChan = make(chan *extractwikipediadump.PageInMongo, 10)
				var pageMap = make(map[int64]*extractwikipediadump.PageInMongo)
				plhwg := sync.WaitGroup{}
				plhwg.Add(1)
				go func() {
					pageMap = pageLinkHandle(revisionChan)
					plhwg.Done()
				}()

				for _, subjectTitle := range subjectList {
					tag := fmt.Sprintf("lv%d-%s-%d", level, subjectTitle, year)
					log.Info().Any("subjectTag", tag).Msg("graph entropy start")
					subRevisionChan, err := mongoClient.Get_pages_subject_cat(tag, 0)
					if err != nil {
						log.Fatal().Err(err).Msg("failed to get pages by year")
					}
					for item := range subRevisionChan {
						revisionChan <- item
					}
				}
				close(revisionChan)
				plhwg.Wait()

				var totalWikiItemSlice []*extractwikipediadump.PageInMongo
				for _, item := range pageMap {
					if item.Redirect != nil {
						continue
					}
					totalWikiItemSlice = append(totalWikiItemSlice, item)
				}
				subjectTitle := "academic circle"
				graphSci := getWorksGraph(totalWikiItemSlice)

				entropy1 := graphSci.DegreeEntropy()
				mongoClient.InsertSubjectEntropy(year, len(graphSci.Nodes), graphSci.EdgeCount, subjectTitle, "degree", level, entropy1)

				entropy2 := graphSci.StructEntropy()
				mongoClient.InsertSubjectEntropy(year, len(graphSci.Nodes), graphSci.EdgeCount, subjectTitle, "struct", level, entropy2)

				log.Info().Any("subjectTag", subjectTitle).Any("graph Nodes", len(graphSci.Nodes)).Msg("graph entropy complete")
				wg.Done()
			})

		}
	}
	wg.Wait()
}
