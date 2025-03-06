package openalexentropy

import (
	"bufio"
	"fmt"
	"graph-computing-go/internal/distanceComplexity"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

// Record 定义一个结构体来存储提取的数据
type nisRecord struct {
	Year     int
	Concept1 string
	Concept2 string
	Distance float64
}

// IsFloatZero 判断浮点数是否近似等于 0
func IsFloatZero(num float64, precision float64) bool {
	return math.Abs(num) < precision
}

func getNisrecord(filePath string) chan *nisRecord {

	// 创建一个切片来存储提取的记录
	var recordsChan = make(chan *nisRecord, 100)

	go func() {

		// 假设数据存储在一个名为 data.txt 的文件中
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Println("无法打开文件:", err)
			return
		}
		defer file.Close()
		// 使用 bufio.Scanner 逐行读取文件
		scanner := bufio.NewScanner(file)
		lastLine := ""
		edgeCount := 0
		for scanner.Scan() {
			line := scanner.Text()
			// 使用制表符分割每行数据
			fields := strings.Split(line, "\t")
			if len(fields) < 3 {
				// 跳过不完整的行
				log.Warn().Any("line", line).Any("lastLine", lastLine).Any("filePath", filePath).Msg("跳过不完整的行")
				continue
			}
			lastLine = line
			// 提取年份
			year, err := strconv.Atoi(fields[0])
			if err != nil {
				log.Warn().Any("fields", fields).Msg("无法将年份转换为整数")
				fmt.Println("无法将年份转换为整数:", err)
				continue
			}

			// 提取 concept1
			concept1 := fields[1]

			// 提取 concept2 和 distance
			var concept2 string
			var distance float64
			if len(fields) == 4 {
				// 如果有四个字段，正常提取
				concept2 = fields[2]
				distanceStr := fields[3]
				distance, err = strconv.ParseFloat(distanceStr, 64)
				if err != nil {
					log.Warn().Any("fields", fields).Msg("无法将距离转换为浮点数")
					fmt.Println("无法将距离转换为浮点数:", err)
					continue
				}
				if IsFloatZero(distance, 1e-9) {
					// log.Warn().Any("line", line).Msg("distance is 0")
					continue
				}
			} else {
				log.Warn().Any("line", line).Msg("line is not 4 fields")
				continue
			}

			// 创建一个新的记录并添加到切片中
			recordsChan <- &nisRecord{
				Year:     year,
				Concept1: concept1,
				Concept2: concept2,
				Distance: distance,
			}
			edgeCount += 1
		}

		if err := scanner.Err(); err != nil {
			log.Warn().Any("filePath", filePath).Msg("读取文件时出错")
			fmt.Println("读取文件时出错:", err)
		}
		log.Info().Any("filePath", filePath).Int("edgeCount", edgeCount).Msg("filePath")
		close(recordsChan)
	}()
	return recordsChan
}

// 使用学术圈, 和学术圈中每篇文章的距离, 计算改进过的距离复杂度
func Lv2DisciplineDistanceComplexity() {
	subjectList := []string{"Mathematics", "Physics", "Computer science", "Engineering", "Medicine",
		"Biology", "Chemistry", "Materials science", "Geology", "Geography", "Environmental science",
		"Economics", "Sociology", "Psychology", "Political science", "Philosophy", "Business", "Art",
		"History"}

	mongoClient := newMongoDataBase(conf.MongoUrl, conf.OpenAlex_Version)
	defer mongoClient.close()

	concept_tree_map := make(map[string][]string)
	for _, subjectTitle := range subjectList {
		subConceptList := mongoClient.GetSubConcepts(subjectTitle)
		if len(subConceptList) == 0 {
			log.Warn().Any("subjectTitle", subjectTitle).Msg("subConceptList is empty")
			continue
		}
		for _, subConecpt := range subConceptList {
			concept_tree_map[subConecpt.DisplayName] = append(concept_tree_map[subConecpt.DisplayName], subjectTitle)
		}
	}
	log.Info().Any("concept_tree_map", concept_tree_map["Work flow"]).Msg("concept_tree_map")
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

			title_id_convert_map := make(map[string]int64)
			var title_id_series int64

			// 存储分年度的计算器
			computerMap := make(map[int]*distanceComplexity.DistanceGraph)

			// 存储分年度的 nodeID
			nodeTitleMap := make(map[int][]string)

			for item := range getNisrecord(filePath) {

				if _, ok := computerMap[item.Year]; !ok {
					computerMap[item.Year] = distanceComplexity.NewDistanceGraph()
				}
				// map id store
				if _, ok := title_id_convert_map[item.Concept1]; !ok {
					title_id_convert_map[item.Concept1] = title_id_series
					title_id_series += 1
				}
				if _, ok := title_id_convert_map[item.Concept2]; !ok {
					title_id_convert_map[item.Concept2] = title_id_series
					title_id_series += 1
				}
				computerMap[item.Year].SetEdge(title_id_convert_map[item.Concept1],
					title_id_convert_map[item.Concept2],
					item.Distance,
				)
				nodeTitleMap[item.Year] = append(nodeTitleMap[item.Year], item.Concept1, item.Concept2)

			}

			for year, computer := range computerMap {
				for _, nodeTitle := range nodeTitleMap[year] {
					if len(concept_tree_map[nodeTitle]) == 0 {
						log.Warn().Any("nodeTitle", nodeTitle).Msg("concept_tree_map not found")
					}
					computer.SetNodeCategory(title_id_convert_map[nodeTitle],
						concept_tree_map[nodeTitle],
					)
				}

				complexityVal := computer.ProgressDistanceComplexity()
				log.Info().Any("len", len(computer.NodesMap)).Int("year", year).Float64("BigDegreeEntropy", complexityVal.BigComplexity).Float64("LittleStructuralEntropy", complexityVal.LittlComplexity).Msg("graph entropy complete")
				mongoClient.InsertDistanceComplexity(year, complexityVal)
			}
		})

	}
	wg.Wait()
}
