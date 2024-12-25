package openalexentropy

type worksMongo struct {
	ID                   int64    `json:"id" bson:"_id"`
	PublicationYear      int32    `json:"publication_year" bson:"publication_year"`
	ReferencedWorksCount int32    `json:"referenced_works_count" bson:"referenced_works_count,omitempty"`
	ReferencedWorks      []int64  `json:"referenced_works" bson:"referenced_works,omitempty"`
	LinksInWorksCount    int32    `json:"-" bson:"links_in_works,omitempty"` //require computing
	ConceptsLv0          []string `json:"-" bson:"Concepts_lv0,omitempty"`   //require computing
}
