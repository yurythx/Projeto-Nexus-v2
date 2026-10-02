package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
)

type ListTTDDRequest struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ListTTDDResponse struct {
	Total int                        `json:"total"`
	Items []domain.ClassificacaoTTDD `json:"items"`
}

type ListWorkflowsRequest struct {
	TenantID   string `json:"tenant_id"`
	Query      string `json:"query"`
	CodigoTTDD string `json:"codigo_ttdd"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

type ListWorkflowsResponse struct {
	Total int               `json:"total"`
	Items []domain.Workflow `json:"items"`
}

type ProceduralChatRequest struct {
	TenantID string `json:"tenant_id"`
	Query    string `json:"query"`
}

type ProceduralChatResponse struct {
	Answer       string            `json:"answer"`
	Score        float64           `json:"score"`
	Refused      bool              `json:"refused"`
	Workflows    []domain.Workflow `json:"workflows,omitempty"`
	GeneratedAt  time.Time         `json:"generated_at"`
}

type CreateWorkflowRequest struct {
	TenantID         string                 `json:"tenant_id"`
	CodigoProcessual string                 `json:"codigo_processual"`
	Titulo           string                 `json:"titulo"`
	Objetivo         string                 `json:"objetivo"`
	PublicoAlvo      string                 `json:"publico_alvo"`
	NivelAcesso      domain.NivelAcesso     `json:"nivel_acesso"`
	HipoteseLegal    string                 `json:"hipotese_legal_restricao,omitempty"`
	CodigoTTDD       string                 `json:"codigo_ttdd"`
	Etapas           []CreateEtapaRequest   `json:"etapas"`
}

type CreateEtapaRequest struct {
	Ordem                   int                          `json:"ordem"`
	UnidadeAdministrativa   string                       `json:"unidade_administrativa"`
	NomeSetor               string                       `json:"nome_setor"`
	AtribuicoesSetor        string                       `json:"atribuicoes_setor"`
	PrazoSLAEmDias          int                          `json:"prazo_sla_em_dias"`
	ManterAbertoAposRemessa bool                         `json:"manter_aberto_apos_remessa"`
	Documentos              []CreateDocumentoRequest     `json:"documentos,omitempty"`
	Transicoes              []CreateTransicaoRequest     `json:"transicoes,omitempty"`
}

type CreateDocumentoRequest struct {
	NomeDocumento         string                  `json:"nome_documento"`
	Obrigatorio           bool                    `json:"obrigatorio"`
	Formato               domain.FormatoDocumento `json:"formato"`
	TipoAssinatura        domain.TipoAssinatura   `json:"tipo_assinatura"`
	ExigeConferenciaCopia bool                    `json:"exige_conferencia_copia"`
	ModeloMinutaPadraoURL string                  `json:"modelo_minuta_padrao_url,omitempty"`
}

type CreateTransicaoRequest struct {
	DestinoEtapaOrdem     int    `json:"destino_etapa_ordem"`
	CondicaoTransicao     string `json:"condicao_transicao"`
	IsDevolucaoDiligencia bool   `json:"is_devolucao_diligencia"`
	DescricaoDiligencia   string `json:"descricao_diligencia,omitempty"`
	DestinoEtapaID        uuid.UUID `json:"-"`
}
