package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/garyburd/redigo/redis"
	"github.com/google/go-github/github"
	"github.com/hugbotme/hug-go/twitter"
	"golang.org/x/oauth2"
	"log"
	netUrl "net/url"
	"strings"
)

const (
	CheckEverythingIsFine = iota
	CheckURLParse
	CheckHasNoReadme
	CheckIsNotAllowed
)

type GitHubURL struct {
	URL        *netUrl.URL
	Owner      string
	Repository string
}

// tokenSource is an oauth2.TokenSource which returns a static access token
type tokenSource struct {
	token *oauth2.Token
}

// Token implements the oauth2.TokenSource interface
func (t *tokenSource) Token() (*oauth2.Token, error) {
	return t.token, nil
}

func ParseGitHubURL(rawurl string) (*GitHubURL, error) {
	parsed, err := netUrl.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	if parsed.Host != "github.com" {
		return nil, errors.New("Not a GitHub URL")
	}

	splitted := strings.Split(parsed.Path, "/")
	owner := splitted[1]
	repository := splitted[2]

	return &GitHubURL{
		URL:        parsed,
		Owner:      owner,
		Repository: repository,
	}, nil
}

func GitHubHasReadme(client *github.Client, url *GitHubURL) (bool, error) {
	owner := url.Owner
	repo := url.Repository

	content, resp, err := client.Repositories.GetReadme(owner, repo, nil)

	if err != nil {
		log.Println(resp, err)
		return false, err
	}

	_, err = base64.StdEncoding.DecodeString(*content.Content)
	if err != nil {
		return false, err
	}

	return true, nil
}

func CanonicalURL(url *GitHubURL) string {
	return "http://github.com/" + url.Owner + "/" + url.Repository
}

func GitHubRepoAllowed(client redis.Conn, url *GitHubURL) bool {
	b, err := redis.Bool(client.Do("EXISTS", "blacklist:"+CanonicalURL(url)))
	if err != nil {
		return true
	}
	return !b
}

func AddToBlacklist(client redis.Conn, url *GitHubURL) error {
	_, err := client.Do("SET", "blacklist:"+CanonicalURL(url), 1, "EX", 7*24*60*60)
	return err
}

func AddToQueue(client redis.Conn, hug *twitter.Hug) error {
	jsonHug, err := json.Marshal(hug)
	if err != nil {
		return err
	}

	_, err = client.Do("RPUSH", "hug:queue", string(jsonHug))
	return err
}

func GitHubClient(accessToken string) *github.Client {
	ts := &tokenSource{
		token: &oauth2.Token{AccessToken: accessToken},
	}

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return github.NewClient(tc)
}

func ProcessURL(gh *github.Client, red redis.Conn, hug twitter.Hug) (int, error) {
	parsed, err := ParseGitHubURL(hug.URL)
	if err != nil {
		return CheckURLParse, err
	}

	has, err := GitHubHasReadme(gh, parsed)
	if err != nil {
		return CheckHasNoReadme, err
	}

	allowed := GitHubRepoAllowed(red, parsed)
	if !allowed {
		return CheckIsNotAllowed, nil
	}

	if has && allowed {
		AddToBlacklist(red, parsed)
	}

	return CheckEverythingIsFine, nil
}
