package main

import (
	"github.com/go-redis/redis/v7"
	"log"
	"net/http"
)

type HealthcheckHandler struct {
	redisClient *redis.Client
}

func (h HealthcheckHandler) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	_, err := h.redisClient.Ping().Result()

	if err == nil {
		w.WriteHeader(200)
	} else {
		log.Printf("HEALTHCHECK FAILED: %s connecting to Redis", err)
		response := GenericErrorResponse{
			Status: "error",
			Detail: "could not contact redis db",
		}
		WriteJsonContent(response, w, 500)
	}
}
