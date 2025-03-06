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

	// cursor, err := c.database.Collection("works").Find(ctx, bson.M{"links_in_works": bson.M{"$gte": 2}})
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

func (c *mongoDataBase) InsertDistanceComplexity(year, complexity any) {

	document := map[string]any{
		"year":       year,
		"complexity": complexity,
	}
	_, err := c.database.Collection("new_distance_complexity").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertTopDisciplineDistanceComplexity(year, discipline, complexity any) {

	document := map[string]any{
		"year":       year,
		"discipline": discipline,
		"complexity": complexity,
	}
	_, err := c.database.Collection("top_discipline_distance_complexity").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertEntropy(year, startPercent, endPercent, graphSize, edgeCount int, rankType, entropyType string, entropy any) {

	document := map[string]any{
		"year":         year,
		"startPercent": startPercent,
		"endPercent":   endPercent,
		"graphSize":    graphSize,
		"edgeCount":    edgeCount,
		"rankType":     rankType,
		"entropyType":  entropyType,
		"entropy":      entropy,
	}
	_, err := c.database.Collection("entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertSubjectEntropy(subject string, year, startPercent, endPercent, graphSize, edgeCount int, rankType, entropyType string, entropy any) {

	document := map[string]any{
		"year":         year,
		"subject":      subject,
		"startPercent": startPercent,
		"endPercent":   endPercent,
		"graphSize":    graphSize,
		"edgeCount":    edgeCount,
		"rankType":     rankType,
		"entropyType":  entropyType,
		"entropy":      entropy,
	}
	_, err := c.database.Collection("subject_entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertNewStructuralEntropy(year, startPercent, endPercent int, rankType, entropy any) {

	document := map[string]any{
		"year":         year,
		"startPercent": startPercent,
		"endPercent":   endPercent,
		"rankType":     rankType,
		"entropy":      entropy,
	}
	_, err := c.database.Collection("new_structural_entropy").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) InsertNewStructuralEntropySubDiscipline(year, startPercent, endPercent int, rankType, subject string, entropy any) {

	document := map[string]any{
		"year":         year,
		"startPercent": startPercent,
		"endPercent":   endPercent,
		"rankType":     rankType,
		"subject":      subject,
		"entropy":      entropy,
	}
	_, err := c.database.Collection("new_structural_entropy_subdiscipline").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

// 因为有两种熵的计算, 所有结果数需要 > 2
func (c *mongoDataBase) IsEntropyComplete(year, startPercent, endPercent int, rankType string) bool {
	count, err := c.database.Collection("entropy").CountDocuments(ctx, bson.M{
		"year":         year,
		"startPercent": startPercent,
		"endPercent":   endPercent,
		"rankType":     rankType,
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to count documents")
	}
	return count > 1
}

// 因为有两种熵的计算, 所有结果数需要 > 2

func (c *mongoDataBase) InsertGraphDegreeStats(year int, graphSize int, linksStats map[int]int, linksInStats map[int]int, linksOutStats map[int]int) {

	document := map[string]any{
		"year":          year,
		"graphSize":     graphSize,
		"linksInStats":  linksInStats,
		"linksStats":    linksStats,
		"linksOutStats": linksOutStats,
	}
	_, err := c.database.Collection("degree_stats").InsertOne(ctx, document)
	if err != nil {
		log.Warn().Err(err).Msg("failed to insert one")
	}
}

func (c *mongoDataBase) GetSubConcepts(topConcept string) []conceptsMongo {

	// cursor, err := c.database.Collection("works").Find(ctx, bson.M{"links_in_works": bson.M{"$gte": 2}})
	cursor, err := c.database.Collection("concepts").Find(ctx, bson.M{"ancestors.displayname": topConcept})
	if err != nil {
		log.Warn().Err(err).Msg("failed to get many")
	}
	result := []conceptsMongo{}
	for cursor.Next(ctx) {
		var page conceptsMongo
		err = cursor.Decode(&page)
		if err != nil {
			log.Warn().Err(err).Msg("failed to decode")
		}
		result = append(result, page)
	}
	return result
}

func (c *mongoDataBase) Get_concepts_works(concepts []string, level int) <-chan *worksMongo {

	filter := bson.M{}
	if level == 0 {
		filter = bson.M{"Concepts_lv0": bson.M{"$in": concepts}}
	} else if level == 1 {
		filter = bson.M{"Concepts_lv1": bson.M{"$in": concepts}}
	} else if level == 2 {
		filter = bson.M{"Concepts_lv2": bson.M{"$in": concepts}}
	}

	cursor, err := c.database.Collection("works").Find(ctx, filter)
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
