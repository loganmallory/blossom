package model

import "github.com/rs/zerolog"

type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (error Error) MarshalZerologObject(e *zerolog.Event) {
	e.Int("code", error.Code).
		Str("msg", error.Msg)
}
