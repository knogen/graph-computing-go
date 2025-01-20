// 计算历年的 Wikipedia snapshop 的学术圈范围的 Google Distance
// 距离越小约好
package wikipediagoogledistance

import (
	"context"
	"math"
	"strings"
	"sync"

	extractwikipediadump "graph-computing-go/internal/extractWikipediadump"

	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
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

// 计算学术圈点之间的 google 距离
func Main() {
	mongoClient := extractwikipediadump.NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	pool, _ := ants.NewPool(20)
	defer pool.Release()
	wg := sync.WaitGroup{}
	for year := 2004; year <= 2024; year += 1 {
		currentYear := year
		wg.Add(1)
		pool.Submit(func() {

			log.Info().Int("year", currentYear).Msg("start google distance")
			revisionChan, err := mongoClient.Get_pages_by_year(currentYear, 0)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to get pages by year")
			}

			pageMap := pageLinkHandle(revisionChan)

			// get linksinMap
			degreeInMap := make(map[int64]*hashset.Set[int64])
			coreDisciplineSet := hashset.New[int64]()

			for _, item := range pageMap {
				if item.Redirect != nil {
					continue
				}

				// 记录需要计算的学科点
				if item.CoreSubjectTag != nil {
					coreDisciplineSet.Add(item.PageID)
				}
			}

			for _, item := range pageMap {
				if item.Redirect != nil {
					continue
				}

				// 构建 linksin
				for _, linksOutID := range item.PageLinksOutIDs {

					// 只记录学术圈的点
					if !coreDisciplineSet.Contains(linksOutID) {
						continue
					}

					if _, ok := degreeInMap[linksOutID]; !ok {
						degreeInMap[linksOutID] = hashset.New[int64]()
					}
					degreeInMap[linksOutID].Add(item.PageID)
				}
			}

			// 计算 Google Distance
			// google距离=[log(max)-log(交集)]/[log(total)-log(min)]
			coreDisciplineList := coreDisciplineSet.Values()
			totalPageCount := len(pageMap)
			for i := 0; i < len(coreDisciplineList); i += 1 {

				dumpData := []extractwikipediadump.GoogleDistance{}
				for j := i + 1; j < len(coreDisciplineList); j += 1 {
					aID := coreDisciplineList[i]
					bID := coreDisciplineList[j]
					if aID > bID {
						aID, bID = bID, aID
					}
					if _, ok := degreeInMap[aID]; !ok {
						continue
					}
					if _, ok := degreeInMap[bID]; !ok {
						continue
					}

					v_Intersection := degreeInMap[aID].Intersection(degreeInMap[bID]).Size()

					if v_Intersection == 0 {
						continue
					}
					v_max := 0
					v_min := 0
					if degreeInMap[aID].Size() > degreeInMap[bID].Size() {
						v_max = degreeInMap[aID].Size()
						v_min = degreeInMap[bID].Size()
					} else {
						v_max = degreeInMap[bID].Size()
						v_min = degreeInMap[aID].Size()
					}
					if v_min == 0 || v_max == 0 {
						continue
					}

					distance := (math.Log2(float64(v_max)) - math.Log2(float64(v_Intersection))) / (math.Log2(float64(totalPageCount)) - math.Log2(float64(v_min)))

					dumpData = append(dumpData, extractwikipediadump.GoogleDistance{
						Year:     currentYear,
						A:        aID,
						B:        bID,
						Distance: distance,
					})
				}
				if len(dumpData) > 0 {
					mongoClient.InsertGoogleDistance(dumpData)
				}

			}

			log.Info().Int("year", year).Msg("graph distance complete")
			wg.Done()
		})

	}
	wg.Wait()
}

func titleFilter(item string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ToLower(item), "_", " "))
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
