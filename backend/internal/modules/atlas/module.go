// Package atlas é o módulo Atlas: Catálogo Normativo Procedural, Fluxos SEI,
// Temporalidade TTDD (CCPAD/CONARQ) e Orientação Procedural com IA.
package atlas

import (
	"context"
	"fmt"
	"os"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/application"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/ai"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/persistence"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/infra/typesense"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/transport"
	"github.com/yurythx/projeto-nexus/internal/platform/auth"
	"github.com/yurythx/projeto-nexus/internal/platform/kernel"
	"github.com/yurythx/projeto-nexus/internal/platform/messaging"
	"github.com/yurythx/projeto-nexus/internal/platform/modkit"
	"github.com/yurythx/projeto-nexus/internal/platform/search"
)

// Key é o identificador único do módulo no Kernel.
const Key = "atlas"

// QueueIndexer recebe os eventos do Atlas para indexação contínua no Typesense.
var QueueIndexer = messaging.QueueSpec{
	Name:    "nexus.atlas.indexer",
	DLQName: "nexus.atlas.indexer.dlq",
	RoutingKeys: []string{
		"atlas.workflow.*",
	},
}

type Module struct {
	deps      modkit.Deps
	svc       *application.Service
	aiSvc     *application.AIService
	handlers  *transport.Handlers
}

// New instancia o módulo Atlas com repositório PostgreSQL, cliente Typesense e LLM.
func New(deps modkit.Deps) *Module {
	repo := persistence.NewPostgresRepository()

	// Configuração do Typesense (com fallback PostgreSQL integrado)
	typesenseURL := os.Getenv("TYPESENSE_URL")
	typesenseKey := os.Getenv("TYPESENSE_API_KEY")
	tsClient := typesense.NewClient(typesenseURL, typesenseKey, deps.Pool, repo, deps.Logger)

	svc := application.NewService(deps.Pool, repo, deps.Outbox, tsClient, deps.Logger)

	// Configuração do Contêiner de IA (OpenAI / Ollama / vLLM com temp=0.05)
	aiEndpoint := os.Getenv("AI_ENDPOINT")
	aiKey := os.Getenv("AI_API_KEY")
	aiModel := os.Getenv("AI_MODEL")
	llmClient := ai.NewLLMClient(aiEndpoint, aiKey, aiModel, deps.Logger)

	aiSvc := application.NewAIService(tsClient, llmClient, deps.Logger)
	handlers := transport.NewHandlers(svc, aiSvc, deps.Logger)

	return &Module{
		deps:     deps,
		svc:      svc,
		aiSvc:    aiSvc,
		handlers: handlers,
	}
}

// Consumers implementa kernel.ConsumerProvider para processamento assíncrono de indexação.
func (m *Module) Consumers() []kernel.Consumer {
	return []kernel.Consumer{{Queue: QueueIndexer, Handler: m.svc.IndexWorkflowEvent}}
}

// Manifest implementa kernel.Plugin.
func (m *Module) Manifest() kernel.Manifest {
	return kernel.Manifest{
		Key:            Key,
		Name:           "Atlas (Catálogo Procedural)",
		Description:    "Catálogo canônico de processos SEI, temporalidade TTDD e orientação procedural com IA.",
		DefaultEnabled: true,
		Public:         true,
		Icon:           "clipboard-list",
		Route:          "/atlas",
		Permissions: []kernel.PermissionInfo{
			{Key: string(auth.PermAtlasRead), Description: "Consultar procedimentos, temporalidade TTDD e orientações de processos"},
			{Key: string(auth.PermAtlasManage), Description: "Cadastrar e gerenciar procedimentos e fluxos no Atlas"},
		},
	}
}

// RegisterRoutes implementa kernel.RouteProvider.
func (m *Module) RegisterRoutes(r kernel.Routes) {
	transport.RegisterRoutes(r, m.handlers, m.deps.Logger)
}

// SearchProviders implementa kernel.SearchProvider para Busca Global.
func (m *Module) SearchProviders() []search.Provider {
	return []search.Provider{atlasSearchProvider{svc: m.svc}}
}

type atlasSearchProvider struct {
	svc *application.Service
}

func (atlasSearchProvider) Module() string { return Key }

func (p atlasSearchProvider) Search(ctx context.Context, identity auth.Identity, query string, limit int) ([]search.Result, error) {
	resp, err := p.svc.ListWorkflows(ctx, application.ListWorkflowsRequest{
		TenantID: "nexus",
		Query:    query,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	var results []search.Result
	for _, wf := range resp.Items {
		updated := wf.UpdatedAt
		results = append(results, search.Result{
			Module:    Key,
			Type:      "workflow",
			ID:        wf.ID.String(),
			Title:     fmt.Sprintf("%s — %s", wf.CodigoProcessual, wf.Titulo),
			Snippet:   wf.Objetivo,
			URL:       fmt.Sprintf("/atlas?id=%s", wf.ID),
			Score:     1.0,
			UpdatedAt: &updated,
		})
	}
	return results, nil
}
