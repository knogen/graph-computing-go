package wikientropy

import (
	extractwikipediadump "graph-computing-go/internal/extractWikipediadump"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

// 统计网络的度数分布
func GraphDegreeStats() {
	mongoClient := extractwikipediadump.NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	pool, _ := ants.NewPool(10)
	defer pool.Release()
	wg := sync.WaitGroup{}
	for year := 2024; year >= 2004; year -= 1 {
		wg.Add(1)
		pool.Submit(func() {

			log.Info().Int("year", year).Msg("graph degree stats start")
			revisionChan, err := mongoClient.Get_pages_by_year(year, 0)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to get pages by year")
			}
			pageMap := pageLinkHandle(revisionChan)

			linksInStats := make(map[int]int)
			linksOutStats := make(map[int]int)
			linksStats := make(map[int]int)

			for _, item := range pageMap {
				if item.Redirect != nil || item.Ns != 0 {
					continue
				}
				linksInStats[len(item.PageLinksOutIDs)] += 1
				linksOutStats[len(item.PageLinksOut)] += 1
				linksStats[len(item.PageLinksOutIDs)+len(item.PageLinksOut)] += 1
			}
			mongoClient.InsertGraphDegreeStats(year, linksStats, linksInStats, linksOutStats)

			log.Info().Int("year", year).Msg("graph degree stats finish")
			wg.Done()
		})
	}
	wg.Wait()
}
