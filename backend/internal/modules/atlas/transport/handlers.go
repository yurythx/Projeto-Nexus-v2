package transport

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nexus/internal/domain/errors"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/application"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/pkg/httputil"
)

type Handlers struct {
	service   *application.Service
	aiService *application.AIService
	logger    *slog.Logger
}

func NewHandlers(service *application.Service, aiService *application.AIService, logger *slog.Logger) *Handlers {
	return &Handlers{
		service:   service,
		aiService: aiService,
		logger:    logger,
	}
}

func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteError(w, r, h.logger, err)
}

// -----------------------------------------------------------------------------
// TTDD (Tabela de Temporalidade e Destinação de Documentos)
// -----------------------------------------------------------------------------

func (h *Handlers) ListTTDD(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.service.ListTTDD(r.Context(), application.ListTTDDRequest{
		Query:  q,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.WriteOKWithMeta(w, resp.Items, map[string]any{
		"total":  resp.Total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handlers) GetTTDD(w http.ResponseWriter, r *http.Request) {
	codigo := chi.URLParam(r, "codigo")
	if strings.TrimSpace(codigo) == "" {
		h.fail(w, r, apperrors.BadRequest("código TTDD obrigatório"))
		return
	}

	item, err := h.service.GetTTDDByCodigo(r.Context(), codigo)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, item)
}

// -----------------------------------------------------------------------------
// Workflows e Procedimentos SEI
// -----------------------------------------------------------------------------

func (h *Handlers) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	codigoTTDD := r.URL.Query().Get("codigo_ttdd")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "nexus"
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.service.ListWorkflows(r.Context(), application.ListWorkflowsRequest{
		TenantID:   tenantID,
		Query:      q,
		CodigoTTDD: codigoTTDD,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.WriteOKWithMeta(w, resp.Items, map[string]any{
		"total":  resp.Total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handlers) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("ID de workflow inválido"))
		return
	}

	wf, err := h.service.GetWorkflowByID(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, wf)
}

func (h *Handlers) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req application.CreateWorkflowRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}

	wf, err := h.service.CreateWorkflow(r.Context(), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, wf)
}

// -----------------------------------------------------------------------------
// Assistente Procedural IA (Grounding Estrito)
// -----------------------------------------------------------------------------

func (h *Handlers) ChatProcedural(w http.ResponseWriter, r *http.Request) {
	var req application.ProceduralChatRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}

	resp, err := h.aiService.AskProceduralQuestion(r.Context(), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.WriteOK(w, resp)
}

// SearchProviders implementa o provedor de busca textual do Atlas para a Busca Global
func (h *Handlers) SearchWorkflows(ctx context.Context, tenantID, query string) ([]domain.Workflow, error) {
	resp, err := h.service.ListWorkflows(ctx, application.ListWorkflowsRequest{
		TenantID: tenantID,
		Query:    query,
		Limit:    10,
	})
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}
