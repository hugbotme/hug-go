package main

import (
	"flag"
	"fmt"
	"github.com/hugbotme/hug-go/config"
	"github.com/hugbotme/hug-go/twitter"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	flagConfigFile *string
	flagPidFile    *string
	flagVersion    *bool
)

const (
	majorVersion = 1
	minorVersion = 0
	patchVersion = 0
)

// Init function to define arguments
func init() {
	flagConfigFile = flag.String("config", "", "Configuration file")
	flagPidFile = flag.String("pidfile", "", "Write the process id into a given file")
	flagVersion = flag.Bool("version", false, "Outputs the version number and exits")
}

func main() {
	flag.Parse()

	// Output the version and exit
	if *flagVersion {
		fmt.Printf("hug v%d.%d.%d\n", majorVersion, minorVersion, patchVersion)
		return
	}

	// Check for configuration file
	if len(*flagConfigFile) <= 0 {
		log.Fatal("No configuration file found. Please add the --config parameter")
	}

	// PID-File
	if len(*flagPidFile) > 0 {
		ioutil.WriteFile(*flagPidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	}

	fmt.Println("Hi, i am hugbot. And now i start to hug you.")

	config, err := config.NewConfiguration(flagConfigFile)
	if err != nil {
		log.Fatal("Configuration initialisation failed:", err)
	}

	githubClient := GitHubClient(config.Github.APIToken)

	redisClient := config.ConnectRedis()
	defer redisClient.Close()

	hugs := make(chan twitter.Hug, 50)
	var lastTweet time.Time
	// TODO: We don`t close channel hugs. We should do this.

	client := twitter.NewClient(config)
	go client.GetMentions(hugs, &lastTweet)
	if err != nil {
		log.Fatal("Twitter client GetMentions failed:", err)
	}

	go QueueTrendingRepositoryWhenIamBored(hugs, &lastTweet, githubClient, redisClient)

	for hug := range hugs {
		log.Print("Got a new hug", hug)
		status, err := ProcessURL(githubClient, redisClient, hug)
		if err != nil {
			log.Println("ProcessURL failed", err)
			continue
		}

		log.Println("ProcessURL", status)

		switch status {
		case CheckEverythingIsFine:
			AddToQueue(redisClient, &hug)
		case CheckHasNoReadme:
			// TODO: Tweet
		case CheckIsNotAllowed:
			// TODO: Tweet
		case CheckURLParse:
			// TODO: Tweet
		}
	}
}
