// Package domain contém as regras do Atlas: a Tabela de Temporalidade e
// Destinação de Documentos (TTDD/CCPAD) e os procedimentos canônicos de
// processo administrativo eletrônico (padrão SEI) — etapas, peças
// exigidas e regras de transição.
package domain

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nexus/internal/domain/pagination"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

// Erros de domínio (traduzidos para HTTP em application.MapError).
var (
	ErrNotFound     = errors.New("atlas: procedimento não encontrado")
	ErrTTDDNotFound = errors.New("atlas: classificação TTDD não encontrada")
	ErrDuplicate    = errors.New("atlas: já existe procedimento com este código e versão")
)

// InvalidError é uma violação de regra de cadastro (422 com a mensagem).
type InvalidError struct{ Msg string }

func (e InvalidError) Error() string { return e.Msg }

func invalid(format string, a ...any) error { return InvalidError{Msg: fmt.Sprintf(format, a...)} }

// DestinacaoFinal é o destino do documento ao fim da guarda (CONARQ).
type DestinacaoFinal string

const (
	DestinacaoGuardaPermanente DestinacaoFinal = "GUARDA_PERMANENTE"
	DestinacaoEliminacao       DestinacaoFinal = "ELIMINACAO"
)

// NivelAcesso é o nível de acesso SEI sugerido para os processos do tipo.
// Descreve o processo que nasce do procedimento — o roteiro em si é
// informação pública (transparência ativa, LAI art. 8º).
type NivelAcesso string

const (
	NivelPublico  NivelAcesso = "PUBLICO"
	NivelRestrito NivelAcesso = "RESTRITO"
	NivelSigiloso NivelAcesso = "SIGILOSO"
)

// FormatoDocumento distingue o documento nascido no sistema do digitalizado.
type FormatoDocumento string

const (
	FormatoNatoDigital         FormatoDocumento = "NATO_DIGITAL"
	FormatoExternoDigitalizado FormatoDocumento = "EXTERNO_DIGITALIZADO"
)

// TipoAssinatura é a forma de assinatura exigida da peça.
type TipoAssinatura string

const (
	AssinaturaIndividual         TipoAssinatura = "INDIVIDUAL"
	AssinaturaConjuntaMultinivel TipoAssinatura = "CONJUNTA_MULTINIVEL"
	AssinaturaEmBloco            TipoAssinatura = "EM_BLOCO"
)

// ClassificacaoTTDD é uma linha da Tabela de Temporalidade.
type ClassificacaoTTDD struct {
	Codigo           string          `json:"codigo"`
	Descritor        string          `json:"descritor"`
	FaseCorrenteAnos int             `json:"fase_corrente_anos"`
	FaseIntermAnos   int             `json:"fase_interm_anos"`
	DestinacaoFinal  DestinacaoFinal `json:"destinacao_final"`
	Observacoes      string          `json:"observacoes"`
	CreatedAt        time.Time       `json:"created_at"`
}

// Workflow é o procedimento canônico de um tipo de processo.
type Workflow struct {
	ID               uuid.UUID          `json:"id"`
	CodigoProcessual string             `json:"codigo_processual"`
	Titulo           string             `json:"titulo"`
	Objetivo         string             `json:"objetivo"`
	PublicoAlvo      string             `json:"publico_alvo"`
	Versao           int                `json:"versao"`
	Ativo            bool               `json:"ativo"`
	NivelAcesso      NivelAcesso        `json:"nivel_acesso"`
	HipoteseLegal    string             `json:"hipotese_legal_restricao"`
	CodigoTTDD       string             `json:"codigo_ttdd"`
	Classificacao    *ClassificacaoTTDD `json:"classificacao,omitempty"`
	TotalEtapas      int                `json:"total_etapas"`
	Etapas           []Etapa            `json:"etapas"`
	CreatedBy        *uuid.UUID         `json:"created_by"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// Etapa é um nó de tramitação (setor) do percurso.
type Etapa struct {
	ID                      uuid.UUID        `json:"id"`
	Ordem                   int              `json:"ordem"`
	UnidadeAdministrativa   string           `json:"unidade_administrativa"`
	NomeSetor               string           `json:"nome_setor"`
	AtribuicoesSetor        string           `json:"atribuicoes_setor"`
	PrazoSLAEmDias          int              `json:"prazo_sla_em_dias"`
	ManterAbertoAposRemessa bool             `json:"manter_aberto_apos_remessa"`
	Documentos              []EtapaDocumento `json:"documentos"`
	Transicoes              []EtapaTransicao `json:"transicoes"`
}

// EtapaDocumento é uma peça exigida na etapa.
type EtapaDocumento struct {
	ID                    uuid.UUID        `json:"id"`
	NomeDocumento         string           `json:"nome_documento"`
	Obrigatorio           bool             `json:"obrigatorio"`
	Formato               FormatoDocumento `json:"formato"`
	TipoAssinatura        TipoAssinatura   `json:"tipo_assinatura"`
	ExigeConferenciaCopia bool             `json:"exige_conferencia_copia"`
	ModeloMinutaPadraoURL string           `json:"modelo_minuta_padrao_url"`
}

// EtapaTransicao leva o processo da etapa a outra (envio regular ou
// devolução em diligência). O destino é identificado pela ordem.
type EtapaTransicao struct {
	ID                    uuid.UUID `json:"id"`
	DestinoOrdem          int       `json:"destino_ordem"`
	CondicaoTransicao     string    `json:"condicao_transicao"`
	IsDevolucaoDiligencia bool      `json:"is_devolucao_diligencia"`
	DescricaoDiligencia   string    `json:"descricao_diligencia"`
}

// Limites de cadastro (espelham as colunas da migration 000132).
const (
	MaxEtapas      = 50
	MaxDocumentos  = 30
	MaxTransicoes  = 10
	MaxPrazoSLA    = 3650
	maxCodigo      = 64
	maxTitulo      = 255
	maxSigla       = 64
	maxNomeSetor   = 150
	maxNomeDoc     = 150
	maxCondicao    = 255
	maxHipotese    = 255
	maxObjetivo    = 5000
	maxAtribuicoes = 5000
)

var codigoProcessual = regexp.MustCompile(`^[A-Z0-9]+(\.[A-Z0-9]+)*$`)

// Normalize apara os textos e põe o código em caixa alta.
func (w *Workflow) Normalize() {
	w.CodigoProcessual = strings.ToUpper(strings.TrimSpace(w.CodigoProcessual))
	w.Titulo = strings.TrimSpace(w.Titulo)
	w.Objetivo = strings.TrimSpace(w.Objetivo)
	w.PublicoAlvo = strings.TrimSpace(w.PublicoAlvo)
	w.HipoteseLegal = strings.TrimSpace(w.HipoteseLegal)
	w.CodigoTTDD = strings.TrimSpace(w.CodigoTTDD)
	for i := range w.Etapas {
		e := &w.Etapas[i]
		e.UnidadeAdministrativa = strings.ToUpper(strings.TrimSpace(e.UnidadeAdministrativa))
		e.NomeSetor = strings.TrimSpace(e.NomeSetor)
		e.AtribuicoesSetor = strings.TrimSpace(e.AtribuicoesSetor)
		for j := range e.Documentos {
			d := &e.Documentos[j]
			d.NomeDocumento = strings.TrimSpace(d.NomeDocumento)
			d.ModeloMinutaPadraoURL = strings.TrimSpace(d.ModeloMinutaPadraoURL)
		}
		for j := range e.Transicoes {
			t := &e.Transicoes[j]
			t.CondicaoTransicao = strings.TrimSpace(t.CondicaoTransicao)
			t.DescricaoDiligencia = strings.TrimSpace(t.DescricaoDiligencia)
		}
	}
}

func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// Validate aplica as regras de cadastro do procedimento (chame Normalize
// antes). A existência do código TTDD é conferida na transação.
func (w Workflow) Validate() error {
	switch {
	case w.CodigoProcessual == "" || tooLong(w.CodigoProcessual, maxCodigo) || !codigoProcessual.MatchString(w.CodigoProcessual):
		return invalid("código processual inválido: use segmentos alfanuméricos separados por ponto (ex.: ADM.LIC.001)")
	case w.Titulo == "" || tooLong(w.Titulo, maxTitulo):
		return invalid("título obrigatório (até %d caracteres)", maxTitulo)
	case w.Objetivo == "" || tooLong(w.Objetivo, maxObjetivo):
		return invalid("objetivo obrigatório (até %d caracteres)", maxObjetivo)
	case w.PublicoAlvo == "" || tooLong(w.PublicoAlvo, maxTitulo):
		return invalid("público-alvo obrigatório (até %d caracteres)", maxTitulo)
	case w.CodigoTTDD == "":
		return invalid("código TTDD obrigatório")
	case w.Versao < 1:
		return invalid("versão deve ser maior ou igual a 1")
	}
	switch w.NivelAcesso {
	case NivelPublico:
	case NivelRestrito, NivelSigiloso:
		// SEI/LAI: restrição de acesso sempre aponta a hipótese legal.
		if w.HipoteseLegal == "" {
			return invalid("nível de acesso %s exige a hipótese legal da restrição", w.NivelAcesso)
		}
	default:
		return invalid("nível de acesso inválido (PUBLICO, RESTRITO ou SIGILOSO)")
	}
	if tooLong(w.HipoteseLegal, maxHipotese) {
		return invalid("hipótese legal até %d caracteres", maxHipotese)
	}
	if len(w.Etapas) == 0 || len(w.Etapas) > MaxEtapas {
		return invalid("o procedimento precisa de 1 a %d etapas", MaxEtapas)
	}
	ordens := map[int]bool{}
	for _, e := range w.Etapas {
		if e.Ordem < 1 {
			return invalid("ordem da etapa deve ser maior ou igual a 1")
		}
		if ordens[e.Ordem] {
			return invalid("ordem de etapa repetida: %d", e.Ordem)
		}
		ordens[e.Ordem] = true
	}
	for _, e := range w.Etapas {
		if err := e.validate(ordens); err != nil {
			return err
		}
	}
	return nil
}

func (e Etapa) validate(ordens map[int]bool) error {
	switch {
	case e.UnidadeAdministrativa == "" || tooLong(e.UnidadeAdministrativa, maxSigla):
		return invalid("etapa %d: sigla da unidade obrigatória (até %d caracteres)", e.Ordem, maxSigla)
	case e.NomeSetor == "" || tooLong(e.NomeSetor, maxNomeSetor):
		return invalid("etapa %d: nome do setor obrigatório (até %d caracteres)", e.Ordem, maxNomeSetor)
	case e.AtribuicoesSetor == "" || tooLong(e.AtribuicoesSetor, maxAtribuicoes):
		return invalid("etapa %d: atribuições obrigatórias (até %d caracteres)", e.Ordem, maxAtribuicoes)
	case e.PrazoSLAEmDias < 0 || e.PrazoSLAEmDias > MaxPrazoSLA:
		return invalid("etapa %d: prazo SLA entre 0 e %d dias", e.Ordem, MaxPrazoSLA)
	case len(e.Documentos) > MaxDocumentos:
		return invalid("etapa %d: no máximo %d peças", e.Ordem, MaxDocumentos)
	case len(e.Transicoes) > MaxTransicoes:
		return invalid("etapa %d: no máximo %d transições", e.Ordem, MaxTransicoes)
	}
	for _, d := range e.Documentos {
		if d.NomeDocumento == "" || tooLong(d.NomeDocumento, maxNomeDoc) {
			return invalid("etapa %d: nome da peça obrigatório (até %d caracteres)", e.Ordem, maxNomeDoc)
		}
		if d.Formato != FormatoNatoDigital && d.Formato != FormatoExternoDigitalizado {
			return invalid("etapa %d: formato da peça inválido (NATO_DIGITAL ou EXTERNO_DIGITALIZADO)", e.Ordem)
		}
		switch d.TipoAssinatura {
		case AssinaturaIndividual, AssinaturaConjuntaMultinivel, AssinaturaEmBloco:
		default:
			return invalid("etapa %d: tipo de assinatura inválido (INDIVIDUAL, CONJUNTA_MULTINIVEL ou EM_BLOCO)", e.Ordem)
		}
		if d.ModeloMinutaPadraoURL != "" && !httpURL(d.ModeloMinutaPadraoURL) {
			return invalid("etapa %d: o modelo de minuta precisa ser um endereço http(s) completo", e.Ordem)
		}
	}
	for _, t := range e.Transicoes {
		switch {
		case !ordens[t.DestinoOrdem]:
			return invalid("etapa %d: a transição aponta para uma etapa inexistente (%d)", e.Ordem, t.DestinoOrdem)
		case t.DestinoOrdem == e.Ordem:
			return invalid("etapa %d: a transição não pode apontar para a própria etapa", e.Ordem)
		case t.CondicaoTransicao == "" || tooLong(t.CondicaoTransicao, maxCondicao):
			return invalid("etapa %d: condição da transição obrigatória (até %d caracteres)", e.Ordem, maxCondicao)
		case t.IsDevolucaoDiligencia && t.DescricaoDiligencia == "":
			return invalid("etapa %d: devolução em diligência exige a descrição da diligência", e.Ordem)
		}
	}
	return nil
}

// httpURL aceita só endereço absoluto http(s) sem credenciais — o link é
// exibido como <a href>, e "javascript:" viraria XSS armazenado.
func httpURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}

// Filter filtra a listagem de procedimentos.
type Filter struct {
	Query      string
	CodigoTTDD string
	// IncluirInativos: só a gestão (atlas:manage) vê os desativados.
	IncluirInativos bool
}

// Repository é a persistência do Atlas.
type Repository interface {
	ListTTDD(ctx context.Context, db database.DBTX, query string, p pagination.Params) ([]ClassificacaoTTDD, int64, error)
	GetTTDD(ctx context.Context, db database.DBTX, codigo string) (ClassificacaoTTDD, error)
	// LockTTDD confirma que a classificação existe e a trava (FOR SHARE)
	// até o fim da transação de cadastro.
	LockTTDD(ctx context.Context, db database.DBTX, codigo string) (bool, error)

	List(ctx context.Context, db database.DBTX, f Filter, p pagination.Params) ([]Workflow, int64, error)
	// Get devolve o procedimento completo (etapas, peças e transições).
	Get(ctx context.Context, db database.DBTX, id uuid.UUID, forUpdate bool) (Workflow, error)
	// Insert grava o procedimento com etapas, peças e transições.
	Insert(ctx context.Context, db database.DBTX, w Workflow) error
	SetAtivo(ctx context.Context, db database.DBTX, id uuid.UUID, ativo bool) error
	// Search é a busca full-text (ativos) com o rank de cada resultado.
	Search(ctx context.Context, db database.DBTX, query string, limit int) ([]Workflow, []float64, error)
	// Candidatos devolve os procedimentos ativos (completos) que casam com
	// algum termo da pergunta — base do grounding do assistente.
	Candidatos(ctx context.Context, db database.DBTX, pergunta string, limit int) ([]Workflow, error)
}

// Assistente redige a orientação a partir EXCLUSIVAMENTE dos procedimentos
// fornecidos (porta implementada por um LLM compatível com a API OpenAI).
type Assistente interface {
	Responder(ctx context.Context, pergunta string, contexto []Workflow) (string, error)
}
