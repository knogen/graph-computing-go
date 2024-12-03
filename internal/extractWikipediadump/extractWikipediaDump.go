package extractwikipediadump

import (
	"context"
	"encoding/json"
	"graph-computing-go/internal/protos/wikiTextParser"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/ider-zh/wikipedia-dump-parser/wikiparser"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rs/zerolog/log"
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

func ExtractWikipediaDump() {

	fileReadThreadCount := 20

	files := getAll7zFileNameFromPath(conf.WikipediaHistoryDumpPath)

	log.Info().Msgf("found %d files", len(files))

	mongoClient := NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	filePathList := []string{}
	for _, fileName := range files {
		filePath := conf.WikipediaHistoryDumpPath + "/" + fileName
		if mongoClient.Is_task_exist(filePath) {
			// log.Info().Str("filePath", filePath).Msg("file already exist")
			continue
		}
		filePathList = append(filePathList, filePath)
		// filePathList = append(filePathList, "/mnt/st01/wikipeida_download/20241101/enwiki-20241101-pages-meta-history2.xml-p151515p151573.7z")
		// break

	}

	wikiparser.Parse7zXmlSeparateFlow(filePathList, fileReadThreadCount, []int32{0}, func(pageChan <-chan *wikiparser.Page, filePath string) {

		log.Info().Str("filePath", filePath).Msg("start")
		revisionDataChan := make(chan *RevisionData, 64)
		wgExtractTag := sync.WaitGroup{}
		wgExtractTag.Add(1)
		go func() {
			defer wgExtractTag.Done()
			pageChanHandle(pageChan, revisionDataChan)
		}()

		PageInMongoChan := make(chan *PageInMongo, 64)
		revisionDataFailChan := make(chan *RevisionData, 64)
		wgWikiParser := sync.WaitGroup{}
		wgWikiParser.Add(1)
		go func() {
			defer wgWikiParser.Done()
			revisionDataHandle(revisionDataChan, PageInMongoChan, revisionDataFailChan)
		}()

		wgDB := sync.WaitGroup{}
		wgDB.Add(2)
		go func() {
			defer wgDB.Done()
			PageInMongoHandle(PageInMongoChan)
		}()
		go func() {
			defer wgDB.Done()
			revisionDataFailHandle(revisionDataFailChan)
		}()

		wgExtractTag.Wait()
		close(revisionDataChan)

		wgWikiParser.Wait()
		close(PageInMongoChan)
		close(revisionDataFailChan)

		wgDB.Wait()
		mongoClient.Insert_task(filePath)
		log.Info().Str("filePath", filePath).Msg("complete")
	})

}

func getAll7zFileNameFromPath(path string) []string {
	rd, err := os.ReadDir(path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read dir")
	}
	var files []string
	for _, f := range rd {
		if f.IsDir() {
			continue
		}
		if f.Name()[len(f.Name())-3:] != ".7z" {
			continue
		}
		files = append(files, f.Name())
	}
	return files
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func pageChanHandle(pageChan <-chan *wikiparser.Page, revisionDataChan chan<- *RevisionData) {
	// 找到年度 年度 snapshot, 将其传输到下一步
	for page := range pageChan {
		if page.Ns != 0 {
			continue
		}
		var revisionList []RevisionData
		for _, revision := range page.Revisions {
			timestamp, err := time.Parse("2006-01-02T15:04:05Z", revision.Timestamp)
			if err != nil {
				log.Warn().Err(err).Str("Timestamp", revision.Timestamp).Msg("failed to parse time")
			}

			// filter 2000 年之前的
			if timestamp.Year() < 2000 {
				continue
			}
			revisionList = append(revisionList, RevisionData{
				Revision:  &revision,
				Timestamp: timestamp,
			})
		}

		// 根据时间戳排序
		sort.Slice(revisionList, func(i, j int) bool {
			return revisionList[i].Timestamp.Before(revisionList[j].Timestamp)
		})

		// 获取开始和结束年份
		startYear := revisionList[0].Timestamp.Year()
		endYear := 2024

		yearSnapshotMap := make(map[int]*RevisionData)
		for i := range revisionList {
			currentYear := revisionList[i].Timestamp.Year()
			yearEndDate := time.Date(currentYear, 12, 31, 23, 59, 59, 0, time.UTC)

			if existingSnapshot, exists := yearSnapshotMap[currentYear]; !exists {
				// 如果是第一个该年的数据点
				yearSnapshotMap[currentYear] = &revisionList[i]
			} else {
				// 比较哪个更接近年底
				existingDiff := abs(existingSnapshot.Timestamp.Sub(yearEndDate))
				currentDiff := abs(revisionList[i].Timestamp.Sub(yearEndDate))

				if currentDiff < existingDiff {
					yearSnapshotMap[currentYear] = &revisionList[i]
				}
			}
		}

		// 打标签
		var lastSnapshot *RevisionData
		for year := startYear; year <= endYear; year++ {
			if snapshot, exists := yearSnapshotMap[year]; exists {
				snapshot.YearTags = append(snapshot.YearTags, year)
				lastSnapshot = snapshot
			} else {
				lastSnapshot.YearTags = append(lastSnapshot.YearTags, year)
			}
		}

		for _, snapshot := range yearSnapshotMap {
			snapshot.PageID = page.ID
			snapshot.Title = page.Title
			snapshot.RevisionID = snapshot.Revision.ID
			if page.Redirect != nil {
				snapshot.RediredTitle = &page.Redirect.Title
			}

			revisionDataChan <- snapshot
		}

	}
}

func revisionDataHandle(revisionDataChan <-chan *RevisionData, PageInMongoChan chan<- *PageInMongo, revisionDataFailChan chan<- *RevisionData) {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(conf.WikiTextParserGrpcUrl, opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect grpc server")
	}
	defer conn.Close()

	rpcClient := wikiTextParser.NewWikiTextParserServiceClient(conn)

	for revisionData := range revisionDataChan {
		if revisionData.Revision == nil {
			log.Warn().Str("Title", revisionData.Title).Any("ID", revisionData.PageID).Msg("revisionData.Revision is nil")
			continue
		}
		if revisionData.RediredTitle != nil {
			PageInMongoChan <- &PageInMongo{
				RevisionID: revisionData.Revision.ID,
				PageID:     revisionData.PageID,
				Title:      revisionData.Title,
				Redirect:   revisionData.RediredTitle,
				YearTags:   revisionData.YearTags,
				Timestamp:  revisionData.Timestamp,
			}
			continue
		}
		var jsonText *wikiTextParser.JsonText
		var err error
		retryInterval := 1 * time.Second
		for i := 0; i < 10; i += 1 {
			jsonText, err = rpcClient.GetWikiTextParse(ctx, &wikiTextParser.WikiText{Text: revisionData.Revision.Text.Value})
			if err != nil {
				log.Warn().Str("Title", revisionData.Title).Err(err).Msg("try to retry")
				time.Sleep(retryInterval)
				retryInterval *= 2 // 每次重试间隔翻倍
			}
			break
		}
		if jsonText == nil {
			log.Warn().Str("Title", revisionData.Title).Err(err).Msg("failed to get wikiTextParse")
			revisionDataFailChan <- revisionData
			continue
		}

		var result pageJson

		err = json.Unmarshal([]byte(jsonText.Text), &result)
		if err != nil {
			log.Warn().Str("Title", revisionData.Title).Err(err).Msg("failed to Unmarshal wikiTextParse")
			revisionDataFailChan <- revisionData
			continue
		}
		PageLinksOut := []string{}
		for _, link := range result.Links.Internal {
			PageLinksOut = append(PageLinksOut, link.Page)
		}
		PageInMongoChan <- &PageInMongo{
			RevisionID:           revisionData.Revision.ID,
			PageID:               revisionData.PageID,
			Title:                revisionData.Title,
			YearTags:             revisionData.YearTags,
			PageLinksOut:         PageLinksOut,
			PageCategoryLinksOut: result.Categories,
			Timestamp:            revisionData.Timestamp,
		}

	}
}

func PageInMongoHandle(PageInMongoChan <-chan *PageInMongo) {
	mongoClient := NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	defer mongoClient.close()
	PageInMongoList := []*PageInMongo{}
	for item := range PageInMongoChan {
		PageInMongoList = append(PageInMongoList, item)
		if len(PageInMongoList) >= 10000 {
			mongoClient.Insert_many_pages(PageInMongoList)
			PageInMongoList = []*PageInMongo{}
		}
	}
	if len(PageInMongoList) > 0 {
		mongoClient.Insert_many_pages(PageInMongoList)
	}
}

func revisionDataFailHandle(revisionDataFailChan <-chan *RevisionData) {
	mongoClient := NewMongoDataBase(conf.MongoUrl, conf.WikiVersion)
	defer mongoClient.close()
	revisionDataList := []*RevisionData{}
	for item := range revisionDataFailChan {
		revisionDataList = append(revisionDataList, item)
		if len(revisionDataList) >= 100 {
			mongoClient.Insert_many_fail_pages(revisionDataList)
			revisionDataList = []*RevisionData{}
		}
	}
	if len(revisionDataList) > 0 {
		mongoClient.Insert_many_fail_pages(revisionDataList)
	}
}
