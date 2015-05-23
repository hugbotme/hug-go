package twitter

import (
	"github.com/ChimeraCoder/anaconda"
	"github.com/hugbotme/hug-go/config"
	"log"
	"net/url"
	"time"
)

type Twitter struct {
	API *anaconda.TwitterApi
}

type Hug struct {
	TweetID string
	URL     string
}

func NewClient(config *config.Configuration) *Twitter {
	anaconda.SetConsumerKey(config.Twitter.ConsumerKey)
	anaconda.SetConsumerSecret(config.Twitter.ConsumerSecret)
	api := anaconda.NewTwitterApi(config.Twitter.AccessToken, config.Twitter.AccessTokenSecret)

	client := Twitter{
		API: api,
	}

	return &client
}

// TODO Add error handling by error channel
// See for an example http://keighl.com/post/handling-errors-from-go-routines/

// TODO Add support for sinceID
// This is useful if this tool needs a restart
func (client *Twitter) GetMentions(hugs chan Hug, lastTweet *time.Time) {
	v := url.Values{}
	stream := client.API.UserStream(v)
	for event := range stream.C {

		switch t := event.(type) {
		case anaconda.Tweet:
			log.Printf("Twitter stream: New event %T\n", t)
			mention := event.(anaconda.Tweet)
			for _, link := range mention.Entities.Urls {
				log.Printf("Twitter stream: New hug for link %s\n", link.Expanded_url)
				toHug := Hug{
					TweetID: mention.IdStr,
					URL:     link.Expanded_url,
				}

				hugs <- toHug
			}

		default:
			log.Printf("Twitter stream: Unsupported event %T\n", t)
		}
	}
}
