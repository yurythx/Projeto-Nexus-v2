// Package application contém os casos de uso do Atlas.
package application

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nexus/internal/domain/errors"
	"github.com/yurythx/projeto-nexus/internal/domain/pagination"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/audit"
	"github.com/yurythx/projeto-nexus/internal/platform/auth"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
	"github.com/yurythx/projeto-nexus/internal/platform/outbox"
)

// Eventos emitidos.
const (
	EventWorkflowCreated     = "atlas.workflow.created"
	EventWorkflowActivated   = "atlas.workflow.activated"
	EventWorkflowDeactivated = "atlas.workflow.deactivated"
)

// Grounding: quantos candidatos o índice entrega e quantos procedimentos,
// no máximo, sustentam uma resposta.
const (
	maxCandidatos = 10
	maxFontes     = 3
)

// Service implementa os casos de uso.
type Service struct {
	pool       *pgxpool.Pool
	repo       domain.Repository
	outbox     *outbox.Writer
	assistente domain.Assistente // nil = síntese canônica, sem IA
	logger     *slog.Logger
	now        func() time.Time
}

// NewService cria o serviço. assistente pode ser nil (IA desligada).
func NewService(pool *pgxpool.Pool, repo domain.Repository, ob *outbox.Writer, assistente domain.Assistente, logger *slog.Logger) *Service {
	return &Service{pool: pool, repo: repo, outbox: ob, assistente: assistente, logger: logger, now: time.Now}
}

// MapError traduz erros de domínio.
func MapError(err error) error {
	var inv domain.InvalidError
	switch {
	case errors.As(err, &inv):
		return apperrors.Validation(inv.Msg)
	case errors.Is(err, domain.ErrNotFound):
		return apperrors.NotFound("procedimento não encontrado")
	case errors.Is(err, domain.ErrTTDDNotFound):
		return apperrors.NotFound("classificação TTDD não encontrada")
	case errors.Is(err, domain.ErrDuplicate):
		return apperrors.Conflict("já existe procedimento com este código e versão")
	}
	return err
}

// ListTTDD lista a Tabela de Temporalidade.
func (s *Service) ListTTDD(ctx context.Context, f domain.FiltroTTDD, p pagination.Params) ([]domain.ClassificacaoTTDD, int64, error) {
	items, total, err := s.repo.ListTTDD(ctx, s.pool, f, p)
	return items, total, MapError(err)
}

// maxExportacao limita a exportação (a TTDD inteira tem ~1.700 séries).
const maxExportacao = 20000

// ExportarTTDD devolve todas as séries que atendem ao filtro (para CSV),
// percorrendo as páginas no teto da paginação.
func (s *Service) ExportarTTDD(ctx context.Context, f domain.FiltroTTDD) ([]domain.ClassificacaoTTDD, error) {
	var out []domain.ClassificacaoTTDD
	for page := 1; len(out) < maxExportacao; page++ {
		p := pagination.New(page, pagination.AbsoluteMaxPageSize, pagination.AbsoluteMaxPageSize)
		items, total, err := s.repo.ListTTDD(ctx, s.pool, f, p)
		if err != nil {
			return nil, MapError(err)
		}
		out = append(out, items...)
		if int64(len(out)) >= total || len(items) == 0 {
			break
		}
	}
	return out, nil
}

// EstruturaTTDD devolve a árvore órgão > função > subfunção da TTDD.
func (s *Service) EstruturaTTDD(ctx context.Context) ([]domain.EstruturaTTDD, error) {
	return s.repo.EstruturaTTDD(ctx, s.pool)
}

// GetTTDD devolve uma classificação.
func (s *Service) GetTTDD(ctx context.Context, codigo string) (domain.ClassificacaoTTDD, error) {
	c, err := s.repo.GetTTDD(ctx, s.pool, codigo)
	return c, MapError(err)
}

// HistoricoTTDD devolve as mudanças da série (prazos anteriores, revogação
// e restabelecimento), da mais recente à mais antiga.
func (s *Service) HistoricoTTDD(ctx context.Context, codigo string) ([]domain.HistoricoTTDD, error) {
	if _, err := s.repo.GetTTDD(ctx, s.pool, codigo); err != nil {
		return nil, MapError(err)
	}
	h, err := s.repo.HistoricoTTDD(ctx, s.pool, codigo)
	return h, MapError(err)
}

// List lista procedimentos (inativos só com f.IncluirInativos — a rota de
// gestão exige atlas:manage).
func (s *Service) List(ctx context.Context, f domain.Filter, p pagination.Params) ([]domain.Workflow, int64, error) {
	items, total, err := s.repo.List(ctx, s.pool, f, p)
	return items, total, MapError(err)
}

// Get devolve o procedimento completo. Desativado só aparece para a gestão.
func (s *Service) Get(ctx context.Context, id uuid.UUID, incluirInativo bool) (domain.Workflow, error) {
	w, err := s.repo.Get(ctx, s.pool, id, false)
	if err == nil && !w.Ativo && !incluirInativo {
		err = domain.ErrNotFound
	}
	return w, MapError(err)
}

// Search alimenta a Busca Global.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]domain.Workflow, []float64, error) {
	return s.repo.Search(ctx, s.pool, query, limit)
}

// Create cadastra um procedimento (atlas:manage exigido na rota) com
// evento e auditoria na mesma transação.
func (s *Service) Create(ctx context.Context, identity auth.Identity, w domain.Workflow) (domain.Workflow, error) {
	w.Normalize()
	if w.Versao == 0 {
		w.Versao = 1
	}
	if err := w.Validate(); err != nil {
		return domain.Workflow{}, MapError(err)
	}
	var out domain.Workflow
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) (err error) {
		out, err = s.inserir(ctx, tx, identity, w)
		return err
	})
	return out, MapError(err)
}

// NovaVersao cadastra a versão seguinte do procedimento `origem` (mesmo
// código processual, conteúdo de w) e desativa as outras versões ativas,
// tudo na mesma transação: a consulta pública nunca vê duas versões
// vigentes nem nenhuma.
func (s *Service) NovaVersao(ctx context.Context, identity auth.Identity, origem uuid.UUID, w domain.Workflow) (domain.Workflow, error) {
	var out domain.Workflow
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		base, err := s.repo.Get(ctx, tx, origem, true)
		if err != nil {
			return err
		}
		maior, err := s.repo.MaxVersao(ctx, tx, base.CodigoProcessual)
		if err != nil {
			return err
		}
		w.Normalize()
		w.CodigoProcessual, w.Versao = base.CodigoProcessual, maior+1
		if err := w.Validate(); err != nil {
			return err
		}
		if out, err = s.inserir(ctx, tx, identity, w); err != nil {
			return err
		}
		ids, err := s.repo.DesativarVersoes(ctx, tx, out.CodigoProcessual, out.ID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			antiga, err := s.repo.Get(ctx, tx, id, false)
			if err != nil {
				return err
			}
			if err := s.substituida(ctx, tx, antiga, out); err != nil {
				return err
			}
		}
		return nil
	})
	return out, MapError(err)
}

// substituida registra (evento e auditoria) a versão desativada por `nova`.
func (s *Service) substituida(ctx context.Context, tx pgx.Tx, antiga, nova domain.Workflow) error {
	if err := s.event(ctx, tx, EventWorkflowDeactivated, antiga); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventWorkflowDeactivated, "atlas_workflow", antiga.ID.String(),
		map[string]any{"ativo": true}, map[string]any{"ativo": false, "substituido_por": nova.ID.String(), "versao": nova.Versao}))
}

// inserir grava o procedimento (já validado) ativo, com evento e auditoria,
// na transação: a série da TTDD tem de existir e estar vigente.
func (s *Service) inserir(ctx context.Context, tx pgx.Tx, identity auth.Identity, w domain.Workflow) (domain.Workflow, error) {
	w.ID, w.Ativo = uuid.New(), true
	if identity.UserID != uuid.Nil {
		uid := identity.UserID
		w.CreatedBy = &uid
	}
	for i := range w.Etapas {
		w.Etapas[i].ID = uuid.New()
		for j := range w.Etapas[i].Documentos {
			w.Etapas[i].Documentos[j].ID = uuid.New()
		}
		for j := range w.Etapas[i].Transicoes {
			w.Etapas[i].Transicoes[j].ID = uuid.New()
		}
	}
	situacao, err := s.repo.LockTTDD(ctx, tx, w.CodigoTTDD)
	if err != nil {
		return w, err
	}
	switch situacao {
	case domain.TTDDInexistente:
		return w, domain.InvalidError{Msg: "código TTDD inexistente na Tabela de Temporalidade: " + w.CodigoTTDD}
	case domain.TTDDRevogada:
		return w, domain.InvalidError{Msg: "a série " + w.CodigoTTDD + " foi revogada na TTDD em vigor: enquadre o procedimento numa série vigente"}
	}
	if err := s.repo.Insert(ctx, tx, w); err != nil {
		return w, err
	}
	out, err := s.repo.Get(ctx, tx, w.ID, false)
	if err != nil {
		return w, err
	}
	if err := s.event(ctx, tx, EventWorkflowCreated, out); err != nil {
		return w, err
	}
	return out, audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventWorkflowCreated, "atlas_workflow", out.ID.String(), nil, summary(out)))
}

// SetAtivo ativa ou desativa um procedimento (atlas:manage na rota).
// Repetir o estado atual não regrava nem emite evento.
func (s *Service) SetAtivo(ctx context.Context, id uuid.UUID, ativo bool) (domain.Workflow, error) {
	var out domain.Workflow
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		before, err := s.repo.Get(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if before.Ativo == ativo {
			out = before
			return nil
		}
		if err := s.repo.SetAtivo(ctx, tx, id, ativo); err != nil {
			return err
		}
		if out, err = s.repo.Get(ctx, tx, id, false); err != nil {
			return err
		}
		event := EventWorkflowDeactivated
		if ativo {
			event = EventWorkflowActivated
		}
		if err := s.event(ctx, tx, event, out); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, event, "atlas_workflow", id.String(),
			map[string]any{"ativo": before.Ativo}, map[string]any{"ativo": out.Ativo}))
	})
	return out, MapError(err)
}

func (s *Service) event(ctx context.Context, tx pgx.Tx, eventType string, w domain.Workflow) error {
	return s.outbox.Write(ctx, tx, eventType, "atlas_workflow", w.ID.String(), uuid.Nil, summary(w))
}

func summary(w domain.Workflow) map[string]any {
	return map[string]any{"id": w.ID.String(), "codigo_processual": w.CodigoProcessual, "versao": w.Versao,
		"titulo": w.Titulo, "codigo_ttdd": w.CodigoTTDD, "nivel_acesso": w.NivelAcesso, "ativo": w.Ativo, "etapas": len(w.Etapas)}
}

// Tipos de fonte do assistente.
const (
	FonteProcedimento = "procedimento"
	FonteTTDD         = "ttdd"
)

// Fonte é o procedimento ou a série da TTDD que sustentou a resposta.
type Fonte struct {
	Tipo       string     `json:"tipo"`
	ID         *uuid.UUID `json:"id,omitempty"`
	Codigo     string     `json:"codigo"`
	Titulo     string     `json:"titulo"`
	Relevancia float64    `json:"relevancia"`
}

// Modos de resposta do assistente.
const (
	ModoIA       = "ia"       // redigida pelo modelo de linguagem
	ModoSintese  = "sintese"  // síntese canônica (IA desligada ou indisponível)
	ModoRecusada = "recusada" // nenhuma fonte acima do limiar
)

// Resposta do assistente procedural.
type Resposta struct {
	Answer      string    `json:"answer"`
	Score       float64   `json:"score"`
	Refused     bool      `json:"refused"`
	Mode        string    `json:"mode"`
	Sources     []Fonte   `json:"sources"`
	GeneratedAt time.Time `json:"generated_at"`
}

// Perguntar responde com grounding estrito: só procedimentos homologados e
// séries da TTDD oficial com relevância >= domain.LimiarRelevancia sustentam
// a resposta (até maxFontes); abaixo disso, a recusa canônica. Exige
// atlas:read (na rota).
func (s *Service) Perguntar(ctx context.Context, pergunta string) (Resposta, error) {
	pergunta = strings.TrimSpace(pergunta)
	resp := Resposta{Sources: []Fonte{}, GeneratedAt: s.now()}

	procs, err := s.repo.Candidatos(ctx, s.pool, pergunta, maxCandidatos)
	if err != nil {
		return resp, MapError(err)
	}
	series, err := s.repo.CandidatosTTDD(ctx, s.pool, pergunta, maxCandidatos)
	if err != nil {
		return resp, MapError(err)
	}
	type candidato struct {
		fonte   Fonte
		sintese string
	}
	var hits []candidato
	for _, w := range procs {
		if r := domain.Relevancia(w, pergunta); r > 0 {
			id := w.ID
			hits = append(hits, candidato{Fonte{Tipo: FonteProcedimento, ID: &id, Codigo: w.CodigoProcessual, Titulo: w.Titulo, Relevancia: r},
				domain.SinteseCanonica(w)})
		}
	}
	for _, c := range series {
		if r := domain.RelevanciaTTDD(c, pergunta); r > 0 {
			hits = append(hits, candidato{Fonte{Tipo: FonteTTDD, Codigo: c.Codigo, Titulo: c.Descritor, Relevancia: r}, domain.SinteseTTDD(c)})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].fonte.Relevancia > hits[j].fonte.Relevancia })
	if len(hits) > 0 {
		resp.Score = hits[0].fonte.Relevancia
	}

	var contexto []string
	for _, h := range hits {
		if h.fonte.Relevancia < domain.LimiarRelevancia || len(contexto) == maxFontes {
			break
		}
		contexto = append(contexto, h.sintese)
		resp.Sources = append(resp.Sources, h.fonte)
	}
	sintese := strings.Join(contexto, "\n\n")

	switch {
	case len(contexto) == 0:
		resp.Answer, resp.Refused, resp.Mode = domain.MensagemRecusa, true, ModoRecusada
	case s.assistente == nil:
		resp.Answer, resp.Mode = sintese, ModoSintese
	default:
		answer, err := s.assistente.Responder(ctx, pergunta, contexto)
		switch {
		case err == nil:
			resp.Answer, resp.Mode = answer, ModoIA
		case errors.Is(err, domain.ErrIADesligada):
			resp.Answer, resp.Mode = sintese, ModoSintese
		default:
			s.logger.WarnContext(ctx, "atlas: assistente de IA indisponível, usando a síntese canônica", "error", err)
			resp.Answer, resp.Mode = sintese, ModoSintese
		}
	}

	// Trilha da consulta sem o texto da pergunta (minimização, LGPD art.
	// 6º III): ela pode trazer dados pessoais de terceiros.
	codigos := make([]string, len(resp.Sources))
	for i, f := range resp.Sources {
		codigos[i] = f.Codigo
	}
	if err := audit.NewWriter(s.pool).Record(ctx, audit.Meta(ctx, "atlas.assistente.consulta", "atlas_assistente", "", nil,
		map[string]any{"modo": resp.Mode, "relevancia": resp.Score, "fontes": codigos, "tamanho_pergunta": len([]rune(pergunta))})); err != nil {
		return Resposta{}, err
	}
	return resp, nil
}
