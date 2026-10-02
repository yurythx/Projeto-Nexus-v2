package domain_test

import (
	"testing"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
)

func TestClassificacaoTTDD_Validate(t *testing.T) {
	tests := []struct {
		name    string
		c       domain.ClassificacaoTTDD
		wantErr bool
	}{
		{
			name: "Válida - Guarda Permanente",
			c: domain.ClassificacaoTTDD{
				Codigo:           "2.0.02.00.07",
				Descritor:        "Pregão Eletrônico",
				FaseCorrenteAnos: 1,
				FaseIntermAnos:   1,
				DestinacaoFinal:  domain.DestinacaoGuardaPermanente,
			},
			wantErr: false,
		},
		{
			name: "Válida - Eliminação",
			c: domain.ClassificacaoTTDD{
				Codigo:           "2.0.01.00.01",
				Descritor:        "Organogramas",
				FaseCorrenteAnos: 1,
				FaseIntermAnos:   1,
				DestinacaoFinal:  domain.DestinacaoEliminacao,
			},
			wantErr: false,
		},
		{
			name: "Código vazio",
			c: domain.ClassificacaoTTDD{
				Codigo:          "",
				Descritor:       "Teste",
				DestinacaoFinal: domain.DestinacaoEliminacao,
			},
			wantErr: true,
		},
		{
			name: "Destinação inválida",
			c: domain.ClassificacaoTTDD{
				Codigo:          "2.0.01.00.99",
				Descritor:       "Inválido",
				DestinacaoFinal: "ARQUIVAMENTO_SIMPLES",
			},
			wantErr: true,
		},
		{
			name: "Fase corrente negativa",
			c: domain.ClassificacaoTTDD{
				Codigo:           "2.0.01.00.99",
				Descritor:        "Inválido",
				FaseCorrenteAnos: -1,
				DestinacaoFinal:  domain.DestinacaoEliminacao,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ClassificacaoTTDD.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
