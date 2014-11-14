package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"

	"code.google.com/p/goauth2/oauth"
	"github.com/reusee/gobchest"
)

func init() {
	gob.Register(new(oauth.Token))
}

func NewOAuthTokenCache(client *gobchest.Client, key string) *OAuthTokenCache {
	return &OAuthTokenCache{
		client: client,
		key:    key,
	}
}

type OAuthTokenCache struct {
	client *gobchest.Client
	key    string
}

func (o *OAuthTokenCache) Token() (*oauth.Token, error) {
	s, err := o.client.Get(o.key)
	if err != nil {
		return nil, fmt.Errorf("no token for %s", o.key)
	}
	var token oauth.Token
	err = json.Unmarshal(s.([]byte), &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (o *OAuthTokenCache) PutToken(token *oauth.Token) error {
	s, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return o.client.Set(o.key, s)
}
