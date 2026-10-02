package application

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/ai"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/typesense"
)

const RelevanceThreshold = 0.65

type AIService struct {
	typesenseClient *typesense.Client
	llmClient       *ai.LLMClient
	logger          *slog.Logger
}

func NewAIService(ts *typesense.Client, llm *ai.LLMClient, logger *slog.Logger) *AIService {
	return &AIService{
		typesenseClient: ts,
		llmClient:       llm,
		logger:          logger,
	}
}

// AskProceduralQuestion processa a pergunta do usuário garantindo Grounding Estrito e prevenção anti-alucinação
func (s *AIService) AskProceduralQuestion(ctx context.Context, req ProceduralChatRequest) (*ProceduralChatResponse, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return &ProceduralChatResponse{
			Answer:      ai.CanonicalRefusalMessage,
			Score:       0.0,
			Refused:     true,
			GeneratedAt: time.Now(),
		}, nil
	}

	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "nexus"
	}

	// 1. Executa busca híbrida no Typesense (alpha = 0.6)
	results, err := s.typesenseClient.SearchHybrid(ctx, tenantID, query)
	if err != nil {
		s.logger.ErrorContext(ctx, "falha na busca híbrida do atlas", "error", err, "query", query)
		return nil, err
	}

	// 2. Avaliação rigorosa do Threshold de Segurança (0.65)
	if len(results) == 0 || results[0].Score < RelevanceThreshold {
		s.logger.InfoContext(ctx, "consulta rejeitada por threshold de relevância insuficiente (grounding estrito)",
			"query", query, "top_score", func() float64 {
				if len(results) > 0 {
					return results[0].Score
				}
				return 0.0
			}())

		return &ProceduralChatResponse{
			Answer:      ai.CanonicalRefusalMessage,
			Score:       func() float64 { if len(results) > 0 { return results[0].Score }; return 0.0 }(),
			Refused:     true,
			GeneratedAt: time.Now(),
		}, nil
	}

	// Filtra workflows que atenderam ao threshold
	var matchedWorkflows []domain.Workflow
	for _, res := range results {
		if res.Score >= RelevanceThreshold && res.Workflow != nil {
			matchedWorkflows = append(matchedWorkflows, *res.Workflow)
		}
	}

	// 3. Geração de resposta estrita via LLM (temperatura 0.05)
	answer, err := s.llmClient.GenerateProceduralAnswer(ctx, query, matchedWorkflows)
	if err != nil {
		s.logger.ErrorContext(ctx, "falha na geração LLM procedural", "error", err)
		return nil, err
	}

	return &ProceduralChatResponse{
		Answer:      answer,
		Score:       results[0].Score,
		Refused:     false,
		Workflows:   matchedWorkflows,
		GeneratedAt: time.Now(),
	}, nil
}
