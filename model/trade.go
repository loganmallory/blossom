package model

import (
	"github.com/rs/zerolog"
)

type Trade struct {
	Symbol     string    `json:"S"`
	Id         int64     `json:"i"`
	Exchange   string    `json:"x"`
	Price      float64   `json:"p"`
	Size       int64     `json:"s"`
	Timestamp  string    `json:"t"`
	Conditions []string  `json:"c"`
	Tape       string    `json:"z"`
}

func (trade Trade) MarshalZerologObject(e *zerolog.Event) {
	e.Str("symbol", trade.Symbol).
		Int64("id", trade.Id).
		Str("exchange", trade.Exchange).
		Float64("price", trade.Price).
		Int64("size", trade.Size).
		Str("timestamp", trade.Timestamp).
		Strs("conditions", trade.Conditions).
		Str("tape", trade.Tape)
}
