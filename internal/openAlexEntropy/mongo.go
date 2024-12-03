package openalexentropy

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

func newMongoDataBase(mongoUrl string, version string) *mongoDataBase {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUrl))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to mongo")
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to ping mongo")
	}

	return &mongoDataBase{
		client:   client,
		database: client.Database("openalex_v" + version),
	}
}

func (c *mongoDataBase) close() {
	err := c.client.Disconnect(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to disconnect from mongo")
	}
}

func (c *mongoDataBase) Get_works() <-chan *worksMongo {

	cursor, err := c.database.Collection("works").Find(ctx, bson.M{})
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert many")
	}
	outchan := make(chan *worksMongo)
	go func() {
		for cursor.Next(ctx) {
			var page worksMongo
			err = cursor.Decode(&page)
			if err != nil {
				log.Warn().Err(err).Msg("failed to decode")
			}
			outchan <- &page
		}
		close(outchan)
	}()
	return outchan
}

func (c *mongoDataBase) InsertEntropy(year, statPercent, endPercent, graphSize int, rankType, entropyType string, entropy any) {

	document := map[string]any{
		"year":        year,
		"statPercent": statPercent,
		"endPercent":  endPercent,
		"graphSize":   graphSize,
		"rankType":    rankType,
		"entropyType": entropyType,
		"entropy":     entropy,
	}
	_, err := c.database.Collection("entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

// 因为有两种熵的计算, 所有结果数需要 > 2
func (c *mongoDataBase) IsEntropyComplete(year, statPercent, endPercent int, rankType string) bool {
	count, err := c.database.Collection("entropy").CountDocuments(ctx, bson.M{
		"year":        year,
		"statPercent": statPercent,
		"endPercent":  endPercent,
		"rankType":    rankType,
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to count documents")
	}
	return count > 1
}
