package web

import (
	alpacaWebsocketClient "blossom/web/clients/websocket"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
)

var logger zerolog.Logger
var eventLogger zerolog.Logger

func init() {
	logFile, _ := os.OpenFile("events.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	logger = log.With().Str("component", "Server").Logger()
	eventLogger = zerolog.New(logFile).With().Logger()
}

func Start() {
	logger.Info().Msg("starting server")
	router := gin.Default()

	api := router.Group("/api/v1")
	{
		api.GET("/hello", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"code": 200, "message": "hello world!"})
		})
		api.POST("/login", login)
	}

	router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(404, gin.H{})
	})

	router.Run(":8080")
}

func login(ctx *gin.Context) {
	logger.Info().Msg("login()")
	type LoginInfo struct {
		Name            string `json:"name"`
		AlpacaApiKey    string `json:"alpacaApiKey"`
		AlpacaApiSecret string `json:"alpacaApiSecret"`
	}
	//get request body
	loginInfo := &LoginInfo{}
	if err := ctx.Bind(loginInfo); err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"status": 400, "message": err.Error()})
		return
	}
	//connect
	if err := alpacaWebsocketClient.EstablishConnection(); err != nil {
		ctx.AbortWithStatusJSON(500, gin.H{"status": 500, "message": err.Error()})
		return
	}
	//authenticate
	badCredentials, err := alpacaWebsocketClient.Authenticate(loginInfo.AlpacaApiKey, loginInfo.AlpacaApiSecret)
	if badCredentials {
		ctx.AbortWithStatusJSON(400, gin.H{"status": 400, "message": err.Error()})
		return
	} else if err != nil {
		ctx.AbortWithStatusJSON(500, gin.H{"status": 500, "message": err.Error()})
		return
	}
	go startAlpacaWebsocketHandler()
	ctx.JSON(200, gin.H{"status": 200, "message": "success"})
	logger.Info().Msg("successfully logged in")
}

func startAlpacaWebsocketHandler() {
	logger.Info().Msg("starting websocket handler")

	//start listening for messages on different thread
	go alpacaWebsocketClient.Listen()

	//subscribe to stocks
	alpacaWebsocketClient.Subscribe([]string{"AAPL", "AMD", "SPY", "QQQ"}, []string{"AAPL", "AMD", "SPY", "QQQ"})

	//catch any os interrupts
	osInterrupts := make(chan os.Signal, 1)
	signal.Notify(osInterrupts, os.Interrupt)

	//handle websocket events
	for {
		select {
		case subscription := <- alpacaWebsocketClient.Subscriptions:
			eventLogger.Info().
				Str("eventType", "subscription").
				Object("event", subscription).
				Msg("event occurred")

		case trade := <- alpacaWebsocketClient.Trades:
			eventLogger.Info().
				Str("eventType", "trade").
				Object("event", trade).
				Msg("event occurred")

		case quote := <- alpacaWebsocketClient.Quotes:
			eventLogger.Info().
				Str("eventType", "quote").
				Object("event", quote).
				Msg("event occurred")

		case err := <- alpacaWebsocketClient.Errata:
			eventLogger.Info().
				Str("eventType", "error").
				Object("event", err).
				Msg("event occurred")

		case fault := <- alpacaWebsocketClient.Faults:
			logger.Error().Err(fault).Msg("connection failed")
			alpacaWebsocketClient.Disconnect()
			return

		case <-osInterrupts:
			logger.Warn().Msg("manual os interrupt")
			alpacaWebsocketClient.Disconnect()
			os.Exit(0)
			return
		}
	}
}
