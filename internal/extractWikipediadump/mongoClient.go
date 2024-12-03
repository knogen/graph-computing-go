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

func (c *mongoDataBase) Get_pages_by_year(year int) (<-chan *PageInMongo, error) {
	outChan := make(chan *PageInMongo, 32)
	filter := bson.M{"year_tags": year}
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

func (c *mongoDataBase) InsertEntropy(year, graphSize int, entropyType string, entropy any) {

	document := map[string]any{
		"year":        year,
		"graphSize":   graphSize,
		"entropyType": entropyType,
		"entropy":     entropy,
	}
	_, err := c.database.Collection("entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}
