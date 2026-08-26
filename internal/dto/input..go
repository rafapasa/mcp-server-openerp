package dto

import "github.com/rafapasa/mcp-server-openerp/internal/models"

type MessageInput struct {
	Text   string
	Audio  []byte
	Image  []byte
	Source models.MessageSource
}
