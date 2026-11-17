package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaster-prj/canopenrest/external/echoserver"
	canopenrestimpl "github.com/jaster-prj/canopenrest/external/echoserver/implementation/canopenrest"
	"github.com/jaster-prj/canopenrest/external/persistence/filestorage"
	"github.com/jaster-prj/canopenrest/usecases/canopenuc"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

const canPort string = "can0"

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	fileStorage, err := filestorage.NewFilestorage()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	var canOpenUC *canopenuc.CanOpenUC
	if os.Getenv("CAN_TRANSPORT") == "udp" {
		host := envOrDefault("CAN_UDP_HOST", "127.0.0.1")
		port := envOrDefault("CAN_UDP_PORT", "46100")
		canOpenUC, err = canopenuc.NewCanOpenUCWithUDP(
			fileStorage,
			fmt.Sprintf("%s:%s", host, port),
			os.Getenv("CAN_UDP_LISTEN"),
		)
	} else {
		canOpenUCConfig := canopenuc.CanOpenUCConfig{
			Persistence: fileStorage,
			CanPort:     envOrDefault("CAN_PORT", "can0"),
		}
		canOpenUC, err = canOpenUCConfig.CreateCanOpenUC()
	}
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	echoServer := echo.New()
	canopenRestHandler, err := canopenrestimpl.NewHandler(canOpenUC)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	var (
		canopenRestEndpoint = "/canopenrest/api/v1"
	)
	apiRegisterer := canopenrestimpl.CreateEndpointRegisterer(canopenRestHandler, canopenRestEndpoint)

	echoserver.NewService(echoServer, []echoserver.IRegisterer{
		apiRegisterer,
	}).
		//		WithDebug().
		WithLogger().
		WithCors().
		WithSwaggerUi("/canopenrest").
		RegisterUrls()

	ex, err := os.Executable()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	exePath := filepath.Dir(ex)
	certPath := filepath.Join(exePath, "certs/cangw.crt")
	keyPath := filepath.Join(exePath, "certs/cangw.key")
	if _, err := os.Stat(certPath); errors.Is(err, os.ErrNotExist) {
		port := 8080
		log.Fatal().Msg(echoServer.Start(fmt.Sprintf(":%d", port)).Error())
	} else if _, err := os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		port := 8080
		log.Fatal().Msg(echoServer.Start(fmt.Sprintf(":%d", port)).Error())
	} else {
		port := 443
		log.Fatal().Msg(echoServer.StartTLS(fmt.Sprintf(":%d", port), certPath, keyPath).Error())
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
