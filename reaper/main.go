package main

import (
	"flag"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"github.com/guardian/mediaflipper/webapp/jobrunner"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"log"
	"time"
)

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

/**
delete all kubernetes job objects associated with the given mediaflipper job
*/
func DeleteK8Job(forId uuid.UUID, jobClient v1.JobInterface, dryRun bool) error {
	matchingJobs, err := jobrunner.FindRunnerFor(forId, jobClient)
	if err != nil {
		log.Printf("ERROR: Could not look up job containers for %s: %s", forId.String(), err)
		return err
	}

	var dryRunValue []string
	if dryRun {
		dryRunValue = []string{"All"}
	} else {
		dryRunValue = nil
	}

	for _, k8job := range *matchingJobs {
		log.Printf("Found %s...", k8job.Name)
		if k8job.Status == models.CONTAINER_ACTIVE {
			log.Printf("%s seems to still be active, not removing it.", k8job.Name)
		} else {
			err := jobClient.Delete(k8job.Name, &metav1.DeleteOptions{
				DryRun: dryRunValue,
			})
			if err != nil {
				log.Printf("ERROR: Could not delete k8 job %s for mediaflipper job %s: %s", k8job.Name, forId.String(), err)
				//not a fatal error
			}
		}
	}
	return nil
}

func ProcessJob(job *models.JobContainer, cutoffTime time.Time, dryRun bool, jobClient v1.JobInterface) error {
	if job.EndTime != nil && job.EndTime.Before(cutoffTime) {
		log.Printf("Removing old job with id %s", job.Id)
		err := DeleteK8Job(job.Id, jobClient, dryRun)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	maxAgeHours := flag.Int64("maxage", 36, "delete jobs and files that have been present for longer than this many hours")
	pageSize := flag.Int64("pagesize", 100, "pull this many jobs from the database at once")
	dryRun := flag.Bool("dryrun", true, "don't actually delete anything")
	kubeConfigPath := flag.String("kubeconfig", "", ".kubeconfig file for running out of cluster. If not specified then in-cluster initialisation will be tried")

	flag.Parse()

	log.Printf("Reading config from serverconfig.yaml")
	config, configReadErr := helpers.ReadConfig("config/serverconfig.yaml")
	log.Print("Done.")

	if configReadErr != nil {
		log.Fatal("No configuration, can't continue")
	}

	log.Printf("Dryrun is %t", *dryRun)
	redisClient, redisErr := SetupRedis(config)
	if redisErr != nil {
		log.Fatal("Could not connect to redis")
	}

	k8Client, _ := GetK8Client(kubeConfigPath)

	startTime := time.Now()

	log.Printf("Reaping of old data starting at %s", startTime)

	jobClient, cliErr := jobrunner.GetJobClient(k8Client)
	if cliErr != nil {
		log.Fatalf("Could not get job client: %s", jobClient)
	}

	cutoffTime := time.Now().Add(-time.Duration(*maxAgeHours) * time.Hour)
	log.Printf("Cutoff time is %s", cutoffTime)

	var cursor uint64 = 0
	for {
		jobs, nextCursor, err := models.ListJobContainers(cursor, *pageSize, redisClient, models.SORT_CTIME_OLDEST)

		if err != nil {
			log.Fatalf("ERROR: Could not retrieve page of jobs: %s", err)
		}

		for _, j := range *jobs {
			procErr := ProcessJob(&j, cutoffTime, *dryRun, jobClient)
			if procErr != nil {
				log.Fatal(procErr)
			}
		}

		if nextCursor == 0 {
			break
		} else {
			cursor = nextCursor
		}
	}

	endTime := time.Now()

	log.Printf("Reaping run completed at %s and took %d seconds", endTime, endTime.Unix()-startTime.Unix())

}
