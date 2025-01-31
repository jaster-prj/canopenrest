package canopenrest

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"

	apicanopenrest "github.com/jaster-prj/canopenrest/external/echoserver/generated/canopenrest"
	"github.com/jaster-prj/canopenrest/external/echoserver/implementation"

	log "github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
	middleware "github.com/oapi-codegen/echo-middleware"
)

// EndpointRegisterer handle Registration of jobs Endpoint to the echo server
type EndpointRegisterer struct {
	handler *Handler
	baseUrl string
}

// CreateEndpointRegisterer returns EndpointRegisterer object
func CreateEndpointRegisterer(handler *Handler, baseUrl string) *EndpointRegisterer {
	if !strings.HasPrefix(baseUrl, "/") {
		baseUrl = `/` + baseUrl
	}
	return &EndpointRegisterer{
		handler: handler,
		baseUrl: baseUrl,
	}
}

// GetBaseUrl returns value of baseUrl
func (er *EndpointRegisterer) GetBaseUrl() string {
	return er.baseUrl
}

// Executes Registration to the given echo-server
func (er *EndpointRegisterer) RegisterEndpoint(router *echo.Group) {
	apicanopenrest.RegisterHandlers(router, er.handler)
}

// Add Swagger-Endpoint to the given echo-server
func (er *EndpointRegisterer) CreateSwagger(router *echo.Group) {
	router.Add("GET", "", func(c echo.Context) error {
		swagger, err := apicanopenrest.GetSwagger()
		if err != nil {
			log.Error().Msgf("Failed to load swagger: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.JSON(http.StatusOK, &swagger)
	})
}

// UseOpts can register optional function like authentification
func (er *EndpointRegisterer) UseOpts(router *echo.Group, opts middleware.Options) {
	swaggerDefinition, err := apicanopenrest.GetSwagger()
	if err != nil {
		log.Error().Str("middleware", "validator").Msg(err.Error())
		return
	}
	router.Use(middleware.OapiRequestValidatorWithOptions(swaggerDefinition, &opts))
}

// NewHandler returns new Handler object
func NewHandler(
	canopenUC implementation.ICanopenRest,
) (*Handler, error) {
	return &Handler{
		canopenUC: canopenUC,
	}, nil
}

// Handler for REST API requests
type Handler struct {
	canopenUC implementation.ICanopenRest
}

// GetNMT handles the GET request for the NMT
func (h *Handler) GetNMT(ctx echo.Context, params apicanopenrest.GetNMTParams) error {
	id, err := h.getIntFromHex(params.Node)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	status, err := h.canopenUC.ReadNmt(int(id))
	if err != nil || status == nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	return ctx.JSON(
		http.StatusOK,
		*status,
	)

}

// PostNMT handles the POST request for the NMT
func (h *Handler) PostNMT(ctx echo.Context, params apicanopenrest.PostNMTParams) error {
	state, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	id, err := h.getIntFromHex(params.Node)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	err = h.canopenUC.WriteNmt(int(id), string(state))
	if err != nil {
		log.Error().Msg(string(state))
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	return ctx.NoContent(http.StatusOK)
}

// GetSDO handles the GET request for the SDO
func (h *Handler) GetSDO(ctx echo.Context, params apicanopenrest.GetSDOParams) error {
	id, err := h.getIntFromHex(params.Node)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	index, err := h.getIntFromHex(params.Index)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	subindex := uint8(0)
	if params.Subindex != nil {
		subindex = uint8(*params.Subindex)
	}
	bytesSDO, err := h.canopenUC.ReadSDO(int(id), uint16(index), subindex)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	accept := ctx.Request().Header.Get("accept")
	switch {
	case strings.Contains(accept, "application/octet-stream"):
		reader := bytes.NewBuffer(bytesSDO)
		return ctx.Stream(http.StatusOK, "application/octet-stream", reader)
	case strings.Contains(accept, "text/plain"):
		responseStr := base64.StdEncoding.EncodeToString(bytesSDO)
		return ctx.String(http.StatusOK, responseStr)
	default:
		log.Error().Msgf("switch to default: %s", accept)
		return ctx.NoContent(http.StatusBadRequest)
	}
}

// PostNMT handles the POST request for the SDO
func (h *Handler) PostSDO(ctx echo.Context, params apicanopenrest.PostSDOParams) error {

	var bytesSDO []byte
	var err error
	content := ctx.Request().Header.Get("Content-Type")
	switch {
	case strings.Contains(content, "application/octet-stream"):
		bytesSDO, err = io.ReadAll(ctx.Request().Body)
		if err != nil {
			log.Error().Msg(err.Error())
			return ctx.NoContent(http.StatusBadRequest)
		}
	case strings.Contains(content, "text/plain"):
		encodedBytes, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			log.Error().Msg(err.Error())
			return ctx.NoContent(http.StatusBadRequest)
		}
		bytesSDO, err = base64.StdEncoding.DecodeString(string(encodedBytes))
		if err != nil {
			log.Error().Msg(err.Error())
			return ctx.NoContent(http.StatusBadRequest)
		}
	default:
		return ctx.NoContent(http.StatusBadRequest)
	}

	id, err := h.getIntFromHex(params.Node)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	index, err := h.getIntFromHex(params.Index)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	subindex := uint8(0)
	if params.Subindex != nil {
		subindex = uint8(*params.Subindex)
	}
	err = h.canopenUC.WriteSDO(int(id), uint16(index), subindex, bytesSDO)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	return ctx.NoContent(http.StatusOK)
}

func (h *Handler) PostNode(ctx echo.Context, params apicanopenrest.PostNodeParams) error {
	bytesEDS, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	id, err := h.getIntFromHex(params.Node)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	err = h.canopenUC.CreateNode(int(id), bytesEDS)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	return ctx.NoContent(http.StatusOK)
}

func (h *Handler) PostFlash(ctx echo.Context, params apicanopenrest.PostFlashParams) error {
	flashFile, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	id, err := h.getIntFromHex(params.Node)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	err = h.canopenUC.FlashNode(int(id), flashFile)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	return ctx.NoContent(http.StatusOK)
}

func (h *Handler) getIntFromHex(hexStr string) (int64, error) {
	numberStr := strings.Replace(hexStr, "0x", "", -1)
	return strconv.ParseInt(numberStr, 16, 64)
}
