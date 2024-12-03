# project 1
project 1 计算 Wikipedia & OpenAlex Graph entropy


# mind storm

1. extrat wikipedia snapshot from history dump

extract snapshot from dump and parse to successful collection
fail item to fail collection
任务比较大, 允许断点重试

# 其他代码

## wtf_wikipedia_server 

gprc server
set env value to WikiTextParserGrpcUrl
the project is on `https://github.com/ider-zh/wtf_wikipedia_server`

## extract openalex to mongodb
the project is on `https://github.com/knogen/openalex-load-go`

# 从 wikipedia xml dump 中提取数据存储到 mongodb

`make wiki-extract`

# 计算 Wikipedia entropy 
`make wiki-entropy`

# 计算 openalex entropy
`make openalex-entropy`