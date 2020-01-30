package files

import (
	"github.com/go-redis/redis/v7"
	"net/http"
)

type FilesEndpoints struct {
	GetFileHandler     GetFileInfo
	StreamFileHandler  StreamFile
	FilesForJobHandler ListByJob
}

func NewFilesEndpoints(redisClient *redis.Client) FilesEndpoints {
	return FilesEndpoints{
		GetFileHandler:     GetFileInfo{redisClient: redisClient},
		StreamFileHandler:  StreamFile{redisClient: redisClient},
		FilesForJobHandler: ListByJob{redisClient: redisClient},
	}
}

func (e FilesEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/get", e.GetFileHandler)
	http.Handle(baseUrl+"/content", e.StreamFileHandler)
	http.Handle(baseUrl+"/byJob", e.FilesForJobHandler)
}
