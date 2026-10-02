package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nexus/internal/domain/pagination"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
	"github.com/yurythx/projeto-nexus/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nexus/internal/platform/outbox"
)

// fakeRepo entrega candidatos fixos ao grounding.
type fakeRepo struct {
	domain.Repository
	candidatos []domain.Workflow
	err        error
}

func (f fakeRepo) Candidatos(context.Context, database.DBTX, string, int) ([]domain.Workflow, error) {
	return f.candidatos, f.err
}

func (f fakeRepo) ListTTDD(context.Context, database.DBTX, string, pagination.Params) ([]domain.ClassificacaoTTDD, int64, error) {
	return nil, 0, f.err
}

type fakeAssistente struct {
	answer   string
	err      error
	contexto []domain.Workflow
}

func (f *fakeAssistente) Responder(_ context.Context, _ string, ctx []domain.Workflow) (string, error) {
	f.contexto = ctx
	return f.answer, f.err
}

func pregao() domain.Workflow {
	return domain.Workflow{ID: uuid.New(), CodigoProcessual: "ADM.LIC.001", Titulo: "Pregão Eletrônico", Objetivo: "Aquisição de bens",
		NivelAcesso: domain.NivelPublico, Etapas: []domain.Etapa{{Ordem: 1, NomeSetor: "Licitações", UnidadeAdministrativa: "LIC"}}}
}

func diarias() domain.Workflow {
	return domain.Workflow{ID: uuid.New(), CodigoProcessual: "RH.DIA.001", Titulo: "Concessão de Diárias", Objetivo: "Viagem a serviço",
		NivelAcesso: domain.NivelPublico}
}

func TestPerguntar(t *testing.T) {
	pool := dbtest.Pool(t) // a consulta é auditada
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()
	repo := fakeRepo{candidatos: []domain.Workflow{diarias(), pregao()}}

	t.Run("IA responde só com o procedimento acima do limiar", func(t *testing.T) {
		ia := &fakeAssistente{answer: "Orientação do modelo"}
		r, err := NewService(pool, repo, outbox.NewWriter("t"), ia, logger).Perguntar(ctx, "  Como funciona o pregão eletrônico?  ")
		if err != nil || r.Mode != ModoIA || r.Answer != "Orientação do modelo" || r.Refused {
			t.Fatalf("resposta = %+v, %v", r, err)
		}
		if len(ia.contexto) != 1 || ia.contexto[0].CodigoProcessual != "ADM.LIC.001" || len(r.Sources) != 1 {
			t.Fatalf("contexto enviado ao modelo: %+v / fontes %+v", ia.contexto, r.Sources)
		}
	})
	t.Run("IA indisponível cai na síntese canônica", func(t *testing.T) {
		ia := &fakeAssistente{err: errors.New("timeout")}
		r, err := NewService(pool, repo, outbox.NewWriter("t"), ia, logger).Perguntar(ctx, "pregão eletrônico")
		if err != nil || r.Mode != ModoSintese || !strings.Contains(r.Answer, "ADM.LIC.001") {
			t.Fatalf("resposta = %+v, %v", r, err)
		}
	})
	t.Run("sem IA configurada usa a síntese", func(t *testing.T) {
		r, err := NewService(pool, repo, outbox.NewWriter("t"), nil, logger).Perguntar(ctx, "pregão eletrônico")
		if err != nil || r.Mode != ModoSintese {
			t.Fatalf("resposta = %+v, %v", r, err)
		}
	})
	t.Run("abaixo do limiar recusa sem chamar o modelo", func(t *testing.T) {
		ia := &fakeAssistente{answer: "não deveria"}
		r, err := NewService(pool, repo, outbox.NewWriter("t"), ia, logger).Perguntar(ctx, "férias no exterior com licença")
		if err != nil || !r.Refused || r.Mode != ModoRecusada || r.Answer != domain.MensagemRecusa || ia.contexto != nil {
			t.Fatalf("resposta = %+v, %v (modelo chamado: %v)", r, err, ia.contexto != nil)
		}
	})
	t.Run("falha do repositório propaga", func(t *testing.T) {
		_, err := NewService(pool, fakeRepo{err: errors.New("db")}, outbox.NewWriter("t"), nil, logger).Perguntar(ctx, "pregão")
		if err == nil {
			t.Fatal("esperado erro")
		}
	})
}

func TestMapError(t *testing.T) {
	for in, want := range map[error]string{
		domain.InvalidError{Msg: "x"}: "x",
		domain.ErrNotFound:            "procedimento não encontrado",
		domain.ErrTTDDNotFound:        "classificação TTDD não encontrada",
		domain.ErrDuplicate:           "já existe procedimento com este código e versão",
	} {
		if got := MapError(in); got == nil || !strings.Contains(got.Error(), want) {
			t.Fatalf("MapError(%v) = %v", in, got)
		}
	}
	other := errors.New("outro")
	if MapError(other) != other || MapError(nil) != nil {
		t.Fatal("erros desconhecidos passam intactos")
	}
}

// A consulta é auditada: sem conseguir gravar a trilha, não há resposta.
func TestPerguntarFalhaNaAuditoria(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // o repositório falso ignora o contexto; a auditoria não
	svc := NewService(pool, fakeRepo{candidatos: []domain.Workflow{pregao()}}, outbox.NewWriter("t"), nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := svc.Perguntar(ctx, "pregão eletrônico"); err == nil {
		t.Fatal("falha ao auditar a consulta deve falhar a resposta")
	}
}
