package ai

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
)

const (
	CanonicalRefusalMessage = "Não localizei nenhum fluxo ou procedimento homologado com base nas diretrizes oficiais da Prefeitura para a sua consulta. Por favor, verifique se a nomenclatura ou sigla do setor está correta ou solicite à Governança/CCPAD o cadastramento do fluxo."
	CanonicalSystemPrompt   = `Você é o Assistente Procedural Canônico do Módulo Atlas da Prefeitura Municipal.
Seu dever é orientar servidores públicos e cidadãos em estrita conformidade com o Padrão SEI de Processo Administrativo Eletrônico e com a Tabela de Temporalidade e Destinação de Documentos (TTDD) homologada pela CCPAD.

DIRETRIZES DE SEGURANÇA E GROUNDING:
1. Baseie sua resposta EXCLUSIVAMENTE nos dados canônicos fornecidos no CONTEXTO FACTUAL.
2. É ESTRITAMENTE PROIBIDO inferir, deduzir, supor ou inventar prazos, setores, documentos, etapas ou bases legais ausentes no contexto.
3. Se o procedimento ou documento solicitado não constar expressamente no contexto oficial, declare imediatamente a impossibilidade de informar com base na TTDD/CCPAD.
4. Responda em português formal, claro e objetivo, estruturando:
   - Identificação do Processo (Código SEI, Título e Código TTDD)
   - Sequência de Tramitação e Setores Responsáveis (com prazos SLA)
   - Peças Obrigatórias (destacando se são NATO_DIGITAL ou EXTERNO_DIGITALIZADO e tipo de assinatura)
   - Destinação Final e Prazos de Arquivamento da TTDD
   - Regras Especiais SEI (como manter aberto após remessa ou diligências)`
)

type LLMClient struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewLLMClient(endpoint, apiKey, model string, logger *slog.Logger) *LLMClient {
	if model == "" {
		model = "llama3"
	}
	return &LLMClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
		model:    model,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		logger: logger,
	}
}

// GenerateProceduralAnswer executa a geração de resposta via LLM com temperatura rígida 0.05
func (c *LLMClient) GenerateProceduralAnswer(ctx context.Context, userQuery string, workflows []domain.Workflow) (string, error) {
	if len(workflows) == 0 {
		return CanonicalRefusalMessage, nil
	}

	contextText := buildFactualContext(workflows)

	// Se endpoint não estiver configurado, gera resposta determinística baseada no contexto factual
	if c.endpoint == "" {
		return generateDeterministicFallback(workflows[0]), nil
	}

	chatURL := fmt.Sprintf("%s/v1/chat/completions", c.endpoint)
	if strings.Contains(c.endpoint, "ollama") && !strings.Contains(c.endpoint, "/v1") {
		chatURL = fmt.Sprintf("%s/api/chat", c.endpoint)
	}

	reqBody := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": CanonicalSystemPrompt},
			{"role": "user", "content": fmt.Sprintf("CONTEXTO FACTUAL HOMOLOGADO:\n%s\n\nCONSULTA DO USUÁRIO: %s", contextText, userQuery)},
		},
		"temperature": 0.05, // Fixada conforme especificação de missão
		"max_tokens":  1200,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return generateDeterministicFallback(workflows[0]), nil
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WarnContext(ctx, "falha ao conectar no contêiner de IA local, usando síntese canônica", "error", err)
		return generateDeterministicFallback(workflows[0]), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.WarnContext(ctx, "erro retornado pela LLM", "status", resp.StatusCode, "body", string(respBody))
		return generateDeterministicFallback(workflows[0]), nil
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"` // Caso seja Ollama puro
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return generateDeterministicFallback(workflows[0]), nil
	}

	if len(chatResp.Choices) > 0 && strings.TrimSpace(chatResp.Choices[0].Message.Content) != "" {
		return chatResp.Choices[0].Message.Content, nil
	}
	if strings.TrimSpace(chatResp.Message.Content) != "" {
		return chatResp.Message.Content, nil
	}

	return generateDeterministicFallback(workflows[0]), nil
}

func buildFactualContext(workflows []domain.Workflow) string {
	var sb strings.Builder
	for _, w := range workflows {
		sb.WriteString(fmt.Sprintf("\n### PROCEDIMENTO: %s (%s)\n", w.Titulo, w.CodigoProcessual))
		sb.WriteString(fmt.Sprintf("- Objetivo: %s\n", w.Objetivo))
		sb.WriteString(fmt.Sprintf("- Nível de Acesso: %s (Hipótese Legal: %s)\n", w.NivelAcesso, w.HipoteseLegal))
		if w.Classificacao != nil {
			sb.WriteString(fmt.Sprintf("- Código TTDD: %s (%s)\n", w.Classificacao.Codigo, w.Classificacao.Descritor))
			sb.WriteString(fmt.Sprintf("- Temporalidade TTDD: Fase Corrente %d anos | Fase Intermediária %d anos | Destinação Final: %s\n",
				w.Classificacao.FaseCorrenteAnos, w.Classificacao.FaseIntermAnos, w.Classificacao.DestinacaoFinal))
		}
		sb.WriteString("- ETAPAS DE TRAMITAÇÃO:\n")
		for _, e := range w.Etapas {
			sb.WriteString(fmt.Sprintf("  Etapa %d: [%s] %s | SLA: %d dias | Manter Aberto após remessa: %t\n",
				e.Ordem, e.UnidadeAdministrativa, e.NomeSetor, e.PrazoSLAEmDias, e.ManterAbertoAposRemessa))
			sb.WriteString(fmt.Sprintf("    Atribuições: %s\n", e.AtribuicoesSetor))
			if len(e.Documentos) > 0 {
				sb.WriteString("    Peças Obrigatórias:\n")
				for _, d := range e.Documentos {
					sb.WriteString(fmt.Sprintf("      * %s (Formato: %s | Assinatura: %s | Exige conferência/atesto de cópia: %t)\n",
						d.NomeDocumento, d.Formato, d.TipoAssinatura, d.ExigeConferenciaCopia))
				}
			}
			if len(e.Transicoes) > 0 {
				sb.WriteString("    Regras de Transição:\n")
				for _, t := range e.Transicoes {
					tipo := "Envio regular"
					if t.IsDevolucaoDiligencia {
						tipo = fmt.Sprintf("Devolução em Diligência (%s)", t.DescricaoDiligencia)
					}
					sb.WriteString(fmt.Sprintf("      -> Condição: %s (%s)\n", t.CondicaoTransicao, tipo))
				}
			}
		}
	}
	return sb.String()
}

func generateDeterministicFallback(w domain.Workflow) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### %s (%s)\n\n", w.Titulo, w.CodigoProcessual))
	sb.WriteString(fmt.Sprintf("**Objetivo:** %s\n\n", w.Objetivo))
	if w.Classificacao != nil {
		sb.WriteString(fmt.Sprintf("📌 **Enquadramento na TTDD:** Código `%s` — *%s*\n", w.Classificacao.Codigo, w.Classificacao.Descritor))
		sb.WriteString(fmt.Sprintf("⏳ **Temporalidade:** %d ano(s) no Arquivo Corrente e %d ano(s) no Arquivo Intermediário.\n",
			w.Classificacao.FaseCorrenteAnos, w.Classificacao.FaseIntermAnos))
		sb.WriteString(fmt.Sprintf("🏛️ **Destinação Final:** `%s`.\n\n", w.Classificacao.DestinacaoFinal))
	}

	sb.WriteString("#### Fluxo de Tramitação SEI:\n")
	for _, e := range w.Etapas {
		sb.WriteString(fmt.Sprintf("**Etapa %d: %s (%s)** — SLA: %d dias\n", e.Ordem, e.NomeSetor, e.UnidadeAdministrativa, e.PrazoSLAEmDias))
		sb.WriteString(fmt.Sprintf("- *Atribuições:* %s\n", e.AtribuicoesSetor))
		if e.ManterAbertoAposRemessa {
			sb.WriteString("- ℹ️ *Regra SEI:* A unidade mantém o processo aberto para acompanhamento após a remessa.\n")
		}
		if len(e.Documentos) > 0 {
			sb.WriteString("- *Peças Exigidas:*\n")
			for _, d := range e.Documentos {
				conferencia := ""
				if d.ExigeConferenciaCopia {
					conferencia = " (exige atesto de cópia autêntica)"
				}
				sb.WriteString(fmt.Sprintf("  • `%s` [%s, Assinatura %s]%s\n", d.NomeDocumento, d.Formato, d.TipoAssinatura, conferencia))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
