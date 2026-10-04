package domain

import "github.com/google/uuid"

// Painel de cobertura do Atlas (ADR 025): quanto da TTDD já tem fluxo
// homologado e modelo, e a lista de trabalho de quem alimenta o Atlas.

// CoberturaOrgao resume as séries vigentes de um órgão.
type CoberturaOrgao struct {
	Prefixo               string `json:"prefixo"`
	Nome                  string `json:"nome"`
	Series                int    `json:"series"`
	SeriesComProcedimento int    `json:"series_com_procedimento"`
	SeriesComModelo       int    `json:"series_com_modelo"`
}

// PecaSemModelo é uma peça de procedimento em vigor sem modelo para baixar
// (nem próprio nem da série).
type PecaSemModelo struct {
	WorkflowID       uuid.UUID `json:"workflow_id"`
	CodigoProcessual string    `json:"codigo_processual"`
	Titulo           string    `json:"titulo"`
	Etapa            int       `json:"etapa"`
	Peca             string    `json:"peca"`
}

// ModeloSemUso é um modelo ativo que nenhuma peça nem série usa.
type ModeloSemUso struct {
	ID   uuid.UUID `json:"id"`
	Nome string    `json:"nome"`
}

// TotaisCobertura somam o painel.
type TotaisCobertura struct {
	Series                int `json:"series"`
	SeriesComProcedimento int `json:"series_com_procedimento"`
	SeriesComModelo       int `json:"series_com_modelo"`
	Procedimentos         int `json:"procedimentos"`
	Pecas                 int `json:"pecas"`
	PecasComModelo        int `json:"pecas_com_modelo"`
	Modelos               int `json:"modelos"`
	ModelosSemUso         int `json:"modelos_sem_uso"`
	// Validação (ADR 028): procedimentos ainda não publicados.
	Rascunhos   int `json:"rascunhos"`
	EmValidacao int `json:"em_validacao"`
}

// Cobertura é o painel.
type Cobertura struct {
	Totais         TotaisCobertura  `json:"totais"`
	Orgaos         []CoberturaOrgao `json:"orgaos"`
	PecasSemModelo []PecaSemModelo  `json:"pecas_sem_modelo"`
	ModelosSemUso  []ModeloSemUso   `json:"modelos_sem_uso"`
}

// Somar preenche os totais das séries a partir dos órgãos e os de peças e
// modelos a partir das listas.
func (c *Cobertura) Somar(procedimentos, pecas, modelos int) {
	t := TotaisCobertura{Procedimentos: procedimentos, Pecas: pecas, PecasComModelo: pecas - len(c.PecasSemModelo),
		Modelos: modelos, ModelosSemUso: len(c.ModelosSemUso)}
	for _, o := range c.Orgaos {
		t.Series += o.Series
		t.SeriesComProcedimento += o.SeriesComProcedimento
		t.SeriesComModelo += o.SeriesComModelo
	}
	c.Totais = t
}
