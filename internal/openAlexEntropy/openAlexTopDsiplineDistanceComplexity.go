package openalexentropy

import (
	"graph-computing-go/internal/distanceComplexity"
	"sync"

	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

type workConfig struct {
	Year    int
	Concept string
}

// 泛型函数，用于获取两个切片的交集
func Intersection[T comparable](s1, s2 []T) []T {
	m := make(map[T]bool)
	for _, v := range s1 {
		m[v] = true
	}
	var result []T
	for _, v := range s2 {
		if m[v] {
			result = append(result, v)
		}
	}
	return result
}

// Contains 泛型函数，用于判断元素 item 是否存在于切片 list 中
func Contains[T comparable](list []T, item T) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// 使用学术圈, 和学术圈中每篇文章的距离, 计算改进过的距离复杂度
// 计算一级学科的距离复杂度&KQI， 使用二级学科作为模块，三级学科作为节点
func TopDisciplineDistanceComplexity() {
	subjectList := []string{"Mathematics", "Physics", "Computer science", "Engineering", "Medicine",
		"Biology", "Chemistry", "Materials science", "Geology", "Geography", "Environmental science",
		"Economics", "Sociology", "Psychology", "Political science", "Philosophy", "Business", "Art",
		"History"}

	mongoClient := newMongoDataBase(conf.MongoUrl, conf.OpenAlex_Version)
	defer mongoClient.close()

	// 获得各个一级学科的二级学科
	conceptLv0toLv1 := make(map[string][]string)
	conceptLv1toLv0 := make(map[string][]string)
	for _, lv0SubjectTitle := range subjectList {
		subConceptList := mongoClient.GetSubConcepts(lv0SubjectTitle)
		if len(subConceptList) == 0 {
			log.Warn().Any("lv0SubjectTitle", lv0SubjectTitle).Msg("subConceptList is empty")
			continue
		}
		for _, subConecpt := range subConceptList {
			if subConecpt.Level == 1 {
				conceptLv0toLv1[lv0SubjectTitle] = append(conceptLv0toLv1[lv0SubjectTitle], subConecpt.DisplayName)
				conceptLv1toLv0[subConecpt.DisplayName] = append(conceptLv1toLv0[subConecpt.DisplayName], lv0SubjectTitle)
			}
		}
	}

	// 获得各个二级学科的三级学科
	conceptLv1toLv2 := make(map[string][]string)
	conceptLv2toLv1 := make(map[string][]string)
	conceptLv2toLv0 := make(map[string][]string)
	for lv1SubjectTitle, lv0TitleList := range conceptLv1toLv0 {
		subConceptList := mongoClient.GetSubConcepts(lv1SubjectTitle)
		if len(subConceptList) == 0 {
			log.Warn().Any("lv1SubjectTitle", lv1SubjectTitle).Msg("subConceptList is empty")
			continue
		}
		for _, subConecpt := range subConceptList {
			if subConecpt.Level == 2 {
				conceptLv1toLv2[lv1SubjectTitle] = append(conceptLv1toLv2[lv1SubjectTitle], subConecpt.DisplayName)
				conceptLv2toLv1[subConecpt.DisplayName] = append(conceptLv2toLv1[subConecpt.DisplayName], lv1SubjectTitle)
				conceptLv2toLv0[subConecpt.DisplayName] = append(conceptLv2toLv0[subConecpt.DisplayName], lv0TitleList...)

			}
		}
	}

	log.Info().Any("conceptLv2toLv1 size", len(conceptLv2toLv1)).Msg("concept lv2 size")

	nisDataPathList := []string{
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2008_2009.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2018_2019.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2022_2023.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_1950_2000.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2016_2017.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2020_2021.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2024_2025.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2000_2005.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2006_2007.txt",
		"/mnt/sata3/openalex_20241031/google_distance_delete_noref_2010_2015.txt",
	}
	pool, _ := ants.NewPool(20)
	defer pool.Release()
	wg := sync.WaitGroup{}

	for _, filePath := range nisDataPathList {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()

			title_id_series := make(map[string]int64)
			var local_title_id int64

			// 存储分年度的计算器
			computerMap := make(map[workConfig]*distanceComplexity.DistanceGraph)

			// 存储分年度的 nodeID
			nodeTitleMap := make(map[workConfig]*hashset.Set[string])

			for item := range getNisrecord(filePath) {
				concept1Lv0ConceptList := conceptLv2toLv0[item.Concept1]
				concept2Lv0ConceptList := conceptLv2toLv0[item.Concept2]
				for _, conceptLv0Title := range Intersection(concept1Lv0ConceptList, concept2Lv0ConceptList) {
					workKey := workConfig{
						Year:    item.Year,
						Concept: conceptLv0Title,
					}

					if _, ok := computerMap[workKey]; !ok {
						computerMap[workKey] = distanceComplexity.NewDistanceGraph()
						nodeTitleMap[workKey] = hashset.New[string]()
					}
					// map id store
					if _, ok := title_id_series[item.Concept1]; !ok {
						title_id_series[item.Concept1] = local_title_id
						local_title_id += 1
					}
					if _, ok := title_id_series[item.Concept2]; !ok {
						title_id_series[item.Concept2] = local_title_id
						local_title_id += 1
					}
					computerMap[workKey].SetEdge(title_id_series[item.Concept1],
						title_id_series[item.Concept2],
						item.Distance,
					)
					nodeTitleMap[workKey].Add(item.Concept1, item.Concept2)
				}
			}

			for workKey, computer := range computerMap {

				for _, nodeLv2Title := range nodeTitleMap[workKey].Values() {

					nodeCategory := []string{}
					for _, lv1Title := range conceptLv2toLv1[nodeLv2Title] {
						if Contains(conceptLv0toLv1[workKey.Concept], lv1Title) {
							nodeCategory = append(nodeCategory, lv1Title)
						}
					}

					if len(nodeCategory) == 0 {
						log.Warn().Any("nodeLv2Title", nodeLv2Title).Msg("nodeCategory not found")
					}
					computer.SetNodeCategory(title_id_series[nodeLv2Title],
						nodeCategory,
					)
				}

				complexityVal := computer.ProgressDistanceComplexity()
				log.Info().Any("len", len(computer.NodesMap)).Any("workKey", workKey).Float64("BigDegreeEntropy", complexityVal.BigComplexity).Float64("LittleStructuralEntropy", complexityVal.LittlComplexity).Msg("graph entropy complete")
				mongoClient.InsertTopDisciplineDistanceComplexity(workKey.Year, workKey.Concept, complexityVal)
			}
		})

	}
	wg.Wait()
}
