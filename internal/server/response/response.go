package response

import (
	"errors"

	"github.com/etoolstec/gokit/apperror"
	respfiber "github.com/etoolstec/gokit/response"
	"github.com/gofiber/fiber/v2"
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
