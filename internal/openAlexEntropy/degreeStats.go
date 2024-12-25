package openalexentropy

import (
	"github.com/ider-zh/graph-entropy-go/graph"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
)

func GraphDegreeStats() {
	mongoClient := newMongoDataBase(conf.MongoUrl, conf.OpenAlex_Version)

	log.Info().Msg("start")

	defer mongoClient.close()

	worksMap := make(map[int64]*worksMongo)
	worksChan := mongoClient.Get_works()

	bar := progressbar.Default(-1, "geting works")
	for item := range worksChan {
		worksMap[item.ID] = item
		bar.Add(1)
	}
	bar.Close()
	log.Info().Int("count", len(worksMap)).Msg("totalWorks")

	// 全图的 linksin 数量排序
	for year := 2024; year >= 1940; year -= 1 {
		log.Info().Any("year", year).Msg("start task")

		newWorksMap := make(map[int64]*worksMongo)
		for key, item := range worksMap {
			if item.PublicationYear <= int32(year) {
				newWorksMap[key] = worksMap[key]
			}
		}
		worksMap, _ = worksShrink(newWorksMap)
		subGraph := getWorksGraphByMap(worksMap)

		linksInStats := make(map[int]int)
		linksOutStats := make(map[int]int)
		linksStats := make(map[int]int)

		for _, item := range subGraph.Nodes {
			linksInStats[len(item.LinksIn)] += 1
			linksOutStats[len(item.LinksOut)] += 1
			linksStats[len(item.LinksIn)+len(item.LinksOut)] += 1
		}
		mongoClient.InsertGraphDegreeStats(year, len(worksMap), linksStats, linksInStats, linksOutStats)

		log.Info().Any("year", year).Msg("task complete")
	}

}

func getWorksGraphByMap(worksMap map[int64]*worksMongo) *graph.Graph[int64] {

	edgeChan := make(chan *graph.Edge[int64], 1024)
	go func() {
		for _, item := range worksMap {
			for _, linksOut := range item.ReferencedWorks {
				edgeChan <- &graph.Edge[int64]{
					From: item.ID,
					To:   linksOut,
				}
			}
		}
		close(edgeChan)
	}()

	worksGraph := graph.NewGraphFromChan(edgeChan)
	return worksGraph
}
