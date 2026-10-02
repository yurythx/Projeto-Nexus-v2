package domain_test

import (
	"testing"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
)

func TestWorkflow_Validate(t *testing.T) {
	tests := []struct {
		name    string
		w       domain.Workflow
		wantErr bool
	}{
		{
			name: "Válido",
			w: domain.Workflow{
				CodigoProcessual: "ADM.LIC.001",
				Titulo:           "Pregão Eletrônico",
				Objetivo:         "Aquisição de Bens",
				CodigoTTDD:       "2.0.02.00.07",
				NivelAcesso:      domain.NivelAcessoPublico,
			},
			wantErr: false,
		},
		{
			name: "Sem código processual",
			w: domain.Workflow{
				CodigoProcessual: "",
				Titulo:           "Pregão Eletrônico",
				Objetivo:         "Aquisição de Bens",
				CodigoTTDD:       "2.0.02.00.07",
				NivelAcesso:      domain.NivelAcessoPublico,
			},
			wantErr: true,
		},
		{
			name: "Sem título",
			w: domain.Workflow{
				CodigoProcessual: "ADM.LIC.001",
				Titulo:           "",
				Objetivo:         "Aquisição de Bens",
				CodigoTTDD:       "2.0.02.00.07",
				NivelAcesso:      domain.NivelAcessoPublico,
			},
			wantErr: true,
		},
		{
			name: "Sem código TTDD",
			w: domain.Workflow{
				CodigoProcessual: "ADM.LIC.001",
				Titulo:           "Pregão",
				Objetivo:         "Aquisição",
				CodigoTTDD:       "",
				NivelAcesso:      domain.NivelAcessoPublico,
			},
			wantErr: true,
		},
		{
			name: "Nível de acesso inválido",
			w: domain.Workflow{
				CodigoProcessual: "ADM.LIC.001",
				Titulo:           "Pregão",
				Objetivo:         "Aquisição",
				CodigoTTDD:       "2.0.02.00.07",
				NivelAcesso:      "SECRETO_CONFIDENCIAL",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.w.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Workflow.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
