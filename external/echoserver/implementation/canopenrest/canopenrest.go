package canopenrest

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaster-prj/canopenrest/entities"
	apicanopenrest "github.com/jaster-prj/canopenrest/external/echoserver/generated/canopenrest"
	"github.com/jaster-prj/canopenrest/external/echoserver/implementation"

	log "github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
	middleware "github.com/oapi-codegen/echo-middleware"
)

type FlashOrderState struct {
	Requested time.Time           `json:"requested"`
	Start     *time.Time          `json:"start,omitempty"`
	Finish    *time.Time          `json:"finish,omitempty"`
	State     entities.FlashState `json:"state"`
	Error     *string             `json:"error,omitempty"`
}

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
		responseStr := fmt.Sprintf("%X", bytesSDO)
		return ctx.String(http.StatusOK, addSpacerToHex(responseStr, ":"))
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
		requestBytes, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			log.Error().Msg(err.Error())
			return ctx.NoContent(http.StatusBadRequest)
		}
		request := strings.ReplaceAll(string(requestBytes), ":", "")
		bytesSDO, err = hex.DecodeString(request)
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
	basePath := os.Getenv("CANOPEN_STORAGE")
	var err error
	if basePath == "" {
		basePath, err = os.UserConfigDir()
		if err != nil {
			return err
		}
	}
	out, err := os.Create(path.Join(basePath, "CanOpenRest", "flashfile.bin"))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, ctx.Request().Body)
	if err != nil {
		return err
	}
	out.Close()
	flashFile, err := os.ReadFile(path.Join(basePath, "CanOpenRest", "flashfile.bin"))
	if err != nil {
		return err
	}
	id, err := h.getIntFromHex(params.Node)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	log.Debug().Msgf("flashFile size: %d", len(flashFile))
	order, err := h.canopenUC.FlashNode(int(id), flashFile, params.Version)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	return ctx.String(http.StatusCreated, order.String())
}

func (h *Handler) GetFlash(ctx echo.Context, params apicanopenrest.GetFlashParams) error {
	testOrderId, err := uuid.Parse(params.Id)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	flashStates, err := h.canopenUC.GetFlashState(testOrderId)
	if err != nil {
		log.Error().Msg(err.Error())
		return ctx.NoContent(http.StatusBadRequest)
	}
	return ctx.JSON(http.StatusOK, &FlashOrderState{
		Requested: flashStates.Requested,
		Start:     flashStates.Start,
		Finish:    flashStates.Finish,
		State:     flashStates.State,
		Error:     flashStates.Error,
	})
}

func (h *Handler) getIntFromHex(hexStr string) (int64, error) {
	numberStr := strings.Replace(hexStr, "0x", "", -1)
	return strconv.ParseInt(numberStr, 16, 64)
}

func addSpacerToHex(hexString string, spacer string) string {
	var result strings.Builder
	for i := 0; i < len(hexString); i += 2 {
		if i > 0 {
			result.WriteString(spacer)
		}
		result.WriteString(hexString[i : i+2])
	}
	return result.String()
}
