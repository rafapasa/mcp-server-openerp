package models

import "fmt"

const (
	StatusPendente   = "pendente"
	StatusConfirmado = "confirmado"
	StatusPreparando = "preparando"
	StatusEntregue   = "entregue"
	StatusCancelado  = "cancelado"
	OrigemWhatsApp   = "whatsapp"
	OrigemDashboard  = "dashboard"
	OrigemAPI        = "api"
)

type Status int

const (
	StatusAtivo     Status = 1
	StatusInativo   Status = 2
	StatusBloqueado Status = 3
)

// Status do cliente (coluna clientes.status — valores minúsculos alinhados ao ENUM MySQL)
const (
	StatusClienteAtivo             = "ativo"
	StatusClienteInativo           = "inativo"
	StatusClientePendenteValidacao = "pendente_validacao"
)

// String retorna a representação textual do status
func (s Status) String() string {
	switch s {
	case StatusAtivo:
		return "Ativo"
	case StatusInativo:
		return "Inativo"
	case StatusBloqueado:
		return "Bloqueado"
	default:
		return "Desconhecido"
	}
}

// IsActive verifica se o status é ativo
func (s Status) IsActive() bool {
	return s == StatusAtivo
}

// IsInactive verifica se o status é inativo
func (s Status) IsInactive() bool {
	return s == StatusInativo
}

// IsBlocked verifica se o status é bloqueado
func (s Status) IsBlocked() bool {
	return s == StatusBloqueado
}

func (s Status) IsValid() error {
	if s == StatusAtivo || s == StatusInativo || s == StatusBloqueado {
		return nil
	}
	return fmt.Errorf("status inválido: %d. Valores válidos são 1 (Ativo), 2 (Inativo), 3 (Bloqueado)", s)
}

type TaskType string

const (
	TaskExtractKeywords TaskType = "extract_keywords"
	TaskResolveIDs      TaskType = "resolve_ids"
	TaskGreeting        TaskType = "greeting"
	TaskTranscribe      TaskType = "transcribe"
	TaskVision          TaskType = "vision"
)

type MessageSource string

const (
	SourceText  MessageSource = "text"
	SourceAudio MessageSource = "audio"
	SourceImage MessageSource = "image"
)

// routing final:
// - Se msg veio de áudio -> TaskTranscribe (groq) faz transcribe + extract keywords
// - Se veio de imagem -> TaskVision (gemini) faz describe + extract keywords
// - Se veio de texto -> LLM_TEXT extrai keywords
// - ResolveIDs sempre LLM_TEXT (texto puro, independente da origem)
