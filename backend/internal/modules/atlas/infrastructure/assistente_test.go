package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
)

func TestLLMResponder(t *testing.T) {
	var got struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct{ Role, Content string }
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, "rota/credencial", http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"  Orientação  "}}]}`))
	}))
	defer srv.Close()

	l := NewLLM(srv.URL+"/", "k", "modelo", time.Second)
	wf := domain.Workflow{CodigoProcessual: "ADM.LIC.001", Titulo: "Pregão", Etapas: []domain.Etapa{{Ordem: 1, NomeSetor: "Licitações"}}}
	answer, err := l.Responder(context.Background(), "como licitar?", []domain.Workflow{wf})
	if err != nil || answer != "Orientação" {
		t.Fatalf("resposta = %q, %v", answer, err)
	}
	if got.Model != "modelo" || got.Temperature > 0.1 || len(got.Messages) != 2 || got.Messages[0].Role != "system" {
		t.Fatalf("requisição inesperada: %+v", got)
	}
	if !strings.Contains(got.Messages[1].Content, "ADM.LIC.001") || !strings.Contains(got.Messages[1].Content, "como licitar?") {
		t.Fatalf("contexto/pergunta ausentes: %s", got.Messages[1].Content)
	}

	// Endpoint já com /v1 não duplica o sufixo.
	if NewLLM("http://x/v1", "", "m", time.Second).url != "http://x/v1/chat/completions" {
		t.Fatal("url com /v1 duplicado")
	}
}

func TestLLMErros(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"http 500": func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "segredo do contexto", 500) },
		"vazia":    func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"choices":[]}`)) },
		"ilegível": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`não é json`)) },
		"sem texto": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":" "}}]}`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(handler)
			defer srv.Close()
			_, err := NewLLM(srv.URL, "", "m", time.Second).Responder(context.Background(), "p", nil)
			if err == nil {
				t.Fatal("esperado erro")
			}
			if strings.Contains(err.Error(), "segredo") {
				t.Fatal("corpo do erro do provedor vazou na mensagem")
			}
		})
	}
	// URL que não vira requisição (caractere de controle).
	if _, err := NewLLM("http://x/\x7f", "", "m", time.Second).Responder(context.Background(), "p", nil); err == nil {
		t.Fatal("esperado erro de URL inválida")
	}
	// Provedor fora do ar.
	if _, err := NewLLM("http://127.0.0.1:1", "", "m", 200*time.Millisecond).Responder(context.Background(), "p", nil); err == nil {
		t.Fatal("esperado erro de conexão")
	}
}
