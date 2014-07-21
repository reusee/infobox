package main

import (
	"errors"

	"code.google.com/p/goauth2/oauth"
)

func (d *Database) TokenCache(key string) *OAuthTokenCache {
	return &OAuthTokenCache{
		db:  d,
		key: key,
	}
}

type OAuthTokenCache struct {
	db  *Database
	key string
}

func (o *OAuthTokenCache) Token() (*oauth.Token, error) {
	token, ok := o.db.OAuthTokens[o.key]
	if !ok {
		return nil, errors.New("no token")
	}
	return token, nil
}

func (o *OAuthTokenCache) PutToken(token *oauth.Token) error {
	o.db.OAuthTokens[o.key] = token
	return nil
}
