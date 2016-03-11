package main

import (
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

// Token implements interface for oauth2.
type Token struct {
	AccessToken string
}

// Token implements interface for oauth2.
func (t *Token) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func newDOClient(token string) *godo.Client {
	t := &Token{AccessToken: token}
	oa := oauth2.NewClient(oauth2.NoContext, t)
	return godo.NewClient(oa)
}
