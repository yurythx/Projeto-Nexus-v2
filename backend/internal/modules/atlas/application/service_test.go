package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/yurythx/projeto-nexus/internal/domain/pagination"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
	"github.com/yurythx/projeto-nexus/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nexus/internal/platform/outbox"
)

// fakeRepo entrega séries fixas ao grounding.
type fakeRepo struct {
	domain.Repository
	series  []domain.ClassificacaoTTDD
	err     error
	errTTDD error
}

func (f fakeRepo) CandidatosTTDD(context.Context, database.DBTX, string, int) ([]domain.ClassificacaoTTDD, error) {
	return f.series, f.errTTDD
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

func serie(codigo, descritor string, corrente, interm int, dest domain.DestinacaoFinal) domain.ClassificacaoTTDD {
	return domain.ClassificacaoTTDD{Codigo: codigo, Descritor: descritor, FaseCorrenteAnos: &corrente, FaseIntermAnos: &interm, DestinacaoFinal: &dest}
}

var (
	pasta        = serie("2.0.07.00.00", "Pasta funcional de Servidores Ativos, inativos e aposentados", 1, 99, domain.DestinacaoEliminacao)
	organogramas = serie("2.0.01.00.01", "Organogramas", 1, 1, domain.DestinacaoEliminacao)
)

// O assistente responde SÓ sobre a TTDD (ADR 021).
func TestPerguntar(t *testing.T) {
	pool := dbtest.Pool(t) // a consulta é auditada
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()
	repo := fakeRepo{series: []domain.ClassificacaoTTDD{organogramas, pasta}}
	const pergunta = "  Qual o prazo de guarda da pasta funcional de servidores?  "
	svc := func(ia domain.Assistente, r fakeRepo) *Service {
		return NewService(pool, r, outbox.NewWriter("t"), ia, logger)
	}

	t.Run("IA responde só com a série acima do limiar", func(t *testing.T) {
		ia := &fakeAssistente{answer: "Orientação do modelo"}
		r, err := svc(ia, repo).Perguntar(ctx, pergunta)
		if err != nil || r.Mode != ModoIA || r.Answer != "Orientação do modelo" || r.Refused {
			t.Fatalf("resposta = %+v, %v", r, err)
		}
		if len(ia.contexto) != 1 || !strings.Contains(ia.contexto[0], "99 anos") || len(r.Sources) != 1 ||
			r.Sources[0].Tipo != FonteTTDD || r.Sources[0].Codigo != "2.0.07.00.00" {
			t.Fatalf("contexto enviado ao modelo: %+v / fontes %+v", ia.contexto, r.Sources)
		}
	})
	t.Run("modelo reconhece pedido fora do objetivo", func(t *testing.T) {
		ia := &fakeAssistente{answer: domain.MensagemForaDoObjetivo}
		r, err := svc(ia, repo).Perguntar(ctx, pergunta)
		if err != nil || !r.Refused || r.Mode != ModoRecusada || r.Answer != domain.MensagemForaDoObjetivo || len(r.Sources) != 0 {
			t.Fatalf("resposta = %+v, %v", r, err)
		}
	})
	for nome, ia := range map[string]domain.Assistente{
		"IA indisponível cai na síntese":   &fakeAssistente{err: errors.New("timeout")},
		"IA desligada na configuração":     &fakeAssistente{err: domain.ErrIADesligada},
		"sem IA configurada usa a síntese": nil,
	} {
		t.Run(nome, func(t *testing.T) {
			r, err := svc(ia, repo).Perguntar(ctx, pergunta)
			if err != nil || r.Mode != ModoSintese || !strings.Contains(r.Answer, "2.0.07.00.00") || !strings.Contains(r.Answer, "eliminação") {
				t.Fatalf("resposta = %+v, %v", r, err)
			}
		})
	}
	// Recusas sem chamar o modelo: assunto fora do objetivo, ou sobre
	// temporalidade sem série correspondente (nenhuma ou abaixo do limiar).
	for pergunta, want := range map[string]string{
		"receita de bolo de cenoura":                  domain.MensagemForaDoObjetivo,
		"Como tramitar um processo de pregão?":        domain.MensagemForaDoObjetivo,
		"qual o prazo de guarda dos alvarás de obra?": domain.MensagemSemSerie,
		"organogramas antigos da secretaria de obras": domain.MensagemForaDoObjetivo, // palavra solta não basta
		"prazo para eliminar os mapas de obras":       domain.MensagemSemSerie,
	} {
		ia := &fakeAssistente{answer: "não deveria"}
		r, err := svc(ia, repo).Perguntar(ctx, pergunta)
		if err != nil || !r.Refused || r.Mode != ModoRecusada || r.Answer != want || ia.contexto != nil || len(r.Sources) != 0 {
			t.Errorf("%q: %+v, %v (modelo chamado: %v)", pergunta, r, err, ia.contexto != nil)
		}
	}
	t.Run("falha do repositório propaga", func(t *testing.T) {
		if _, err := svc(nil, fakeRepo{errTTDD: errors.New("db")}).Perguntar(ctx, pergunta); err == nil {
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
	svc := NewService(pool, fakeRepo{series: []domain.ClassificacaoTTDD{pasta}}, outbox.NewWriter("t"), nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := svc.Perguntar(ctx, "prazo de guarda da pasta funcional"); err == nil {
		t.Fatal("falha ao auditar a consulta deve falhar a resposta")
	}
}
