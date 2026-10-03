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
	series     []domain.ClassificacaoTTDD
	err        error
	errTTDD    error
}

func (f fakeRepo) CandidatosTTDD(context.Context, database.DBTX, string, int) ([]domain.ClassificacaoTTDD, error) {
	return f.series, f.errTTDD
}

func (f fakeRepo) Candidatos(context.Context, database.DBTX, string, int) ([]domain.Workflow, error) {
	return f.candidatos, f.err
}

func (f fakeRepo) ListTTDD(context.Context, database.DBTX, domain.FiltroTTDD, pagination.Params) ([]domain.ClassificacaoTTDD, int64, error) {
	return nil, 0, f.err
}

type fakeAssistente struct {
	answer   string
	err      error
	contexto []string
}

func (f *fakeAssistente) Responder(_ context.Context, _ string, ctx []string) (string, error) {
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
		if len(ia.contexto) != 1 || !strings.Contains(ia.contexto[0], "ADM.LIC.001") || len(r.Sources) != 1 ||
			r.Sources[0].Tipo != FonteProcedimento || r.Sources[0].ID == nil || r.Sources[0].Codigo != "ADM.LIC.001" {
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
	t.Run("IA desligada na configuração usa a síntese", func(t *testing.T) {
		ia := &fakeAssistente{err: domain.ErrIADesligada}
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
		for _, r := range []fakeRepo{{err: errors.New("db")}, {errTTDD: errors.New("db")}} {
			if _, err := NewService(pool, r, outbox.NewWriter("t"), nil, logger).Perguntar(ctx, "pregão"); err == nil {
				t.Fatal("esperado erro")
			}
		}
	})
	t.Run("série da TTDD sustenta a resposta junto com o procedimento", func(t *testing.T) {
		um := 1
		serie := domain.ClassificacaoTTDD{Codigo: "2.0.02.00.07", Descritor: "Processos relativos a Pregão Presencial/Pregão Eletrônico",
			FaseCorrenteAnos: &um, FaseIntermAnos: &um}
		alheia := domain.ClassificacaoTTDD{Codigo: "2.0.01.00.01", Descritor: "Organogramas"}
		r, err := NewService(pool, fakeRepo{candidatos: []domain.Workflow{pregao()}, series: []domain.ClassificacaoTTDD{alheia, serie}},
			outbox.NewWriter("t"), nil, logger).Perguntar(ctx, "prazo de guarda do pregão eletrônico")
		if err != nil || r.Refused || len(r.Sources) != 2 {
			t.Fatalf("resposta = %+v, %v", r, err)
		}
		tipos := map[string]bool{}
		for _, f := range r.Sources {
			tipos[f.Tipo] = true
		}
		if !tipos[FonteTTDD] || !tipos[FonteProcedimento] || !strings.Contains(r.Answer, "2.0.02.00.07") || !strings.Contains(r.Answer, "ADM.LIC.001") {
			t.Fatalf("fontes e síntese: %+v\n%s", r.Sources, r.Answer)
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
