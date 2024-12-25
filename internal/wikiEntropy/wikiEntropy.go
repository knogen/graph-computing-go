package wikientropy

import (
	"context"
	"math"
	"slices"
	"strings"
	"sync"

	extractwikipediadump "graph-computing-go/internal/extractWikipediadump"

	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/ider-zh/graph-entropy-go/graph"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/sethvargo/go-envconfig"
)

type defaultConfig struct {
	WikiVersion              string `env:"WIKI_VERSION, default=v0.0.1"`
	WikipediaHistoryDumpPath string `env:"Wikipedia_History_Dump_Path, default=/data/wikipedia/"`
	MongoUrl                 string `env:"Mongo_Url, default=mongo://localhost:27017"`
	WikiTextParserGrpcUrl    string `env:"WikiText_Parser_Grpc_Url, default=localhost:50051"`
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

func taskPorpetionGenerate() []percentRange {

	// 添加检测范围
	percentPlan := []percentRange{}
	for _, stepEnd := range []int{10, 20, 40, 60, 80, 100} {
		stepStart := 0
		percentPlan = append(percentPlan, percentRange{
			Start: stepStart,
			End:   stepEnd,
		})
	}

	// for _, stepEnd := range []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100} {
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

	return percentPlan
}

func Main() {
	mongoClient := extractwikipediadump.NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	pool, _ := ants.NewPool(20)
	defer pool.Release()
	wg := sync.WaitGroup{}
	// for year := 2024; year >= 2001; year -= 1 {
	for year := 2004; year <= 2024; year += 1 {

		log.Info().Int("year", year).Msg("graph entropy start")
		revisionChan, err := mongoClient.Get_pages_by_year(year, 0)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get pages by year")
		}

		pageMap := pageLinkHandle(revisionChan)

		// 按 in-degree 排序
		graphLinksInCountMap := make(map[int64]int)
		for _, item := range pageMap {
			if item.Redirect != nil {
				continue
			}
			for _, linksOutID := range item.PageLinksOutIDs {
				graphLinksInCountMap[linksOutID] += 1
			}
		}

		// totalWikiItemSlice without redirect
		totalWikiItemSlice := rankWikiItemSlices(pageMap, graphLinksInCountMap)

		// 获取 task percent
		for _, taskItem := range taskPorpetionGenerate() {

			wg.Add(1)
			pool.Submit(func() {

				log.Info().Int("year", year).Int("start", taskItem.Start).Int("end", taskItem.End).Msg("graph entropy start")
				wikiItemSlice := sliceWikiItemByPercent(totalWikiItemSlice, taskItem.Start, taskItem.End)

				// 对 page 进行排序
				graphSci := getWorksGraph(wikiItemSlice)

				log.Info().Int("total page:", len(pageMap)).Int("total item:", len(graphSci.Nodes)).Msg("graph build finish")

				entropy1 := graphSci.DegreeEntropy()
				mongoClient.InsertEntropy(year, len(graphSci.Nodes), graphSci.EdgeCount, taskItem.Start, taskItem.End, "degree", entropy1)

				entropy2 := graphSci.StructEntropy()
				mongoClient.InsertEntropy(year, len(graphSci.Nodes), graphSci.EdgeCount, taskItem.Start, taskItem.End, "struct", entropy2)

				log.Info().Int("year", year).Int("start", taskItem.Start).Int("end", taskItem.End).Msg("graph entropy complete")
				wg.Done()
			})
		}
		wg.Wait()
	}
}

func titleFilter(item string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ToLower(item), "_", " "))
}

func getWorksGraph(pageItemSlice []*extractwikipediadump.PageInMongo) *graph.Graph[int64] {

	IDMap := hashset.New[int64]()
	for _, item := range pageItemSlice {
		IDMap.Add(item.PageID)
	}

	edgeChan := make(chan *graph.Edge[int64], 1024)
	go func() {
		bar := progressbar.Default(-1)
		for _, item := range pageItemSlice {

			for _, linksOut := range item.PageLinksOutIDs {

				if !IDMap.Contains(linksOut) {
					continue
				}

				edgeChan <- &graph.Edge[int64]{
					From: item.PageID,
					To:   linksOut,
				}
				bar.Add(1)

			}
		}
		bar.Close()
		close(edgeChan)
	}()

	worksGraph := graph.NewGraphFromChan(edgeChan)

	return worksGraph
}

// // doc: https://github.com/jackc/pgx/blob/master/copy_from_test.go
func pageLinkHandle(revisionChan <-chan *extractwikipediadump.PageInMongo) map[int64]*extractwikipediadump.PageInMongo {

	PageNameMap := make(map[string]*extractwikipediadump.PageInMongo)
	PageOriginNameMap := make(map[string]*extractwikipediadump.PageInMongo)
	pageIDMap := make(map[int64]*extractwikipediadump.PageInMongo)

	for item := range revisionChan {

		// 有冲突就保留后来的, 非 redirect 的 item
		// 正则化的Name
		titleFilterResult := titleFilter(item.Title)
		if v, ok := PageNameMap[titleFilterResult]; !ok {
			PageNameMap[titleFilterResult] = item
		} else {
			if item.Redirect == nil && v.Redirect != nil {
				PageNameMap[titleFilterResult] = item
			}
		}
		// 原始Name
		if v, ok := PageOriginNameMap[item.Title]; !ok {
			PageOriginNameMap[item.Title] = item
		} else {
			if item.Redirect == nil && v.Redirect != nil {
				PageOriginNameMap[item.Title] = item
			}
		}

		pageIDMap[item.PageID] = item
	}
	log.Info().Int("total page:", len(PageNameMap)).Int("total item:", len(pageIDMap)).Msg("redirect finsh")

	// build page redirect
	for _, item := range PageOriginNameMap {
		// redirect
		redirect_title := item.Redirect
		if redirect_title != nil {
			var redirect_id int64
			for i := 0; i < 3; i++ {

				// origin title redirect
				if subItem, ok := PageOriginNameMap[*redirect_title]; ok {
					if subItem.Redirect != nil {
						// redirect to redirect
						redirect_title = subItem.Redirect
						continue
					} else {
						// find the origin page
						redirect_id = subItem.PageID
						break
					}
				}

				// Normalize title redirect
				redirect_title := titleFilter(*redirect_title)
				if subItem, ok := PageNameMap[redirect_title]; ok {
					if subItem.Redirect != nil {
						// redirect to redirect
						// redirect_title = subItem.Redirect
						continue
					} else {
						// find the origin page
						redirect_id = subItem.PageID
						break
					}
				}
				// redirect to non-existing page
				break
			}

			if redirect_id > 0 && redirect_id != item.PageID {
				item.RedirectID = &redirect_id
			}
		}
	}
	log.Info().Int("pageSize", len(PageOriginNameMap)).Msg("page redirect finish")

	// build page linksOut
	linksOutCount := 0
	for _, item := range PageOriginNameMap {

		// page linksOut
		for _, linksOutTitle := range item.PageLinksOut {

			// support redirect 3 times
			var linksOutID int64
			for i := 0; i < 3; i++ {

				if subItem, ok := PageOriginNameMap[linksOutTitle]; ok {
					if subItem.Redirect != nil {
						linksOutTitle = *subItem.Redirect
						continue
					} else {
						linksOutID = subItem.PageID
						break
					}
				}

				linksOutTitle = titleFilter(linksOutTitle)
				if subItem, ok := PageNameMap[linksOutTitle]; ok {
					if subItem.Redirect != nil {
						linksOutTitle = *subItem.Redirect
						continue
					} else {
						linksOutID = subItem.PageID
						break
					}
				}
				break
			}

			if linksOutID > 0 && linksOutID != item.PageID {
				item.PageLinksOutIDs = append(item.PageLinksOutIDs, linksOutID)
				linksOutCount += 1
			}
		}

	}
	log.Info().Int("linksOutCount", linksOutCount).Msg("page linksOut link finish")

	// collected linksin
	// logical
	// page out to pageLinksInIDs, category out to categoryLinksin
	return pageIDMap
}

func rankWikiItemSlices(wikiItemMap map[int64]*extractwikipediadump.PageInMongo, graphLinksInCountMap map[int64]int) []*extractwikipediadump.PageInMongo {
	// 过滤掉 redirect
	wikiItemSlice := []*extractwikipediadump.PageInMongo{}
	for key, item := range wikiItemMap {
		if item.Redirect != nil {
			continue
		}
		wikiItemSlice = append(wikiItemSlice, wikiItemMap[key])
	}
	// 降序
	slices.SortFunc(wikiItemSlice, func(a, b *extractwikipediadump.PageInMongo) int {
		return int(graphLinksInCountMap[b.PageID] - graphLinksInCountMap[a.PageID])
	})
	return wikiItemSlice
}

func sliceWikiItemByPercent(
	wikiItemSlice []*extractwikipediadump.PageInMongo,
	startPercent,
	endPercent int,
) []*extractwikipediadump.PageInMongo {

	if startPercent == 0 && endPercent == 100 {
		return wikiItemSlice
	}
	startKeyPosition := math.Ceil(float64(len(wikiItemSlice)) * float64(startPercent) / float64(100))
	endKeyPosition := math.Ceil(float64(len(wikiItemSlice)) * float64(endPercent) / float64(100))

	return wikiItemSlice[int(startKeyPosition):int(endKeyPosition)]
}
