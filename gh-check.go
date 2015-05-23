package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	netUrl "net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"

	"github.com/garyburd/redigo/redis"
	"github.com/google/go-github/github"
)

type GitHubUrl struct {
	Url        *netUrl.URL
	Owner      string
	Repository string
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

	content, _, err := client.Repositories.GetReadme(owner, repo, nil)

	if err != nil {
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
	_, err := client.Do("SET", "blacklist:"+CanonicalUrl(url), 1)
	return err
}

func AddToProcess(client redis.Conn, url *GitHubUrl) error {
	_, err := client.Do("RPUSH", "hugbot:urls", CanonicalUrl(url))
	return err
}

func main() {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "efab2f1d9053080aac507ddab10272efea804398"},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [github url]\n", os.Args[0])
		return
	}

	url := os.Args[1]
	if url == "" {
		fmt.Printf("Usage: %s [github url]\n", os.Args[0])
		return
	}

	parsed, _ := ParseGitHubUrl(url)

	fmt.Printf("owner: '%s'\nrepo: '%s'\n", parsed.Owner, parsed.Repository)

	client := github.NewClient(tc)

	has, _ := GitHubHasReadme(client, parsed)
	fmt.Println("has readme", has)

	c, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println("error", err)
		return
	}
	defer c.Close()

	allowed := GitHubRepoAllowed(c, parsed)
	fmt.Println("is allowed", allowed)
}
