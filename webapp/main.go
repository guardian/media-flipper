package main

import (
	"flag"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/analysis"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/initiator"
	"github.com/guardian/mediaflipper/webapp/jobrunner"
	"github.com/guardian/mediaflipper/webapp/jobs"
	"github.com/guardian/mediaflipper/webapp/jobtemplate"
	"github.com/guardian/mediaflipper/webapp/models"
	"github.com/guardian/mediaflipper/webapp/thumbnail"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
)

type MyHttpApp struct {
	index       IndexHandler
	healthcheck HealthcheckHandler
	static      StaticFilesHandler
	templates   jobtemplate.TemplateEndpoints
	initiators  initiator.InitiatorEndpoints
	jobs        jobs.JobsEndpoints
	analysers   analysis.AnalysisEndpoints
	thumbnails  thumbnail.ThumbnailEndpoints
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

func GetK8Client(kubeConfigPath *string) (*kubernetes.Clientset, error) {
	var k8Client *kubernetes.Clientset
	var cliErr error

	if kubeConfigPath == nil || *kubeConfigPath == "" {
		k8Client, cliErr = jobrunner.InClusterClient()
	} else {
		k8Client, cliErr = jobrunner.OutOfClusterClient(*kubeConfigPath)
	}

	if cliErr != nil {
		log.Printf("ERROR: Can't establish communication with Kubernetes. Job-running functionality won't work.")
		return nil, cliErr
	} else {
		log.Print("Got k8client.")
	}
	return k8Client, nil
}

func main() {
	var app MyHttpApp

	kubeConfigPath := flag.String("kubeconfig", "", ".kubeconfig file for running out of cluster. If not specified then in-cluster initialisation will be tried")
	flag.Parse()

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

	k8Client, _ := GetK8Client(kubeConfigPath)

	templateMgr, mgrLoadErr := models.NewJobTemplateManager("config/standardjobtemplate.yaml")

	if mgrLoadErr != nil {
		log.Printf("Could not initialise template manager: %s", mgrLoadErr)
	}

	runner := jobrunner.NewJobRunner(redisClient, k8Client, templateMgr, 10, 2)

	app.index.filePath = "static/index.html"
	app.index.contentType = "text/html"
	app.healthcheck.redisClient = redisClient
	app.static.basePath = "static"
	app.static.uriTrim = 2
	app.initiators = initiator.NewInitiatorEndpoints(config, redisClient, &runner)
	app.jobs = jobs.NewJobsEndpoints(redisClient, k8Client, templateMgr)
	app.analysers = analysis.NewAnalysisEndpoints(redisClient)
	app.templates = jobtemplate.NewTemplateEndpoints(templateMgr)
	app.thumbnails = thumbnail.NewThumbnailEndpoints(redisClient)

	http.Handle("/", app.index)
	http.Handle("/healthcheck", app.healthcheck)
	http.Handle("/static/", app.static)

	app.initiators.WireUp("/api/flip")
	app.jobs.WireUp("/api/job")
	app.analysers.WireUp("/api/analysis")
	app.templates.WireUp("/api/jobtemplate")
	app.thumbnails.WireUp("/api/thumbnail")

	log.Printf("Starting server on port 9000")
	startServerErr := http.ListenAndServe(":9000", nil)

	if startServerErr != nil {
		log.Fatal(startServerErr)
	}
}
