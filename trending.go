package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/google/go-github/github"
	"github.com/hugbotme/hug-go/twitter"
	"log"
	"time"
)

func QueueTrendingRepositoryWhenIamBored(hugs chan twitter.Hug, lastTweet *time.Time, gh *github.Client, red redis.Conn) {
	timeToTick := time.Duration(15) * time.Minute
	for {
		ticker := time.NewTicker(timeToTick)
		<-ticker.C

		if time.Since(*lastTweet) < timeToTick {
			timeToTick = timeToTick - time.Since(*lastTweet)
		} else {
			timeToTick = time.Duration(15) * time.Minute
			QueueTrendingTopic(hugs, gh, red)
		}
	}
}

func QueueTrendingTopic(hugs chan twitter.Hug, gh *github.Client, redisClient redis.Conn) {
	url, err := redis.String(redisClient.Do("SPOP", "hug:bored-urls"))
	if err != nil {
		log.Println(err)
	}

	toHug := twitter.Hug{
		TweetID: "",
		URL:     url,
	}

	hugs <- toHug
}
