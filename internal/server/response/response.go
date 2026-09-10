package response

import (
	"errors"
	"strconv"

	"github.com/etoolstec/gokit/apperror"
	respfiber "github.com/etoolstec/gokit/response"
	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

func OK[T any](c *fiber.Ctx, data T) error {
	return respfiber.OK(c, data)
}

func OKWithMessage[T any](c *fiber.Ctx, data T, message string) error {
	return respfiber.OKWithMessage(c, data, message)
}

func Created[T any](c *fiber.Ctx, data T) error {
	return respfiber.Created(c, data)
}

func Paginated[T any](c *fiber.Ctx, data []T, total int64, page, limit int) error {
	return respfiber.Paginated(c, data, total, page, limit)
}

func NoContent(c *fiber.Ctx) error {
	return respfiber.NoContent(c)
}

func Success(c *fiber.Ctx, status string) error {
	return respfiber.Success(c, status)
}

func Deleted(c *fiber.Ctx, id any) error {
	return respfiber.Deleted(c, id)
}

func Deactivated(c *fiber.Ctx) error {
	return respfiber.Deactivated(c)
}

func Error(c *fiber.Ctx, status int, errorCode string, message string) error {
	return respfiber.Error(c, status, errorCode, message)
}

func ValidationError(c *fiber.Ctx, message string) error {
	return respfiber.ValidationError(c, message)
}

func NotFound(c *fiber.Ctx, message string) error {
	return respfiber.NotFound(c, message)
}

func InternalError(c *fiber.Ctx, message string) error {
	return respfiber.InternalError(c, message)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return respfiber.Unauthorized(c, message)
}

func Forbidden(c *fiber.Ctx, message string) error {
	return respfiber.Forbidden(c, message)
}

func Conflict(c *fiber.Ctx, message string) error {
	return respfiber.Conflict(c, message)
}

func BadRequest(c *fiber.Ctx, message string) error {
	return respfiber.BadRequest(c, message)
}

// FromError mapeia error (preferencialmente *apperror.AppError) para resposta HTTP.
// Handler da API do front deve usar isto em erros vindos de service/repo/LLM/Redis.
func FromError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	status := apperror.GetHTTPStatus(err)
	msg := err.Error()
	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Message != "" {
		msg = appErr.Message
	}
	switch status {
	case 400:
		return BadRequest(c, msg)
	case 401:
		return Unauthorized(c, msg)
	case 403:
		return Forbidden(c, msg)
	case 404:
		return NotFound(c, msg)
	case 409:
		return Conflict(c, msg)
	case 429:
		return Error(c, 429, "too_many_requests", msg)
	case 503:
		return Error(c, 503, "service_unavailable", msg)
	default:
		if status >= 400 && status < 600 {
			return Error(c, status, "error", msg)
		}
		return InternalError(c, msg)
	}
}

func ParseIDParam(c *fiber.Ctx, param string) (uint, error) {
	val := c.Params(param)
	if val == "" {
		return 0, fiber.NewError(fiber.StatusBadRequest, "ID é obrigatório")
	}
	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func ParseIDParamOrRespond(c *fiber.Ctx, param string) (uint, bool) {
	id, err := ParseIDParam(c, param)
	if err != nil {
		_ = ValidationError(c, "ID deve ser um número válido")
		return 0, false
	}
	return id, true
}

func GetQueryInt(c *fiber.Ctx, key string, defaultValue int) int {
	v := c.Query(key)
	if v == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return i
}

func GetQueryString(c *fiber.Ctx, key string, defaultValue string) string {
	v := c.Query(key)
	if v == "" {
		return defaultValue
	}
	return v
}

func GetPaginatedRequest(c *fiber.Ctx) dto.PaginatedRequest {
	r := dto.PaginatedRequest{Page: GetQueryInt(c, "page", 1), Limit: GetQueryInt(c, "limit", 20)}
	r.Normalize()
	return r
}

func GetTenantID(c *fiber.Ctx) (uint, error) {
	v := c.Get("X-Tenant-ID")
	if v == "" {
		return 0, fiber.NewError(fiber.StatusBadRequest, "X-Tenant-ID header é obrigatório")
	}
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func GetTenantIDOrRespond(c *fiber.Ctx) (uint, bool) {
	id, err := GetTenantID(c)
	if err != nil {
		_ = BadRequest(c, err.Error())
		return 0, false
	}
	return id, true
}
