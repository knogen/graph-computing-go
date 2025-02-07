package extractwikipediadump

import (
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoDataBase struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoDataBase(mongoUrl string, version string) *mongoDataBase {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUrl))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to mongo")
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to ping mongo")
	}

	mdb := &mongoDataBase{
		client:   client,
		database: client.Database("wikipedia_dump_v" + version),
	}
	mdb.initIndex()

	return mdb
}

func (c *mongoDataBase) initIndex() {
	// mods := []mongo.IndexModel{
	// 	{Keys: bson.M{"artID": 1}},
	// 	{Keys: bson.M{"revID": 1}},
	// 	{Keys: bson.M{"ref.doi": 1}},
	// 	{Keys: bson.M{"ref.title": 1}},
	// 	{Keys: bson.M{"match.magID": 1}},
	// 	{Keys: bson.M{"match.mode": 1}},
	// }
	// _, err := c.database.Collection("test").Indexes().CreateMany(ctx, mods)
	// if err != nil {
	// 	log.Warn().Err(err).Msg("failed to create index")
	// }

	mods := []mongo.IndexModel{
		{Keys: bson.M{"year_tags": 1}},
	}
	_, err := c.database.Collection("revision_complete").Indexes().CreateMany(ctx, mods)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create index")
	}

}

func (c *mongoDataBase) close() {
	err := c.client.Disconnect(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to disconnect from mongo")
	}
}

func (c *mongoDataBase) Is_task_exist(taskID string) bool {
	filter := bson.M{"taskID": taskID}
	count, err := c.database.Collection("task").CountDocuments(ctx, filter)
	if err != nil {
		log.Warn().Err(err).Msg("failed to count documents")
	}
	return count > 0
}

func (c *mongoDataBase) Insert_task(taskID string) {
	_, err := c.database.Collection("task").InsertOne(ctx, bson.M{"taskID": taskID})
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert task")
	}
}

func (c *mongoDataBase) Insert_many_pages(pageList []*PageInMongo) {

	opts := options.InsertMany().SetOrdered(false)
	// Convert pageList to []interface{}
	interfaceList := make([]interface{}, len(pageList))
	for i, page := range pageList {
		interfaceList[i] = page
	}

	_, err := c.database.Collection("revision_complete").InsertMany(ctx, interfaceList, opts)

	if err != nil {
		log.Warn().Err(err).Msg("failed to insert many")
	}

}

func (c *mongoDataBase) Insert_many_fail_pages(pageList []*RevisionData) {

	opts := options.InsertMany().SetOrdered(false)
	// Convert pageList to []interface{}
	interfaceList := make([]interface{}, len(pageList))
	for i, page := range pageList {
		interfaceList[i] = page
	}

	_, err := c.database.Collection("revision_fail").InsertMany(ctx, interfaceList, opts)

	if err != nil {
		log.Warn().Err(err).Msg("failed to insert many")
	}

}

func (c *mongoDataBase) Get_pages_by_year(year, ns int) (<-chan *PageInMongo, error) {
	outChan := make(chan *PageInMongo, 32)
	filter := bson.M{"year_tags": year, "ns": ns}
	cursor, err := c.database.Collection("revision_complete").Find(ctx, filter)
	if err != nil {
		log.Warn().Err(err).Msg("failed to find documents")
		return nil, err
	}

	go func() {
		defer close(outChan)
		for cursor.Next(ctx) {
			var page *PageInMongo
			err := cursor.Decode(&page)
			if err != nil {
				log.Warn().Err(err).Msg("failed to decode document")
				continue
			}
			outChan <- page
		}
	}()
	return outChan, nil
}

func (c *mongoDataBase) Get_pages_subject_cats(tags []string, ns int) (<-chan *PageInMongo, error) {
	outChan := make(chan *PageInMongo, 32)
	filter := bson.M{"core_subject_tag": bson.M{"$in": tags}, "ns": ns}
	cursor, err := c.database.Collection("revision_complete").Find(ctx, filter)
	if err != nil {
		log.Warn().Err(err).Msg("failed to find documents")
		return nil, err
	}

	go func() {
		defer close(outChan)
		for cursor.Next(ctx) {
			var page *PageInMongo
			err := cursor.Decode(&page)
			if err != nil {
				log.Warn().Err(err).Msg("failed to decode document")
				continue
			}
			outChan <- page
		}
	}()
	return outChan, nil
}

func (c *mongoDataBase) InsertEntropy(year, graphSize, edgeCount, startPercent, endPercent int, entropyType string, entropy any) {

	document := map[string]any{
		"year":         year,
		"graphSize":    graphSize,
		"edgeCount":    edgeCount,
		"entropyType":  entropyType,
		"entropy":      entropy,
		"startPercent": startPercent,
		"endPercent":   endPercent,
	}
	_, err := c.database.Collection("entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

type InDegreeCountData struct {
	ID     string `bson:"_id"`
	PageID int64  `bson:"pageID"`
	Count  int    `bson:"count"`
	Year   int    `bson:"year"`
}

func (c *mongoDataBase) InsertMnayInDegreeCount(data []*InDegreeCountData) {

	InsertData := make([]interface{}, len(data))
	for i, v := range data {
		InsertData[i] = v
	}
	_, err := c.database.Collection("revision_in_degree_count").InsertMany(ctx, InsertData)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertGoogleDistance(data []GoogleDistance) {

	opts := options.InsertMany().SetOrdered(false)
	// Convert pageList to []interface{}
	interfaceList := make([]interface{}, len(data))
	for i, page := range data {
		interfaceList[i] = page
	}

	_, err := c.database.Collection("google_distance").InsertMany(ctx, interfaceList, opts)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) GetGoogleDistanceByYear(year int) (<-chan *GoogleDistance, error) {

	outChan := make(chan *GoogleDistance, 32)
	filter := bson.M{"year": year, "distance": bson.M{"$gt": 0}}
	cursor, err := c.database.Collection("google_distance").Find(ctx, filter)

	if err != nil {
		log.Warn().Err(err).Msg("failed to find documents")
		return nil, err
	}

	go func() {
		defer close(outChan)
		for cursor.Next(ctx) {
			var page *GoogleDistance
			err := cursor.Decode(&page)
			if err != nil {
				log.Warn().Err(err).Msg("failed to decode document")
				continue
			}
			outChan <- page
		}
	}()
	return outChan, nil
}

func (c *mongoDataBase) InsertSubjectEntropy(year, graphSize, edgeCount int, subject, entropyType string, level int, entropy any) {

	document := map[string]any{
		"year":        year,
		"graphSize":   graphSize,
		"edgeCount":   edgeCount,
		"subject":     subject,
		"level":       level,
		"entropyType": entropyType,
		"entropy":     entropy,
	}
	_, err := c.database.Collection("subject_entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertNewStructuralEntropy(year, level int, entropy any) {

	document := map[string]any{
		"year":    year,
		"level":   level,
		"entropy": entropy,
	}
	_, err := c.database.Collection("new_structural_entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertDistanceComplexity(year, level int, complexity any) {

	document := map[string]any{
		"year":       year,
		"level":      level,
		"complexity": complexity,
	}
	_, err := c.database.Collection("new_distance_compllexity").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertGraphDegreeStats(year int, linksStats map[int]int, linksInStats map[int]int, linksOutStats map[int]int) {

	document := map[string]any{
		"year":          year,
		"linksInStats":  linksInStats,
		"linksStats":    linksStats,
		"linksOutStats": linksOutStats,
	}
	_, err := c.database.Collection("degree_stats").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}
