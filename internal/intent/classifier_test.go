package intent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifyV2Handoff(t *testing.T) {
	for _, input := range []string{
		"quero falar com atendente",
		"preciso de ajuda humana",
		"quero suporte",
	} {
		require.Equal(t, IntentFalarComAtendente, ClassifyV2(input, time.Time{}).Type, input)
	}
}

func TestClassifyV2VoltarProBot(t *testing.T) {
	for _, input := range []string{"voltar pro bot", "sair do atendimento"} {
		require.Equal(t, IntentVoltarProBot, ClassifyV2(input, time.Time{}).Type, input)
	}
}
