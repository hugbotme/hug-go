package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
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

func connectRedis(config config.RedisConfiguration) redis.Conn {
	redisClient, err := redis.Dial("tcp", config.Url)
	if err != nil {
		log.Fatal("Redis client init (connect) failed:", err)
	}

	if len(config.Auth) == 0 {
		return redisClient
	}

	if _, err := redisClient.Do("AUTH", config.Auth); err != nil {
		redisClient.Close()
		log.Fatal("Redis client init (auth) failed:", err)
	}
	return redisClient
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
	// TODO extract redis credentials to config

	redisClient := connectRedis(config.Redis)
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
		status, err := ProcessURL(githubClient, redisClient, hug)
		if err != nil {
			log.Println(err)
		}

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
