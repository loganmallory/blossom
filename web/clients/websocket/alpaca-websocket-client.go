package alpacawebsocketclient

//https://github.com/gorilla/websocket/blob/master/examples/echo/client.go

import (
	"blossom/model"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"time"
)

const alpacaWebsocketUrl = "wss://stream.data.alpaca.markets/v2/sip"

var logger zerolog.Logger
var connLogger zerolog.Logger
var authLogger zerolog.Logger
var discLogger zerolog.Logger
var scribeLogger zerolog.Logger
var decoLogger zerolog.Logger

var connection *websocket.Conn
var Subscriptions chan model.Subscription
var Trades chan model.Trade
var Quotes chan model.Quote
var Errata chan model.Error
var Faults chan error

func init() {
	logger       = log.With().Str("component", "AlpacaWebsocketClient").Str("url", alpacaWebsocketUrl).Logger()
	connLogger   = logger.With().Str("phase", "EstablishConnection()").Logger()
	authLogger   = logger.With().Str("phase", "Authenticate()").Logger()
	discLogger   = logger.With().Str("phase", "Disconnect()").Logger()
	scribeLogger = logger.With().Str("phase", "Scribe()").Logger()
	decoLogger   = logger.With().Str("phase", "DecodeMessage()").Logger()

	Subscriptions = make(chan model.Subscription, 128)
	Trades        = make(chan model.Trade, 1024)
	Quotes        = make(chan model.Quote, 4096)
	Errata        = make(chan model.Error, 128)
	Faults        = make(chan error, 128)
}

func EstablishConnection() error {
	connLogger.Info().Msg("establishing websocket connection")

	//establish connection
	conn, _, err := websocket.DefaultDialer.Dial(alpacaWebsocketUrl, nil)
	if err != nil {
		connLogger.Error().Err(err).Msg("failed to dial websocket")
		return err
	}

	//assign the connection
	connection = conn

	//confirm success message from Alpaca
	if _, messageBytes, err := connection.ReadMessage(); err == nil {
		message := string(messageBytes)
		if message != "[{\"T\":\"success\",\"msg\":\"connected\"}]" {
			err = errors.New("failed to verify connection: " + message)
			connLogger.Error().Err(err).Msg("failed to verify connection")
			return err
		}
		connLogger.Info().Msg("successfully established websocket connection")
		return nil
	}
	connLogger.Error().Err(err).Msg("failed to read connection response message")
	return err
}

func Authenticate(apiKey string, apiSecret string) (bool, error) { //return true if bad credentials (user error)
	authLogger.Info().Msg("authenticating websocket connection")

	//send auth message
	authenticationMessage := fmt.Sprintf("{\"action\": \"auth\", \"key\": \"%s\", \"secret\": \"%s\"}", apiKey, apiSecret)
	err := connection.WriteMessage(websocket.TextMessage, []byte(authenticationMessage))
	if err != nil {
		authLogger.Error().Err(err).Msg("failed to send authentication message")
		return false, err
	}

	//confirm success message from Alpaca
	if _, messageBytes, err := connection.ReadMessage(); err == nil {
		message := string(messageBytes)
		if message != "[{\"T\":\"success\",\"msg\":\"authenticated\"}]" {
			err = errors.New("failed to verify authentication: " + message)
			authLogger.Error().Err(err).Msg("failed to verify authentication")
			return true, err
		}
		authLogger.Info().Msg("successfully authenticated websocket connection")
		return false, nil
	}
	authLogger.Error().Err(err).Msg("failed to read authentication response message")
	return false, err
}

func Disconnect() {
	discLogger.Info().Msg("closing websocket connection")

	//try to send closure message
	for i := 1; i <= 10; i++ {
		err := connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err == nil {
			break
		}
		discLogger.Error().Err(err).Msgf("failed to send closure message - attempt %d out of 10")
		time.Sleep(time.Second)
	}

	//wait for closure confirmation from remote
	select {
	case err := <- Faults:
		if "websocket: close 1000 (normal)" == err.Error() {
			discLogger.Info().Msg("remote confirmed disconnect")
			return
		}
	case <-time.After(time.Second * 10):
		discLogger.Warn().Int("waitedSec", 10).Msg("remote did not confirm disconnect")
		return
	}
}

func scribe(action string, tradeSymbols []string, quoteSymbols []string) {
	scribeLogger.Info().Strs("tradeSymbols", tradeSymbols).Strs("quoteSymbols", quoteSymbols).Msg(action)

	const messageTemplate = "{\"action\": \"%s\", \"trades\": %s, \"quotes\": %s, \"bars\": []}"

	//create json list strings
	tradeSymbolsBytes, errT := json.Marshal(tradeSymbols)
	if errT != nil {
		scribeLogger.Error().Err(errT).Strs("tradeSymbols", tradeSymbols).Msg("failed to convert trade symbols to json")
	}
	quoteSymbolsBytes, errQ := json.Marshal(quoteSymbols)
	if errQ != nil {
		scribeLogger.Error().Err(errQ).Strs("quoteSymbols", quoteSymbols).Msg("failed to convert quote symbols to json")
	}

	//if at least trade or quotes marshal succeed, send message
	var message string
	if errT == nil && errQ == nil { //if both succeeded
		message = fmt.Sprintf(messageTemplate, action, string(tradeSymbolsBytes), string(quoteSymbolsBytes))
	} else if errT == nil { //if trades succeeded
		message = fmt.Sprintf(messageTemplate, action, string(tradeSymbolsBytes), "[]")
	} else if errQ == nil { //if quotes succeeded
		message = fmt.Sprintf(messageTemplate, action, "[]", string(quoteSymbolsBytes))
	} else { //neither succeeded
		return
	}
	err := connection.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		scribeLogger.Error().Err(err).Bytes("bytes", []byte(message)).Msgf("failed to send %s message", action)
	}
}

func Subscribe(tradeSymbols []string, quoteSymbols []string) {
	scribe("subscribe", tradeSymbols[:], quoteSymbols[:])
}

func Unsubscribe(tradeSymbols []string, quoteSymbols []string) {
	scribe("unsubscribe", tradeSymbols[:], quoteSymbols[:])
}

func Listen() {
	logger.Info().Str("phase", "Listen()").Msg("started listening for websocket messages")
	//infinite loop
	for {
		_, messageBytes, err := connection.ReadMessage()
		if err != nil {
			Faults <- err
			return //once an error is returned, all future messages will also be errors
		}
		//async decode message
		go decodeMessage(messageBytes)
	}
}

func decodeMessage(messageBytes []byte) {
	//get list of raw messages -- https://stackoverflow.com/questions/42721732/is-there-a-way-to-have-json-unmarshal-select-struct-type-based-on-type-prope
	var rawEventBytes []json.RawMessage
	err := json.Unmarshal(messageBytes, &rawEventBytes)
	if err != nil {
		decoLogger.Error().Err(err).Bytes("bytes", messageBytes).Str("target", "[]json.RawMessage").Msg("failed to unmarshal message bytes")
		return
	}

	//for each event
	for _, rawBytes := range rawEventBytes {

		//convert to generic map
		var obj map[string]interface{}
		err = json.Unmarshal(rawBytes, &obj)
		if err != nil {
			decoLogger.Error().Err(err).Bytes("bytes", rawBytes).Str("target", "map[string]interface{}").Msg("failed to unmarshal message bytes")
			continue
		}

		//get event type
		eventType, castOk := obj["T"].(string)
		if !castOk {
			decoLogger.Error().Bytes("bytes", rawBytes).Msg("failed to get event type")
			continue
		}

		//unmarshal into event type
		switch eventType {
		case "subscription":
			event := &model.Subscription{}
			if err = json.Unmarshal(rawBytes, event); err != nil {
				decoLogger.Error().Err(err).Bytes("bytes", rawBytes).Str("target", "Subscription").Msg("failed to unmarshal message bytes")
				continue
			}
			Subscriptions <- *event
		case "t":
			event := &model.Trade{}
			if err = json.Unmarshal(rawBytes, event); err != nil {
				decoLogger.Error().Err(err).Bytes("bytes", rawBytes).Str("target", "Trade").Msg("failed to unmarshal message bytes")
				continue
			}
			Trades <- *event
		case "q":
			event := &model.Quote{}
			if err = json.Unmarshal(rawBytes, event); err != nil {
				decoLogger.Error().Err(err).Bytes("bytes", rawBytes).Str("target", "Quote").Msg("failed to unmarshal message bytes")
				continue
			}
			Quotes <- *event
		case "error":
			event := &model.Error{}
			if err = json.Unmarshal(rawBytes, event); err != nil {
				decoLogger.Error().Err(err).Bytes("bytes", rawBytes).Str("target", "Error").Msg("failed to unmarshal message bytes")
				continue
			}
			Errata <- *event
		default:
			decoLogger.Error().Str("eventType", eventType).Msg("unrecognized event type")
		}
	}
}
