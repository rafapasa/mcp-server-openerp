package dto

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TenantID uint   `json:"tenant_id"` // vem do X-Tenant-ID header ou body
}

type LoginResponse struct {
	Token   string  `json:"token"`
	User    UserDTO `json:"user"`
	Expires string  `json:"expires_at"`
}

type UserDTO struct {
	ID       uint   `json:"id"`
	TenantID uint   `json:"tenant_id"`
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}
