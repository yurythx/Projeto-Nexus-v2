package application_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/application"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/ai"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/typesense"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

// mockRepo implementa domain.Repository para testes
type mockRepo struct {
	wfs []domain.Workflow
}

func (m *mockRepo) ListClassificacoes(ctx context.Context, dbtx database.DBTX, query string, limit, offset int) ([]domain.ClassificacaoTTDD, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetClassificacaoByCodigo(ctx context.Context, dbtx database.DBTX, codigo string) (*domain.ClassificacaoTTDD, error) {
	return nil, nil
}
func (m *mockRepo) SaveClassificacao(ctx context.Context, dbtx database.DBTX, c *domain.ClassificacaoTTDD) error {
	return nil
}
func (m *mockRepo) ListWorkflows(ctx context.Context, dbtx database.DBTX, tenantID string, query string, codigoTTDD string, limit, offset int) ([]domain.Workflow, int, error) {
	return m.wfs, len(m.wfs), nil
}
func (m *mockRepo) GetWorkflowByID(ctx context.Context, dbtx database.DBTX, id uuid.UUID) (*domain.Workflow, error) {
	for _, w := range m.wfs {
		if w.ID == id {
			return &w, nil
		}
	}
	return nil, domain.ErrWorkflowNotFound
}
func (m *mockRepo) GetWorkflowByCodigo(ctx context.Context, dbtx database.DBTX, tenantID, codigoProcessual string, versao int) (*domain.Workflow, error) {
	for _, w := range m.wfs {
		if w.CodigoProcessual == codigoProcessual {
			return &w, nil
		}
	}
	return nil, domain.ErrWorkflowNotFound
}
func (m *mockRepo) SaveWorkflow(ctx context.Context, dbtx database.DBTX, w *domain.Workflow) error {
	return nil
}
func (m *mockRepo) SearchProceduralGrounding(ctx context.Context, dbtx database.DBTX, tenantID string, query string, limit int) ([]domain.Workflow, error) {
	return m.wfs, nil
}

func TestAIService_GroundingStrictThreshold(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mock := &mockRepo{
		wfs: []domain.Workflow{
			{
				ID:               uuid.New(),
				TenantID:         "nexus",
				CodigoProcessual: "ADM.LIC.001",
				Titulo:           "Processo Licitatório de Pregão Eletrônico",
				Objetivo:         "Contratação de bens e serviços comuns",
				CodigoTTDD:       "2.0.02.00.07",
				Ativo:            true,
				NivelAcesso:      domain.NivelAcessoPublico,
			},
		},
	}

	tsClient := typesense.NewClient("", "", nil, mock, logger)
	llmClient := ai.NewLLMClient("", "", "", logger)
	aiService := application.NewAIService(tsClient, llmClient, logger)

	t.Run("Consulta fora de contexto deve retornar recusa canônica (score < 0.65)", func(t *testing.T) {
		resp, err := aiService.AskProceduralQuestion(context.Background(), application.ProceduralChatRequest{
			TenantID: "nexus",
			Query:    "Como solicitar autorização para viagem espacial à lua?",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !resp.Refused {
			t.Errorf("esperava Refused = true, obteve %t", resp.Refused)
		}
		if resp.Answer != ai.CanonicalRefusalMessage {
			t.Errorf("esperava mensagem canônica de recusa, obteve: %s", resp.Answer)
		}
	})

	t.Run("Consulta pertinente deve atender ao threshold e retornar orientação", func(t *testing.T) {
		resp, err := aiService.AskProceduralQuestion(context.Background(), application.ProceduralChatRequest{
			TenantID: "nexus",
			Query:    "Pregão Eletrônico ADM.LIC.001",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resp.Refused {
			t.Errorf("esperava Refused = false para consulta homologada, obteve true com score %f", resp.Score)
		}
		if resp.Score < 0.65 {
			t.Errorf("score deve ser >= 0.65, obteve %f", resp.Score)
		}
	})

	t.Run("Consulta vazia retorna recusa canônica imediatamente", func(t *testing.T) {
		resp, err := aiService.AskProceduralQuestion(context.Background(), application.ProceduralChatRequest{
			TenantID: "nexus",
			Query:    "   ",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !resp.Refused {
			t.Errorf("esperava Refused = true para consulta vazia")
		}
		if resp.Answer != ai.CanonicalRefusalMessage {
			t.Errorf("esperava mensagem canônica de recusa")
		}
	})
}
