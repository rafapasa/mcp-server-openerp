package response

import (
    "strconv"
    "github.com/gofiber/fiber/v2"
    "github.com/rafapasa/mcp-server-openerp/internal/dto"
)

func OK[T any](c *fiber.Ctx, data T) error {
    return c.Status(fiber.StatusOK).JSON(dto.NewApiResponse(data))
}
func OKWithMessage[T any](c *fiber.Ctx, data T, message string) error {
    return c.Status(fiber.StatusOK).JSON(dto.NewApiResponseWithMessage(data, message))
}
func Created[T any](c *fiber.Ctx, data T) error {
    return c.Status(fiber.StatusCreated).JSON(dto.NewApiResponse(data))
}
func Paginated[T any](c *fiber.Ctx, data []T, total int64, page, limit int) error {
    return c.Status(fiber.StatusOK).JSON(dto.NewPaginated(data, total, page, limit))
}
func NoContent(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) }
func Success(c *fiber.Ctx, status string) error {
    return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{Status: status})
}
func Deleted(c *fiber.Ctx, id any) error {
    return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{Status: "deleted", ID: id})
}
func Deactivated(c *fiber.Ctx) error {
    return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{Status: "deactivated"})
}
func Error(c *fiber.Ctx, status int, errorCode string, message string) error {
    return c.Status(status).JSON(dto.ErrorResponse{Error: errorCode, Message: message})
}
func ErrorWithDetails(c *fiber.Ctx, status int, errorCode string, message string, details any) error {
    return c.Status(status).JSON(dto.ErrorResponse{Error: errorCode, Message: message, Details: details})
}
func ValidationError(c *fiber.Ctx, message string) error { return Error(c, fiber.StatusBadRequest, "validation_error", message) }
func NotFound(c *fiber.Ctx, message string) error { return Error(c, fiber.StatusNotFound, "not_found", message) }
func InternalError(c *fiber.Ctx, message string) error { return Error(c, fiber.StatusInternalServerError, "internal_error", message) }
func Unauthorized(c *fiber.Ctx, message string) error { return Error(c, fiber.StatusUnauthorized, "unauthorized", message) }
func Forbidden(c *fiber.Ctx, message string) error { return Error(c, fiber.StatusForbidden, "forbidden", message) }
func Conflict(c *fiber.Ctx, message string) error { return Error(c, fiber.StatusConflict, "conflict", message) }
func BadRequest(c *fiber.Ctx, message string) error { return Error(c, fiber.StatusBadRequest, "bad_request", message) }

func ParseIDParam(c *fiber.Ctx, param string) (uint, error) {
    val := c.Params(param)
    if val == "" { return 0, fiber.NewError(fiber.StatusBadRequest, "ID é obrigatório") }
    id, err := strconv.ParseUint(val, 10, 64)
    if err != nil { return 0, err }
    return uint(id), nil
}
func ParseIDParamOrRespond(c *fiber.Ctx, param string) (uint, bool) {
    id, err := ParseIDParam(c, param)
    if err != nil { _ = ValidationError(c, "ID deve ser um número válido"); return 0, false }
    return id, true
}
func GetQueryInt(c *fiber.Ctx, key string, defaultValue int) int {
    v := c.Query(key)
    if v == "" { return defaultValue }
    i, err := strconv.Atoi(v)
    if err != nil { return defaultValue }
    return i
}
func GetQueryString(c *fiber.Ctx, key string, defaultValue string) string {
    v := c.Query(key)
    if v == "" { return defaultValue }
    return v
}
func GetPaginatedRequest(c *fiber.Ctx) dto.PaginatedRequest {
    r := dto.PaginatedRequest{Page: GetQueryInt(c, "page", 1), Limit: GetQueryInt(c, "limit", 20)}
    r.Normalize()
    return r
}
func GetTenantID(c *fiber.Ctx) (uint, error) {
    v := c.Get("X-Tenant-ID")
    if v == "" { return 0, fiber.NewError(fiber.StatusBadRequest, "X-Tenant-ID header é obrigatório") }
    id, err := strconv.ParseUint(v, 10, 64)
    if err != nil { return 0, err }
    return uint(id), nil
}
func GetTenantIDOrRespond(c *fiber.Ctx) (uint, bool) {
    id, err := GetTenantID(c)
    if err != nil { _ = BadRequest(c, err.Error()); return 0, false }
    return id, true
}
