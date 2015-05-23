package config

import (
	"encoding/json"
	"io/ioutil"
)

type Configuration struct {
	Twitter twitterConfiguration `json:"twitter"`
	Github  GithubConfiguration  `json:"github"`
	Redis   RedisConfiguration   `json:"redis"`
}

type twitterConfiguration struct {
	ConsumerKey       string `json:"consumer-key"`
	ConsumerSecret    string `json:"consumer-secret"`
	AccessToken       string `json:"access-token"`
	AccessTokenSecret string `json:"access-token-secret"`
}

type GithubConfiguration struct {
	APIToken string `json:"api-token"`
}

type RedisConfiguration struct {
	Url  string `json:"url"`
	Auth string `json:"auth"`
}

func NewConfiguration(configFile *string) (*Configuration, error) {
	fileContent, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	var config Configuration
	err = json.Unmarshal(fileContent, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
