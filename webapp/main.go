package main

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/analysis"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/initiator"
	"github.com/guardian/mediaflipper/webapp/jobrunner"
	"github.com/guardian/mediaflipper/webapp/jobs"
	"log"
	"net/http"
)

type MyHttpApp struct {
	index       IndexHandler
	healthcheck HealthcheckHandler
	static      StaticFilesHandler
	initiators  initiator.InitiatorEndpoints
	jobs        jobs.JobsEndpoints
	analysers   analysis.AnalysisEndpoints
}

func SetupRedis(config *helpers.Config) (*redis.Client, error) {
	log.Printf("Connecting to Redis on %s", config.Redis.Address)
	client := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: config.Redis.Password,
		DB:       config.Redis.DBNum,
	})

	_, err := client.Ping().Result()
	if err != nil {
		log.Printf("Could not contact Redis: %s", err)
		return nil, err
	}
	log.Printf("Done.")
	return client, nil
}

func main() {
	var app MyHttpApp

	/*
		read in config and establish connection to persistence layer
	*/
	log.Printf("Reading config from serverconfig.yaml")
	config, configReadErr := helpers.ReadConfig("config/serverconfig.yaml")
	log.Print("Done.")

	if configReadErr != nil {
		log.Fatal("No configuration, can't continue")
	}

	redisClient, redisErr := SetupRedis(config)
	if redisErr != nil {
		log.Fatal("Could not connect to redis")
	}

	k8Client, _ := jobrunner.InClusterClient()

	log.Print("Got k8client: ", k8Client)
	app.index.filePath = "public/index.html"
	app.index.contentType = "text/html"
	app.index.exactMatchPath = "/"
	app.healthcheck.redisClient = redisClient
	app.static.basePath = "public"
	app.static.uriTrim = 2
	app.initiators = initiator.NewInitiatorEndpoints(config, redisClient)
	app.jobs = jobs.NewJobsEndpoints(redisClient)
	app.analysers = analysis.NewAnalysisEndpoints(redisClient)

	http.Handle("/default", http.NotFoundHandler())
	http.Handle("/", app.index)
	http.Handle("/healthcheck", app.healthcheck)
	http.Handle("/static/", app.static)

	app.initiators.WireUp("/api/flip")
	app.jobs.WireUp("/api/job")
	app.analysers.WireUp("/api/analysis")

	log.Printf("Starting server on port 9000")
	startServerErr := http.ListenAndServe(":9000", nil)

	if startServerErr != nil {
		log.Fatal(startServerErr)
	}
}
