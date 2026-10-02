package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type NivelAcesso string

const (
	NivelAcessoPublico  NivelAcesso = "PUBLICO"
	NivelAcessoRestrito NivelAcesso = "RESTRITO"
	NivelAcessoSigiloso NivelAcesso = "SIGILOSO"
)

type FormatoDocumento string

const (
	FormatoNatoDigital         FormatoDocumento = "NATO_DIGITAL"
	FormatoExternoDigitalizado FormatoDocumento = "EXTERNO_DIGITALIZADO"
)

type TipoAssinatura string

const (
	TipoAssinaturaIndividual         TipoAssinatura = "INDIVIDUAL"
	TipoAssinaturaConjuntaMultinivel TipoAssinatura = "CONJUNTA_MULTINIVEL"
	TipoAssinaturaEmBloco            TipoAssinatura = "EM_BLOCO"
)

// Workflow representa a tipologia canônica de um processo administrativo SEI.
type Workflow struct {
	ID                     uuid.UUID          `json:"id"`
	TenantID               string             `json:"tenant_id"`
	CodigoProcessual       string             `json:"codigo_processual"`
	Titulo                 string             `json:"titulo"`
	Objetivo               string             `json:"objetivo"`
	PublicoAlvo            string             `json:"publico_alvo"`
	Versao                 int                `json:"versao"`
	Ativo                  bool               `json:"ativo"`
	NivelAcesso            NivelAcesso        `json:"nivel_acesso"`
	HipoteseLegal          string             `json:"hipotese_legal_restricao,omitempty"`
	CodigoTTDD             string             `json:"codigo_ttdd"`
	Classificacao          *ClassificacaoTTDD `json:"classificacao,omitempty"`
	Etapas                 []Etapa            `json:"etapas,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

// Etapa representa um nó de tramitação ou setor no percurso processual SEI.
type Etapa struct {
	ID                     uuid.UUID         `json:"id"`
	WorkflowID             uuid.UUID         `json:"workflow_id"`
	Ordem                  int               `json:"ordem"`
	UnidadeAdministrativa  string            `json:"unidade_administrativa"`
	NomeSetor              string            `json:"nome_setor"`
	AtribuicoesSetor       string            `json:"atribuicoes_setor"`
	PrazoSLAEmDias         int               `json:"prazo_sla_em_dias"`
	ManterAbertoAposRemessa bool              `json:"manter_aberto_apos_remessa"`
	Documentos             []EtapaDocumento  `json:"documentos,omitempty"`
	Transicoes             []EtapaTransicao  `json:"transicoes,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
}

// EtapaDocumento representa uma peça ou minuta obrigatória autuada na etapa.
type EtapaDocumento struct {
	ID                    uuid.UUID        `json:"id"`
	EtapaID               uuid.UUID        `json:"etapa_id"`
	NomeDocumento         string           `json:"nome_documento"`
	Obrigatorio           bool             `json:"obrigatorio"`
	Formato               FormatoDocumento `json:"formato"`
	TipoAssinatura        TipoAssinatura   `json:"tipo_assinatura"`
	ExigeConferenciaCopia bool             `json:"exige_conferencia_copia"`
	ModeloMinutaPadraoURL string           `json:"modelo_minuta_padrao_url,omitempty"`
	CreatedAt             time.Time        `json:"created_at"`
}

// EtapaTransicao define a regra de envio para o próximo setor ou trilha de diligência.
type EtapaTransicao struct {
	ID                    uuid.UUID `json:"id"`
	OrigemEtapaID         uuid.UUID `json:"origem_etapa_id"`
	DestinoEtapaID        uuid.UUID `json:"destino_etapa_id"`
	CondicaoTransicao     string    `json:"condicao_transicao"`
	IsDevolucaoDiligencia bool      `json:"is_devolucao_diligencia"`
	DescricaoDiligencia   string    `json:"descricao_diligencia,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

func (w *Workflow) Validate() error {
	if strings.TrimSpace(w.CodigoProcessual) == "" {
		return errors.New("código processual é obrigatório")
	}
	if strings.TrimSpace(w.Titulo) == "" {
		return errors.New("título do workflow é obrigatório")
	}
	if strings.TrimSpace(w.Objetivo) == "" {
		return errors.New("objetivo do workflow é obrigatório")
	}
	if strings.TrimSpace(w.CodigoTTDD) == "" {
		return errors.New("código TTDD de temporalidade é obrigatório")
	}
	if w.NivelAcesso != NivelAcessoPublico && w.NivelAcesso != NivelAcessoRestrito && w.NivelAcesso != NivelAcessoSigiloso {
		return errors.New("nível de acesso inválido")
	}
	return nil
}
