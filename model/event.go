package model

//https://alpaca.markets/docs/api-documentation/api-v2/market-data/alpaca-data-api-v2/real-time/
//event types are: success | error | subscription | t | q | b
//success / error / subscription will come in singleton list
//trades, quotes, bars will come in mixed list
//Example of ONE message:
// [{"T":"t","i":96921,"S":"AAPL","x":"D","p":126.55,"s":1,"t":"2021-02-22T15:51:44.208Z","c":["@","I"],"z":"C"},
//  {"T":"q","S":"AMD","bx":"U","bp":87.66,"bs":1,"ax":"X","ap":87.67,"as":1,"t":"2021-02-22T15:51:45.3355677Z","c":["R"],"z":"C"},
//  {"T":"b","S":"SPY","o":388.985,"h":389.13,"l":388.975,"c":389.12,"v":49378,"t":"2021-02-22T19:15:00Z"}]

import (
	"github.com/rs/zerolog"
)

type Event interface {
	MarshalZerologObject(e *zerolog.Event)
}
