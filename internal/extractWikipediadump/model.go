package extractwikipediadump

import (
	"time"

	"github.com/ider-zh/wikipedia-dump-parser/wikiparser"
)

type pageJson struct {
	Title string `json:"title,omitempty"`
	ID    int64  `json:"id,omitempty"`
	// NS         int32    `json:"ns,omitempty"`
	Redirect   string   `json:"redirect,omitempty"`
	Categories []string `json:"categories,omitempty"`
	// Infoboxes  []map[string]interface{} `json:"infoboxes,omitempty"`
	// Images     []struct {
	// 	File    string `json:"file,omitempty"`
	// 	Thumb   string `json:"thumb,omitempty"`
	// 	URL     string `json:"url,omitempty"`
	// 	Caption string `json:"caption,omitempty"`
	// 	Links   []interface {
	// 	} `json:"links,omitempty"`
	// } `json:"images,omitempty"`
	// Coordinates []struct {
	// 	Display  string `json:"display,omitempty"`
	// 	Template string `json:"template,omitempty"`
	// 	Props    struct {
	// 		Region string `json:"region,omitempty"`
	// 	} `json:"props,omitempty"`
	// 	Lat float64 `json:"lat,omitempty"`
	// 	Lon float64 `json:"lon,omitempty"`
	// } `json:"coordinates,omitempty"`
	// Plaintext string `json:"plaintext,omitempty"`
	Links struct {
		Internal []struct {
			Page   string `json:"page,omitempty"`
			Anchor string `json:"anchor,omitempty"`
			Text   string `json:"text,omitempty"`
		} `json:"internal,omitempty"`
		// External []struct {
		// 	Site string `json:"site,omitempty"`
		// 	Text string `json:"text,omitempty"`
		// } `json:"external,omitempty"`
	} `json:"links,omitempty"`
}

type PageInMongo struct {
	RevisionID           int64     `bson:"_id,omitempty"`
	PageID               int64     `bson:"page_id,omitempty"`
	Timestamp            time.Time `bson:"timestamp,omitempty"`
	Ns                   int32     `bson:"ns"`
	Title                string    `bson:"title,omitempty"`
	Redirect             *string   `bson:"redirect,omitempty"`
	PageLinksOut         []string  `bson:"page_links_out,omitempty"`
	PageCategoryLinksOut []string  `bson:"page_category_links_out,omitempty"`
	YearTags             []int     `bson:"year_tags,omitempty"`
	PageLinksOutIDs      []int64   `bson:"-"`
	PageLinksInIDs       []int64   `bson:"-"`
	RedirectID           *int64    `bson:"-"`
	CoreSubjectTag       []string  `bson:"core_subject_tag,omitempty"`
}

type RevisionData struct {
	RevisionID      int64                `bson:"_id,omitempty"`
	Revision        *wikiparser.Revision `bson:"revision,omitempty"`
	YearTags        []int                `bson:"year_tags,omitempty"`
	Ns              int32                `bson:"ns"`
	Timestamp       time.Time            `bson:"timestamp,omitempty"`
	PageID          int64                `bson:"page_id,omitempty"`
	Title           string               `bson:"title,omitempty"`
	RediredTitle    *string              `bson:"redired_title,omitempty"`
	PageLinksOutIDs []int64              `bson:"-"`
	PageLinksInIDs  []int64              `bson:"-"`
	RedirectID      *int64               `bson:"-"`
}

type GoogleDistance struct {
	Year     int     `bson:"year"`
	A        int64   `bson:"a"`
	B        int64   `bson:"b"`
	Distance float64 `bson:"distance"`
}
