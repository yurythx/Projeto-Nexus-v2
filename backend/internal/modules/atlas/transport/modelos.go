package transport

import (
	"io"
	"mime"
	"net/http"
	"strconv"

	apperrors "github.com/yurythx/projeto-nexus/internal/domain/errors"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/application"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/auth"
	"github.com/yurythx/projeto-nexus/pkg/httputil"
)

// Biblioteca de modelos de documento (ADR 024): consulta e download
// públicos (como o resto do Atlas); cadastro e versões com atlas:manage.
// O arquivo passa pela API (multipart na ida, stream na volta): mesma
// origem do sistema, sem URL do MinIO exposta ao navegador.

// maxCorpoModelo: o arquivo (até 10 MB) mais os campos do formulário.
const maxCorpoModelo = domain.MaxModeloBytes + 1<<20

// semAutoria tira quem publicou das respostas públicas (minimização, LGPD
// art. 6º III): a consulta anônima não precisa saber quem foi.
func semAutoria(m domain.Modelo) domain.Modelo {
	m.UpdatedBy, m.Atual.CreatedBy = "", ""
	for i := range m.Versoes {
		m.Versoes[i].CreatedBy = ""
	}
	return m
}

func (h *Handlers) ListModelosPublic(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListModelos(r.Context(), false)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for i := range items {
		items[i] = semAutoria(items[i])
	}
	httputil.WriteOK(w, items)
}

func (h *Handlers) ListModelosAdmin(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListModelos(r.Context(), true)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, items)
}

func (h *Handlers) GetModelo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	m, err := h.svc.GetModelo(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, semAutoria(m))
}

// ArquivoModelo baixa o arquivo da versão atual (ou de ?versao=N).
func (h *Handlers) ArquivoModelo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	versao := 0
	if q := r.URL.Query().Get("versao"); q != "" {
		if versao, err = strconv.Atoi(q); err != nil || versao < 1 {
			h.fail(w, r, apperrors.BadRequest("versão inválida"))
			return
		}
	}
	v, rc, err := h.svc.ArquivoModelo(r.Context(), id, versao)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", v.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(v.Tamanho, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": v.ArquivoNome}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("ETag", `"`+v.SHA256+`"`)
	if _, err := io.Copy(w, rc); err != nil {
		h.logger.WarnContext(r.Context(), "atlas: download de modelo interrompido", "error", err)
	}
}

// formulario lê o multipart: os campos e o arquivo (obrigatório).
func (h *Handlers) formulario(w http.ResponseWriter, r *http.Request) (application.ArquivoEnviado, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCorpoModelo)
	if err := r.ParseMultipartForm(maxCorpoModelo); err != nil {
		return application.ArquivoEnviado{}, apperrors.BadRequest("envie um formulário multipart de até 10 MB")
	}
	a := application.ArquivoEnviado{Nota: r.FormValue("nota")}
	f, hdr, err := r.FormFile("arquivo")
	if err != nil {
		return a, apperrors.Validation("arquivo do modelo obrigatório")
	}
	defer f.Close()
	// O formulário inteiro cabe no limite de memória do ParseMultipartForm:
	// o arquivo já está em memória e a leitura não falha.
	a.Nome = hdr.Filename
	a.Conteudo, _ = io.ReadAll(f)
	return a, nil
}

// CriarModelo: multipart com nome, descricao, arquivo e nota (opcional).
func (h *Handlers) CriarModelo(w http.ResponseWriter, r *http.Request) {
	a, err := h.formulario(w, r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.CriarModelo(r.Context(), identity, domain.Modelo{Nome: r.FormValue("nome"), Descricao: r.FormValue("descricao")}, a)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, m)
}

// NovaVersaoModelo: multipart com arquivo e nota (opcional).
func (h *Handlers) NovaVersaoModelo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	a, err := h.formulario(w, r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.NovaVersaoModelo(r.Context(), identity, id, a)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, m)
}

type alterarModeloRequest struct {
	Nome      string `json:"nome" validate:"required,max=150"`
	Descricao string `json:"descricao" validate:"max=2000"`
	Ativo     bool   `json:"ativo"`
}

// AlterarModelo muda nome, descrição e situação (ativo).
func (h *Handlers) AlterarModelo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req alterarModeloRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	m, err := h.svc.AlterarModelo(r.Context(), identity, domain.Modelo{ID: id, Nome: req.Nome, Descricao: req.Descricao, Ativo: req.Ativo})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, m)
}
