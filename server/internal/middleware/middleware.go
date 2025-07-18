package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// httpStatusMap 错误对应的HTTP状态码映射
var httpStatusMap = map[error]int{
	// 400 Bad Request
	errs.ErrInvalidRequest:      400,
	errs.ErrMissingAPIKey:       400,
	errs.ErrMissingChallenge:    400,
	errs.ErrMissingSessionID:    400,
	errs.ErrMissingUsername:     400,
	errs.ErrMissingName:         400,
	errs.ErrMissingDeviceInfo:   400,
	errs.ErrMissingAdminKey:     400,
	errs.ErrInvalidKey:          400,
	errs.ErrInvalidDeviceID:     400,
	errs.ErrDeviceAlreadyExists: 400,
	errs.ErrDeviceNotActive:     400,
	errs.ErrDeviceAlreadyBound:  400,
	errs.ErrUserAlreadyExists:   400,
	errs.ErrSessionExpired:      400,
	errs.ErrSessionCompleted:    400,

	// 401 Unauthorized
	errs.ErrAPIKeyInvalid: 401,

	// 403 Forbidden
	errs.ErrPermissionDenied: 403,

	// 404 Not Found
	errs.ErrUserNotFound:    404,
	errs.ErrDeviceNotFound:  404,
	errs.ErrSessionNotFound: 404,

	// 503 Service Unavailable
	errs.ErrUserNotOnline: 503,
}

// getHTTPStatus 获取错误对应的HTTP状态码
func getHTTPStatus(err error) int {
	if status, ok := httpStatusMap[err]; ok {
		return status
	}
	return 500 // 默认内部服务器错误
}

// skipper 函数用于跳过特定路由
func skipper(c echo.Context) bool {
	// 跳过 /ws WebSocket 路由
	if c.Path() == "/ws" {
		return true
	}
	// 跳过管理员面板路由
	if c.Path() == "/admin" {
		return true
	}
	return false
}

// SetupMiddleware 设置中间件
func SetupMiddleware(e *echo.Echo) {
	// 1. CORS中间件 - 全局应用
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// 2. 恢复中间件（从panic中恢复） - 全局应用
	e.Use(middleware.Recover())

	// 3. 安全头中间件 - 跳过 /ws
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
		Skipper:               skipper,
	}))

	// 4. 请求ID中间件 - 全局应用
	e.Use(middleware.RequestID())

	// 5. 自定义日志中间件 - 全局应用
	e.Use(LoggerMiddleware())

	// 6. 限流中间件 - 跳过 /ws，使用配置中的限流设置
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store:   middleware.NewRateLimiterMemoryStore(rate.Limit(global.Config.HTTP.RateLimit)),
		Skipper: skipper,
	}))

	// 7. 请求大小限制 - 跳过 /ws，使用配置中的大小限制
	e.Use(middleware.BodyLimitWithConfig(middleware.BodyLimitConfig{
		Limit:   global.Config.HTTP.RequestBodySize,
		Skipper: skipper,
	}))

	// 8. 超时中间件 - 跳过 /ws，使用配置中的超时设置
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: global.Config.HTTP.RequestTimeout,
		Skipper: skipper,
	}))
}

// LoggerMiddleware 自定义日志中间件
func LoggerMiddleware() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogURI:       true,
		LogError:     true,
		LogMethod:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogRequestID: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				logger.Logger.Error("HTTP请求错误", "error", v.Error, "method", v.Method, "uri", v.URI, "status", v.Status)
			}
			return nil
		},
	})
}

// ErrorHandler 自定义错误处理中间件
func ErrorHandler(err error, c echo.Context) {
	code := getHTTPStatus(err)
	message := err.Error()

	// 记录错误日志
	logger.Logger.Error("HTTP错误处理", "error", err, "method", c.Request().Method, "uri", c.Request().RequestURI, "status", code)

	if !c.Response().Committed {
		if c.Request().Method == echo.HEAD {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, &response.Response{
				Success: false,
				Message: message,
			})
		}
		if err != nil {
			// 忽略发送错误
		}
	}
}
