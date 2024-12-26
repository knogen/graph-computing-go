# graph-computing-go
统计计算 Wikipedia & OpenAlex 的点边数量, 网络熵(度分布熵, 结构熵)


## wtf_wikipedia_server 

gprc server
set env value to WikiTextParserGrpcUrl
the project is on `https://github.com/ider-zh/wtf_wikipedia_server`

## extract openalex to mongodb
the project is on `https://github.com/knogen/openalex-load-go`

# 从 wikipedia xml dump 中提取数据存储到 mongodb

`make wiki-extract`

这个服务需要 [wtf_wikipedia_server](https://github.com/ider-zh/wtf_wikipedia_server), wtf_wikipedia_server 提供将 wikiText 提取 pagelinks, categories 等服务. 
wiki-extract 会 扫描所有的 history xml 文件, 找到每个 page 接近年末的 revison, 解析其 wikitext, 并且标记 year_tag, 以便能够找到每一年的 snapshop.
扫描的时候只关注 ns 为 0,14 的 page. 后续还可以根据 categories 找到学科.

数据存储到 mongodb, 可能需要手动创建一些索引加速.

# 计算 Wikipedia 网络的度数分布
`go run main.go wikiDegreeStats`

会找到从 2004 到 2024 年的 snapshot, 计算每个 snapshot 的度数分布, 并存储到 mongodb.

# 计算 Wikipedia entropy 

## 计算 Wikipedia 网络子图的熵
`go run main.go wikiEntropy -t total`

 会找到从 2004 到 2024 年的 snapshot,按照 page 的入度排序, 取其中的 top [10,20,40,60,80,100]%节点, 生成子图, 计算每个 snapshot 的度分布熵, 结构熵, 并存储到 mongodb.

## 标记 Wikipedia core 学科

lab/wikipedia_subject_entropy.ipynb
通过预定义的学科和组合, 找到每年 snapshop 中 core 学科, 没有 core 学科的使用原始学科, 计划使用 1,2,3 层不同的学科层数, 完成之后发现只有3层的学科其 paage 才达到数百到数千, 后面的学科规模, 熵值计算都是使用学科3层子图. 最后将其标记到 数据库原始记录中, 方便下面的学科计算


## 计算 Wikipedia 学科网络的熵
`go run main.go wikiEntropy -t subject`

对学科进行熵值计算, 排除了 Art 学科, 计算逻辑同前的 wikipedia 计算逻辑.

# 统计 OpenAlex 网络的度数分布

`go run main.go openalexDegreeStats`

统计从 1940 年到 2024 年, 每年子图的网络节点度数分布
经过会议多次讨论, 后面的网络子图计算使用 In-Degree>=2 的子图计算

# 计算 openalex entropy

## 计算 OpenAlex 网络子图的熵
`go run main.go oae -t total`
计算 OpenAlex 网络的熵值, 先过滤一遍网络, 只保留 In-Degree >= 2 的节点, 然后按年过滤子网,年份取 1940~2024 再将网络节点按照入度排序, 这里计算了2中入度排序, 一种是处理好的当前子图的入度排序, 这种切分后的子网标记为 rankType=current, 另一种是处理前的OpenAlex的入度排序, 标记为 rankType=total. 后面统计的时候使用的是 rankType=current 的子图.
网络按照入度排序后, 取其中的 top [10,20,40,60,80,100]%节点, 生成子图, 计算每个 snapshot 的度分布熵, 结构熵, 并存储到 mongodb.

## 计算 OpenAlex 学科子图的熵
`go run main.go oae -t subject`

通过再 OpenAlex 导入项目中设定的 Concepts_lv0 字段, 能够找到学科, 再按照网络子图的计算逻辑, 计算出每个 snapshot 的度分布熵, 结构熵, 并存储到 mongodb.


# 统计代码

`lab/stats.ipynb`
将Wikipedia, OpenAlex 网络统计信息整理到 Excel 表格, 并添加折线图.

`lab/entropy.ipynb`
统计 Wikipedia, OpenAlex 度分布熵整理到 Excel 表格, 并添加折线图.
