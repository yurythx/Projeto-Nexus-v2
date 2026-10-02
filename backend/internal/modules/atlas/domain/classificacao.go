package domain

import (
	"errors"
	"strings"
	"time"
)

// DestinacaoFinal define os destinos válidos previstos na TTDD / CONARQ.
type DestinacaoFinal string

const (
	DestinacaoGuardaPermanente DestinacaoFinal = "GUARDA_PERMANENTE"
	DestinacaoEliminacao       DestinacaoFinal = "ELIMINACAO"
)

// ClassificacaoTTDD representa uma tipologia documental e suas regras de temporalidade.
type ClassificacaoTTDD struct {
	Codigo           string          `json:"codigo"`
	Descritor        string          `json:"descritor"`
	FaseCorrenteAnos int             `json:"fase_corrente_anos"`
	FaseIntermAnos   int             `json:"fase_interm_anos"`
	DestinacaoFinal  DestinacaoFinal `json:"destinacao_final"`
	Observacoes      string          `json:"observacoes,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// Validate valida as regras mínimas da classificação documental.
func (c *ClassificacaoTTDD) Validate() error {
	if strings.TrimSpace(c.Codigo) == "" {
		return errors.New("código de classificação é obrigatório")
	}
	if strings.TrimSpace(c.Descritor) == "" {
		return errors.New("descritor da classificação é obrigatório")
	}
	if c.FaseCorrenteAnos < 0 {
		return errors.New("fase corrente deve ser maior ou igual a zero")
	}
	if c.FaseIntermAnos < 0 {
		return errors.New("fase intermediária deve ser maior ou igual a zero")
	}
	if c.DestinacaoFinal != DestinacaoGuardaPermanente && c.DestinacaoFinal != DestinacaoEliminacao {
		return errors.New("destinação final inválida (deve ser GUARDA_PERMANENTE ou ELIMINACAO)")
	}
	return nil
}
