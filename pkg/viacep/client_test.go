package viacep

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPClientBuscarNormalizaResposta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/89874000/json/", r.URL.Path)
		_, _ = w.Write([]byte(`{"cep":"89874-000","logradouro":"RUA DAS FLORES","bairro":"CENTRO","localidade":"MARAVILHA","uf":"sc"}`))
	}))
	defer server.Close()

	result, err := NewHTTPClient(server.URL, server.Client()).Buscar(context.Background(), "89874-000")
	require.NoError(t, err)
	require.Equal(t, "Rua Das Flores", result.Logradouro)
	require.Equal(t, "Centro", result.Bairro)
	require.Equal(t, "Maravilha", result.Cidade)
	require.Equal(t, "SC", result.Estado)
	require.Equal(t, "89874-000", result.CEP)
}

func TestHTTPClientBuscarRejeitaCEPInvalidoOuNaoEncontrado(t *testing.T) {
	client := NewHTTPClient("http://127.0.0.1", nil)
	_, err := client.Buscar(context.Background(), "123")
	require.Error(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"erro":true}`))
	}))
	defer server.Close()
	_, err = NewHTTPClient(server.URL, server.Client()).Buscar(context.Background(), "89874000")
	require.Error(t, err)
}
