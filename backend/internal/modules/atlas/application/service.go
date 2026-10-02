package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nexus/internal/domain/errors"
	"github.com/yurythx/projeto-nexus/internal/domain/events"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/typesense"
	"github.com/yurythx/projeto-nexus/internal/platform/audit"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
	"github.com/yurythx/projeto-nexus/internal/platform/outbox"
)

const (
	EventWorkflowCreated = "atlas.workflow.created"
	EventWorkflowUpdated = "atlas.workflow.updated"
	EventTTDDCreated     = "atlas.ttdd.created"
)

type Service struct {
	pool            *pgxpool.Pool
	repo            domain.Repository
	outbox          *outbox.Writer
	typesenseClient *typesense.Client
	logger          *slog.Logger
}

func NewService(pool *pgxpool.Pool, repo domain.Repository, ob *outbox.Writer, ts *typesense.Client, logger *slog.Logger) *Service {
	return &Service{
		pool:            pool,
		repo:            repo,
		outbox:          ob,
		typesenseClient: ts,
		logger:          logger,
	}
}

// MapError traduz erros de domínio e invariantes para respostas padronizadas da API
func MapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrWorkflowNotFound):
		return apperrors.NotFound("procedimento processual não encontrado")
	case errors.Is(err, domain.ErrClassificacaoNotFound):
		return apperrors.NotFound("classificação TTDD não encontrada")
	case errors.Is(err, domain.ErrEtapaNotFound):
		return apperrors.NotFound("etapa de tramitação não encontrada")
	case errors.Is(err, domain.ErrInvalidWorkflowVersion):
		return apperrors.Conflict("versão de workflow já existente para este código")
	case errors.Is(err, domain.ErrInvalidInput):
		return apperrors.Validation(err.Error())
	}
	return err
}

func (s *Service) ListTTDD(ctx context.Context, req ListTTDDRequest) (*ListTTDDResponse, error) {
	items, total, err := s.repo.ListClassificacoes(ctx, s.pool, req.Query, req.Limit, req.Offset)
	if err != nil {
		return nil, MapError(err)
	}
	return &ListTTDDResponse{
		Total: total,
		Items: items,
	}, nil
}

func (s *Service) GetTTDDByCodigo(ctx context.Context, codigo string) (*domain.ClassificacaoTTDD, error) {
	c, err := s.repo.GetClassificacaoByCodigo(ctx, s.pool, codigo)
	if err != nil {
		return nil, MapError(err)
	}
	return c, nil
}

func (s *Service) ListWorkflows(ctx context.Context, req ListWorkflowsRequest) (*ListWorkflowsResponse, error) {
	items, total, err := s.repo.ListWorkflows(ctx, s.pool, req.TenantID, req.Query, req.CodigoTTDD, req.Limit, req.Offset)
	if err != nil {
		return nil, MapError(err)
	}
	return &ListWorkflowsResponse{
		Total: total,
		Items: items,
	}, nil
}

func (s *Service) GetWorkflowByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	wf, err := s.repo.GetWorkflowByID(ctx, s.pool, id)
	if err != nil {
		return nil, MapError(err)
	}
	return wf, nil
}

// CreateWorkflow grava o workflow, emite o evento no Transactional Outbox e registra
// a trilha de auditoria encadeada com SHA-256 na MESMA transação atômica.
func (s *Service) CreateWorkflow(ctx context.Context, req CreateWorkflowRequest) (*domain.Workflow, error) {
	if strings.TrimSpace(req.TenantID) == "" {
		req.TenantID = "nexus"
	}

	ttdd, err := s.repo.GetClassificacaoByCodigo(ctx, s.pool, req.CodigoTTDD)
	if err != nil {
		return nil, fmt.Errorf("código TTDD inválido: %w", err)
	}

	wf := &domain.Workflow{
		ID:               uuid.New(),
		TenantID:         req.TenantID,
		CodigoProcessual: strings.TrimSpace(req.CodigoProcessual),
		Titulo:           strings.TrimSpace(req.Titulo),
		Objetivo:         strings.TrimSpace(req.Objetivo),
		PublicoAlvo:      strings.TrimSpace(req.PublicoAlvo),
		Versao:           1,
		Ativo:            true,
		NivelAcesso:      req.NivelAcesso,
		HipoteseLegal:    strings.TrimSpace(req.HipoteseLegal),
		CodigoTTDD:       req.CodigoTTDD,
		Classificacao:    ttdd,
	}

	if err := wf.Validate(); err != nil {
		return nil, MapError(err)
	}

	etapaIDsByOrdem := make(map[int]uuid.UUID)
	for _, eReq := range req.Etapas {
		etapaIDsByOrdem[eReq.Ordem] = uuid.New()
	}

	var etapas []domain.Etapa
	for _, eReq := range req.Etapas {
		etapaID := etapaIDsByOrdem[eReq.Ordem]
		e := domain.Etapa{
			ID:                     etapaID,
			WorkflowID:             wf.ID,
			Ordem:                  eReq.Ordem,
			UnidadeAdministrativa:  eReq.UnidadeAdministrativa,
			NomeSetor:              eReq.NomeSetor,
			AtribuicoesSetor:       eReq.AtribuicoesSetor,
			PrazoSLAEmDias:         eReq.PrazoSLAEmDias,
			ManterAbertoAposRemessa: eReq.ManterAbertoAposRemessa,
		}

		for _, dReq := range eReq.Documentos {
			e.Documentos = append(e.Documentos, domain.EtapaDocumento{
				ID:                    uuid.New(),
				EtapaID:               etapaID,
				NomeDocumento:         dReq.NomeDocumento,
				Obrigatorio:           dReq.Obrigatorio,
				Formato:               dReq.Formato,
				TipoAssinatura:        dReq.TipoAssinatura,
				ExigeConferenciaCopia: dReq.ExigeConferenciaCopia,
				ModeloMinutaPadraoURL: dReq.ModeloMinutaPadraoURL,
			})
		}

		for _, tReq := range eReq.Transicoes {
			destinoID := etapaIDsByOrdem[tReq.DestinoEtapaOrdem]
			e.Transicoes = append(e.Transicoes, domain.EtapaTransicao{
				ID:                    uuid.New(),
				OrigemEtapaID:         etapaID,
				DestinoEtapaID:        destinoID,
				CondicaoTransicao:     tReq.CondicaoTransicao,
				IsDevolucaoDiligencia: tReq.IsDevolucaoDiligencia,
				DescricaoDiligencia:   tReq.DescricaoDiligencia,
			})
		}

		etapas = append(etapas, e)
	}

	wf.Etapas = etapas

	// Commit atômico: Workflow + Outbox + Trilha de Auditoria Imutável
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SaveWorkflow(ctx, tx, wf); err != nil {
			return err
		}

		payload := map[string]any{
			"id":                wf.ID.String(),
			"codigo_processual": wf.CodigoProcessual,
			"titulo":            wf.Titulo,
			"codigo_ttdd":       wf.CodigoTTDD,
			"nivel_acesso":      wf.NivelAcesso,
		}

		if s.outbox != nil {
			if err := s.outbox.Write(ctx, tx, EventWorkflowCreated, "atlas_workflow", wf.ID.String(), uuid.Nil, payload); err != nil {
				return fmt.Errorf("outbox write: %w", err)
			}
		}

		meta := audit.Meta(ctx, EventWorkflowCreated, "atlas_workflow", wf.ID.String(), nil, wf)
		return audit.NewWriter(tx).Record(ctx, meta)
	})

	if err != nil {
		return nil, MapError(err)
	}

	s.logger.InfoContext(ctx, "workflow atlas cadastrado com outbox e auditoria",
		"id", wf.ID, "codigo_processual", wf.CodigoProcessual, "titulo", wf.Titulo)

	return wf, nil
}

// IndexWorkflowEvent processa o evento assíncrono emitido pelo Outbox e indexa no Typesense
func (s *Service) IndexWorkflowEvent(ctx context.Context, ev events.Event) error {
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal atlas event payload: %w", err)
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return fmt.Errorf("parse workflow id %q: %w", payload.ID, err)
	}

	wf, err := s.repo.GetWorkflowByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("obter workflow para indexação: %w", err)
	}

	if s.typesenseClient != nil {
		if err := s.typesenseClient.IndexWorkflow(ctx, wf); err != nil {
			s.logger.WarnContext(ctx, "falha na indexação Typesense", "error", err, "wf_id", wf.ID)
		}
	}
	return nil
}

