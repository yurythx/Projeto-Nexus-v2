package transport

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/yurythx/projeto-nexus/internal/domain/errors"
	"github.com/yurythx/projeto-nexus/internal/platform/auth"
	"github.com/yurythx/projeto-nexus/pkg/httputil"
)

// Importação e exportação de procedimentos em lote (ADR 025).

type importacaoRequest struct {
	Conteudo string `json:"conteudo" validate:"required"`
	// Hash devolvido pela simulação: aplicar exige o mesmo arquivo.
	Hash string `json:"hash" validate:"omitempty,len=64"`
}

func (h *Handlers) importacao(w http.ResponseWriter, r *http.Request, aplicar bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCarga)
	var req importacaoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.fail(w, r, apperrors.BadRequest("corpo inválido ou acima de 10 MB"))
		return
	}
	if err := httputil.Validate(req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	out, err := h.svc.ImportarProcedimentos(r.Context(), identity, req.Conteudo, req.Hash, aplicar)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

// SimularImportacao mostra o que o arquivo cria ou muda, sem gravar.
func (h *Handlers) SimularImportacao(w http.ResponseWriter, r *http.Request) {
	h.importacao(w, r, false)
}

// AplicarImportacao grava o arquivo simulado (hash), se não houver erros.
func (h *Handlers) AplicarImportacao(w http.ResponseWriter, r *http.Request) {
	h.importacao(w, r, true)
}

// ExportarProcedimentos baixa os procedimentos em vigor no formato da
// importação (procedimentos.json).
func (h *Handlers) ExportarProcedimentos(w http.ResponseWriter, r *http.Request) {
	arq, err := h.svc.ExportarProcedimentos(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="procedimentos.json"`)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(arq)
}

// Cobertura é o painel de cobertura do Atlas (gestão).
func (h *Handlers) Cobertura(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Cobertura(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, c)
}
