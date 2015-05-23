package twitter

import (
	"github.com/ChimeraCoder/anaconda"
	"github.com/hugbotme/hug-go/config"
	"log"
	"net/url"
	"sync"
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
	var sinceID string
	var sinceIDSet bool
	sinceID = ""

	var mutex = &sync.Mutex{}

	for {
		sinceIDSet = false

		v := url.Values{}
		if len(sinceID) > 0 {
			v.Set("since_id", sinceID)
		}

		mentions, err := client.API.GetMentionsTimeline(v)
		if err != nil {
			log.Printf("Twitter API GetMentionsTimeline-Error: %s", err)
			// Set API Request throttling, because of the twitter API Rate limit
			// Currently 15 requests for a 15 minute window are allowed
			// An alternative would be API throttling like api.SetDelay(60 * time.Second)
			time.Sleep(60 * time.Second)
			continue
		}

		for _, mention := range mentions {
			if sinceIDSet == false {
				sinceID = mention.IdStr
				sinceIDSet = true

				// Update last tweet Check
				now := time.Now()
				mutex.Lock()
				lastTweet = &now
				mutex.Unlock()
			}

			for _, link := range mention.Entities.Urls {
				toHug := Hug{
					TweetID: mention.IdStr,
					URL:     link.Expanded_url,
				}

				hugs <- toHug
			}
		}

		// Set API Request throttling, because of the twitter API Rate limit
		// Currently 15 requests for a 15 minute window are allowed
		time.Sleep(60 * time.Second)
	}
}
