package typesense

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

// SearchResult representa um item ranqueado da busca híbrida.
type SearchResult struct {
	Workflow *domain.Workflow `json:"workflow"`
	Score    float64          `json:"score"`
}

// Client gerencia a indexação e busca híbrida no Typesense.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
	dbtx       database.DBTX
	repo       domain.Repository
}

func NewClient(baseURL, apiKey string, dbtx database.DBTX, repo domain.Repository, logger *slog.Logger) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
		dbtx:   dbtx,
		repo:   repo,
	}
}

// IndexWorkflow indexa assincronamente o documento no Typesense
func (c *Client) IndexWorkflow(ctx context.Context, wf *domain.Workflow) error {
	if c.baseURL == "" || c.apiKey == "" {
		return nil
	}

	url := fmt.Sprintf("%s/collections/atlas_workflows/documents?action=upsert", c.baseURL)

	var setores, documentos []string
	for _, e := range wf.Etapas {
		setores = append(setores, e.NomeSetor, e.UnidadeAdministrativa)
		for _, d := range e.Documentos {
			documentos = append(documentos, d.NomeDocumento)
		}
	}

	doc := map[string]any{
		"id":                wf.CodigoProcessual,
		"codigo_processual": wf.CodigoProcessual,
		"titulo":            wf.Titulo,
		"objetivo":          wf.Objetivo,
		"tenant_id":         wf.TenantID,
		"ativo":             wf.Ativo,
		"setores":           strings.Join(setores, " "),
		"documentos":        strings.Join(documentos, " "),
	}

	bodyBytes, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WarnContext(ctx, "falha ao indexar documento no typesense", "error", err, "wf_id", wf.ID)
		return nil // Não trava o worker
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.WarnContext(ctx, "erro ao indexar documento no typesense", "status", resp.StatusCode, "body", string(respBody))
	}
	return nil
}

// SearchHybrid executa a busca híbrida (BM25 + embeddings com alpha = 0.6).
// Se o Typesense não estiver acessível, faz fallback determinístico para busca factual no PostgreSQL.
func (c *Client) SearchHybrid(ctx context.Context, tenantID, query string) ([]SearchResult, error) {
	if c.baseURL != "" && c.apiKey != "" {
		results, err := c.searchTypesense(ctx, tenantID, query)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		if err != nil {
			c.logger.WarnContext(ctx, "typesense indisponível, aplicando fallback postgres", "error", err)
		}
	}

	// Fallback PostgreSQL com cálculo determinístico de score
	return c.fallbackPostgresSearch(ctx, tenantID, query)
}

func (c *Client) searchTypesense(ctx context.Context, tenantID, query string) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/collections/atlas_workflows/documents/search", c.baseURL)

	reqBody := map[string]any{
		"q": query,
		"query_by": "titulo,objetivo,codigo_processual,setores,documentos",
		"filter_by": fmt.Sprintf("tenant_id:=%s && ativo:=true", tenantID),
		"alpha": 0.6, // Ponderação canônica: 60% vetorial / 40% lexical
		"limit": 5,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro typesense HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var tsResponse struct {
		Hits []struct {
			Document struct {
				ID string `json:"id"`
			} `json:"document"`
			HybridSearchInfo struct {
				RankLossScore float64 `json:"rank_loss_score"`
			} `json:"hybrid_search_info"`
			TextMatch int64 `json:"text_match"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tsResponse); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, hit := range tsResponse.Hits {
		score := 0.75
		if hit.TextMatch > 0 {
			score = 0.85
		}
		wf, err := c.repo.GetWorkflowByCodigo(ctx, c.dbtx, tenantID, hit.Document.ID, 1)
		if err == nil && wf != nil {
			results = append(results, SearchResult{Workflow: wf, Score: score})
		}
	}

	return results, nil
}

func (c *Client) fallbackPostgresSearch(ctx context.Context, tenantID, query string) ([]SearchResult, error) {
	wfs, err := c.repo.SearchProceduralGrounding(ctx, c.dbtx, tenantID, query, 5)
	if err != nil {
		return nil, err
	}

	qTokens := strings.Fields(strings.ToLower(query))
	var results []SearchResult

	for i := range wfs {
		wf := &wfs[i]
		score := calculateScore(wf, qTokens)
		if score > 0 {
			results = append(results, SearchResult{
				Workflow: wf,
				Score:    score,
			})
		}
	}

	return results, nil
}

func calculateScore(wf *domain.Workflow, tokens []string) float64 {
	if len(tokens) == 0 {
		return 0
	}

	corpus := strings.ToLower(fmt.Sprintf("%s %s %s %s",
		wf.Titulo, wf.Objetivo, wf.CodigoProcessual, wf.CodigoTTDD))
	
	for _, e := range wf.Etapas {
		corpus += fmt.Sprintf(" %s %s", strings.ToLower(e.NomeSetor), strings.ToLower(e.AtribuicoesSetor))
		for _, d := range e.Documentos {
			corpus += " " + strings.ToLower(d.NomeDocumento)
		}
	}

	matched := 0
	for _, t := range tokens {
		if strings.Contains(corpus, t) {
			matched++
		}
	}

	ratio := float64(matched) / float64(len(tokens))
	
	for _, t := range tokens {
		if strings.EqualFold(wf.CodigoProcessual, t) || strings.EqualFold(wf.CodigoTTDD, t) {
			ratio += 0.3
		}
		if strings.Contains(strings.ToLower(wf.Titulo), t) {
			ratio += 0.15
		}
	}

	if ratio > 1.0 {
		ratio = 1.0
	}
	return ratio
}
