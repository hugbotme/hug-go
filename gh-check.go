package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/google/go-github/github"
	"github.com/hugbotme/hug-go/twitter"
	"golang.org/x/oauth2"
	netUrl "net/url"
	"strings"
)

type GitHubUrl struct {
	Url        *netUrl.URL
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

func ParseGitHubUrl(rawurl string) (*GitHubUrl, error) {
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

	return &GitHubUrl{
		Url:        parsed,
		Owner:      owner,
		Repository: repository,
	}, nil
}

func GitHubHasReadme(client *github.Client, url *GitHubUrl) (bool, error) {
	owner := url.Owner
	repo := url.Repository

	content, resp, err := client.Repositories.GetReadme(owner, repo, nil)

	if err != nil {
		fmt.Println(resp, err)
		return false, err
	}

	_, err = base64.StdEncoding.DecodeString(*content.Content)
	if err != nil {
		return false, err
	}

	return true, nil
}

func CanonicalUrl(url *GitHubUrl) string {
	return "http://github.com/" + url.Owner + "/" + url.Repository
}

func GitHubRepoAllowed(client redis.Conn, url *GitHubUrl) bool {
	b, err := redis.Bool(client.Do("EXIST", "blacklist:"+CanonicalUrl(url)))
	if err != nil {
		return true
	}
	return !b
}

func AddToBlacklist(client redis.Conn, url *GitHubUrl) error {
	_, err := client.Do("SET", "blacklist:"+CanonicalUrl(url), 1, "EX", 7*24*60*60)
	return err
}

func AddToProcess(client redis.Conn, url *GitHubUrl) error {
	_, err := client.Do("RPUSH", "hugbot:urls", CanonicalUrl(url))
	return err
}

func GitHubClient(access_token string) *github.Client {
	ts := &tokenSource{
		token: &oauth2.Token{AccessToken: access_token},
	}

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return github.NewClient(tc)
}

func ProcessUrl(gh *github.Client, red redis.Conn, hug twitter.Hug) error {
	parsed, _ := ParseGitHubUrl(hug.Url)

	has, err := GitHubHasReadme(gh, parsed)
	if err != nil {
		return err
	}

	allowed := GitHubRepoAllowed(red, parsed)

	if has && allowed {
		AddToBlacklist(red, parsed)
	}

	return nil
}
