package dto

// ============================================================
// RESPONSE DTOs PADRONIZADOS - MCP SERVER OPEN ERP
// Inspirado no ERP completo (Gin) adaptado para Fiber
// SEMPRE: {data: ...} ou {data: [], total, page, limit, total_pages}
// ERRO: {error: code, message: msg}
// ============================================================

// ApiResponse - envelope padrão para resposta única
type ApiResponse[T any] struct {
    Data    T      `json:"data"`
    Message string `json:"message,omitempty"`
}

// PaginatedResponse - envelope padrão para listas
type PaginatedResponse[T any] struct {
    Data       []T   `json:"data"`
    Total      int64 `json:"total"`
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    TotalPages int64 `json:"total_pages"`
}

type PaginatedRequest struct {
    Page  int `query:"page"`
    Limit int `query:"limit"`
}

func (r *PaginatedRequest) Normalize() {
    if r.Page <= 0 { r.Page = 1 }
    if r.Limit <= 0 || r.Limit > 100 { r.Limit = 20 }
}
func (r PaginatedRequest) Offset() int { return (r.Page - 1) * r.Limit }

type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
    Details any    `json:"details,omitempty"`
}

type SuccessResponse struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
    ID      any    `json:"id,omitempty"`
}

func NewPaginated[T any](data []T, total int64, page, limit int) PaginatedResponse[T] {
    if data == nil { data = []T{} }
    totalPages := int64(0)
    if limit > 0 { totalPages = (total + int64(limit) - 1) / int64(limit) }
    return PaginatedResponse[T]{Data: data, Total: total, Page: page, Limit: limit, TotalPages: totalPages}
}
func NewApiResponse[T any](data T) ApiResponse[T] { return ApiResponse[T]{Data: data} }
func NewApiResponseWithMessage[T any](data T, msg string) ApiResponse[T] { return ApiResponse[T]{Data: data, Message: msg} }
