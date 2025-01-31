package echoserver

import (
	"github.com/labstack/echo/v4"
	middleware "github.com/oapi-codegen/echo-middleware"
)

// Interface for an Endpoint Registrator
type IRegisterer interface {
	GetBaseUrl() string
	RegisterEndpoint(server *echo.Group)
	CreateSwagger(server *echo.Group)
	UseOpts(server *echo.Group, opts middleware.Options)
}
