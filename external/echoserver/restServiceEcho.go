package echoserver

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// EndpointGroup manage single api endpoint
type EndpointGroup struct {
	group   *echo.Group
	reg     IRegisterer
	baseUrl string
}

// Service for REST API server
type Service struct {
	server         *echo.Echo
	endpointGroups []EndpointGroup
}

// NewService creates a new service object containing the REST API server
func NewService(echoServer *echo.Echo, regs []IRegisterer) *Service {
	echoGroups := map[string]*echo.Group{}
	endpointGroups := []EndpointGroup{}
	for _, reg := range regs {
		group, ok := echoGroups[reg.GetBaseUrl()]
		if !ok {
			group = echoServer.Group(reg.GetBaseUrl())
			echoGroups[reg.GetBaseUrl()] = group
		}
		endpointGroup := EndpointGroup{
			group:   group,
			reg:     reg,
			baseUrl: reg.GetBaseUrl(),
		}
		endpointGroups = append(endpointGroups, endpointGroup)
	}
	return &Service{
		endpointGroups: endpointGroups,
		server:         echoServer,
	}
}

// WithLogger adds a request logger to to the service
func (s *Service) WithLogger() *Service {
	// Log all requests
	for _, endpointGroup := range s.endpointGroups {
		endpointGroup.group.Use(echomiddleware.Logger())
	}
	return s
}

// WithSwaggerUi adds a swagger ui to the service
func (s *Service) WithSwaggerUi(baseUrl string) *Service {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	distFolder := filepath.Join(exePath, "dist")
	uri := baseUrl + "/swaggerui"
	s.server.Static(uri, distFolder)
	s.server.Static("/swaggerui", distFolder)
	for _, endpointGroup := range s.endpointGroups {
		endpointGroup.reg.CreateSwagger(endpointGroup.group)
	}
	return s
}

// e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
// 	AllowOrigins: []string{"*"},
// 	AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
//   }))

// WithCors adds CORS middleware to the service
func (s *Service) WithCors() *Service {
	corsConfig := echomiddleware.CORSConfig{
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowOrigins: []string{"*"},
	}
	s.server.Use(
		echomiddleware.CORSWithConfig(corsConfig),
	)
	return s
}

// Add middleware to log request content
func (s *Service) WithDebug() *Service {
	s.server.Use(
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				log.Debug().
					Str("middleware", "debug").
					Str("url", fmt.Sprintf("%v", c.Request().URL)).
					Str("header", fmt.Sprintf("%v", c.Request().Header)).
					Msg("New Request")
				if err := next(c); err != nil {
					c.Error(err)
				}
				return nil
			}
		},
	)
	return s
}

// RegisterUrls registers URLs of Endpoint Groups
func (s *Service) RegisterUrls() {
	for _, endpointGroup := range s.endpointGroups {
		endpointGroup.reg.RegisterEndpoint(endpointGroup.group)
		log.Debug().Msgf("url registered: %s", endpointGroup.reg.GetBaseUrl())
	}
}
