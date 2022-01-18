package model

import (
	"github.com/rs/zerolog"
)

type Quote struct {
	Symbol          string    `json:"S"`
	AskExchangeCode string    `json:"ax"`
	AskPrice        float64   `json:"ap"`
	AskSize         int64     `json:"as"`
	BidExchangeCode string    `json:"bx"`
	BidPrice        float64   `json:"bp"`
	BidSize         int64     `json:"bs"`
	Timestamp       string    `json:"t"`
	Conditions      []string  `json:"c"`
	Tape            string    `json:"z"`
}

func (quote Quote) MarshalZerologObject(e *zerolog.Event) {
	e.Str("symbol", quote.Symbol).
		Str("askExchangeCode", quote.AskExchangeCode).
		Float64("askPrice", quote.AskPrice).
		Int64("askSize", quote.AskSize).
		Str("bidExchangeCode", quote.BidExchangeCode).
		Float64("bidPrice", quote.BidPrice).
		Int64("bidSize", quote.BidSize).
		Str("timestamp", quote.Timestamp).
		Strs("conditions", quote.Conditions).
		Str("tape", quote.Tape)
}
