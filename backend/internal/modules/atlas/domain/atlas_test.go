package domain

import (
	"errors"
	"strings"
	"testing"
)

func validWorkflow() Workflow {
	return Workflow{
		CodigoProcessual: " adm.lic.001 ", Titulo: "Pregão Eletrônico", Objetivo: "Aquisição de bens comuns",
		PublicoAlvo: "Secretarias", Versao: 1, NivelAcesso: NivelPublico, CodigoTTDD: "2.0.02.00.07",
		Etapas: []Etapa{
			{Ordem: 1, UnidadeAdministrativa: "sec/dem", NomeSetor: "Demandante", AtribuicoesSetor: "Elaborar o DFD", PrazoSLAEmDias: 5,
				Documentos: []EtapaDocumento{{NomeDocumento: "DFD", Obrigatorio: true, Formato: FormatoNatoDigital, TipoAssinatura: AssinaturaIndividual}},
				Transicoes: []EtapaTransicao{{DestinoOrdem: 2, CondicaoTransicao: "DFD aprovado"}}},
			{Ordem: 2, UnidadeAdministrativa: "SEMAD/LIC", NomeSetor: "Licitações", AtribuicoesSetor: "Conduzir o certame",
				Transicoes: []EtapaTransicao{{DestinoOrdem: 1, CondicaoTransicao: "Pendência", IsDevolucaoDiligencia: true, DescricaoDiligencia: "Completar o DFD"}}},
		},
	}
}

func TestNormalize(t *testing.T) {
	w := validWorkflow()
	w.Normalize()
	if w.CodigoProcessual != "ADM.LIC.001" || w.Etapas[0].UnidadeAdministrativa != "SEC/DEM" {
		t.Fatalf("normalização: %q %q", w.CodigoProcessual, w.Etapas[0].UnidadeAdministrativa)
	}
}

func TestValidate(t *testing.T) {
	ok := validWorkflow()
	ok.Normalize()
	if err := ok.Validate(); err != nil {
		t.Fatalf("procedimento válido recusado: %v", err)
	}
	cases := map[string]func(w *Workflow){
		"código com espaço":            func(w *Workflow) { w.CodigoProcessual = "ADM LIC" },
		"sem título":                   func(w *Workflow) { w.Titulo = "" },
		"sem objetivo":                 func(w *Workflow) { w.Objetivo = "" },
		"sem público-alvo":             func(w *Workflow) { w.PublicoAlvo = "" },
		"sem TTDD":                     func(w *Workflow) { w.CodigoTTDD = "" },
		"versão zero":                  func(w *Workflow) { w.Versao = 0 },
		"nível inválido":               func(w *Workflow) { w.NivelAcesso = "SECRETO" },
		"restrito sem hipótese":        func(w *Workflow) { w.NivelAcesso = NivelRestrito },
		"sem etapas":                   func(w *Workflow) { w.Etapas = nil },
		"ordem repetida":               func(w *Workflow) { w.Etapas[1].Ordem = 1 },
		"ordem zero":                   func(w *Workflow) { w.Etapas[0].Ordem = 0 },
		"etapa sem setor":              func(w *Workflow) { w.Etapas[0].NomeSetor = "" },
		"prazo negativo":               func(w *Workflow) { w.Etapas[0].PrazoSLAEmDias = -1 },
		"formato inválido":             func(w *Workflow) { w.Etapas[0].Documentos[0].Formato = "PAPEL" },
		"assinatura inválida":          func(w *Workflow) { w.Etapas[0].Documentos[0].TipoAssinatura = "NENHUMA" },
		"minuta javascript:":           func(w *Workflow) { w.Etapas[0].Documentos[0].ModeloMinutaPadraoURL = "javascript:alert(1)" },
		"transição para etapa ausente": func(w *Workflow) { w.Etapas[0].Transicoes[0].DestinoOrdem = 9 },
		"transição para si mesma":      func(w *Workflow) { w.Etapas[0].Transicoes[0].DestinoOrdem = 1 },
		"diligência sem descrição":     func(w *Workflow) { w.Etapas[1].Transicoes[0].DescricaoDiligencia = "" },
		"hipótese longa": func(w *Workflow) {
			w.NivelAcesso, w.HipoteseLegal = NivelRestrito, strings.Repeat("x", 256)
		},
		"etapa sem sigla":        func(w *Workflow) { w.Etapas[0].UnidadeAdministrativa = "" },
		"etapa sem atribuições":  func(w *Workflow) { w.Etapas[0].AtribuicoesSetor = "" },
		"peças demais":           func(w *Workflow) { w.Etapas[0].Documentos = make([]EtapaDocumento, MaxDocumentos+1) },
		"transições demais":      func(w *Workflow) { w.Etapas[0].Transicoes = make([]EtapaTransicao, MaxTransicoes+1) },
		"peça sem nome":          func(w *Workflow) { w.Etapas[0].Documentos[0].NomeDocumento = "" },
		"transição sem condição": func(w *Workflow) { w.Etapas[0].Transicoes[0].CondicaoTransicao = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			w := validWorkflow()
			w.Normalize()
			mutate(&w)
			var inv InvalidError
			if err := w.Validate(); !errors.As(err, &inv) || err.Error() == "" {
				t.Fatalf("esperado InvalidError, veio %v", err)
			}
		})
	}
	restrito := validWorkflow()
	restrito.Normalize()
	restrito.NivelAcesso, restrito.HipoteseLegal = NivelRestrito, "LAI art. 31"
	restrito.Etapas[0].Documentos[0].ModeloMinutaPadraoURL = "https://modelos.exemplo.gov.br/dfd.docx"
	if err := restrito.Validate(); err != nil {
		t.Fatalf("restrito com hipótese legal e minuta https: %v", err)
	}
}

func TestTermos(t *testing.T) {
	got := strings.Join(Termos("Como tramitar um processo de Pregão Eletrônico? ADM.LIC.001."), ",")
	if got != "tramitar,processo,pregao,eletronico,adm.lic.001" {
		t.Fatalf("termos = %s", got)
	}
	if len(Termos("de que como")) != 0 {
		t.Fatal("só stopwords não gera termos")
	}
}

func TestRelevancia(t *testing.T) {
	w := validWorkflow()
	w.Normalize()
	if r := Relevancia(w, "Quais documentos para o pregão eletrônico?"); r < LimiarRelevancia {
		t.Fatalf("pergunta coberta abaixo do limiar: %.2f", r)
	}
	if r := Relevancia(w, "ADM.LIC.001"); r < LimiarRelevancia {
		t.Fatalf("código exato abaixo do limiar: %.2f", r)
	}
	if r := Relevancia(w, "licença para viagem internacional"); r >= LimiarRelevancia {
		t.Fatalf("pergunta sem relação acima do limiar: %.2f", r)
	}
	if Relevancia(w, "") != 0 || Relevancia(w, "de o a") != 0 {
		t.Fatal("pergunta vazia tem relevância zero")
	}
}

func TestSinteseCanonica(t *testing.T) {
	w := validWorkflow()
	w.Normalize()
	w.Classificacao = &ClassificacaoTTDD{Codigo: "2.0.02.00.07", Descritor: "Pregão", FaseCorrenteAnos: 1, FaseIntermAnos: 4, DestinacaoFinal: DestinacaoGuardaPermanente}
	w.Etapas[0].ManterAbertoAposRemessa = true
	w.Etapas[0].Documentos = append(w.Etapas[0].Documentos, EtapaDocumento{NomeDocumento: "Cópia do RG", Formato: FormatoExternoDigitalizado,
		TipoAssinatura: AssinaturaIndividual, ExigeConferenciaCopia: true})
	if Relevancia(w, "pregão") == 0 {
		t.Fatal("descritor da TTDD entra no corpus")
	}
	s := SinteseCanonica(w)
	for _, want := range []string{"ADM.LIC.001", "Pregão", "GUARDA_PERMANENTE", "1. Demandante", "DFD", "segue para a etapa 2", "devolve à etapa 1 em diligência", "mantém o processo aberto", "peça opcional", "exige conferência da cópia"} {
		if !strings.Contains(s, want) {
			t.Fatalf("síntese sem %q:\n%s", want, s)
		}
	}
}

func TestSinteseRestrito(t *testing.T) {
	w := validWorkflow()
	w.NivelAcesso, w.HipoteseLegal = NivelRestrito, "LAI art. 31"
	if !strings.Contains(SinteseCanonica(w), "hipótese legal: LAI art. 31") {
		t.Fatal("síntese de procedimento restrito informa a hipótese legal")
	}
}
