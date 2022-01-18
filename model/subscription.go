package model

import "github.com/rs/zerolog"

type Subscription struct {
	Trades []string `json:"trades"`
	Quotes []string `json:"quotes"`
}

func (subscription Subscription) MarshalZerologObject(e *zerolog.Event) {
	e.Strs("trades", subscription.Trades).
		Strs("quotes", subscription.Quotes)
}
