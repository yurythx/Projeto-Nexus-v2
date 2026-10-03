package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
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
	w.Classificacao = &ClassificacaoTTDD{Codigo: "2.0.02.00.07", Descritor: "Pregão", FaseCorrenteAnos: ptr(1), FaseIntermAnos: ptr(4), DestinacaoFinal: ptr(DestinacaoGuardaPermanente)}
	w.Etapas[0].ManterAbertoAposRemessa = true
	w.Etapas[0].Documentos = append(w.Etapas[0].Documentos, EtapaDocumento{NomeDocumento: "Cópia do RG", Formato: FormatoExternoDigitalizado,
		TipoAssinatura: AssinaturaIndividual, ExigeConferenciaCopia: true})
	if Relevancia(w, "pregão") == 0 {
		t.Fatal("descritor da TTDD entra no corpus")
	}
	s := SinteseCanonica(w)
	for _, want := range []string{"ADM.LIC.001", "Pregão", "guarda permanente", "1 ano", "4 anos", "Prazo total de guarda antes da destinação: 5 ano(s)", "1. Demandante", "DFD", "segue para a etapa 2", "devolve à etapa 1 em diligência", "mantém o processo aberto", "peça opcional", "exige conferência da cópia"} {
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

func ptr[T any](v T) *T { return &v }

func TestFaseEDestinacao(t *testing.T) {
	for _, c := range []struct {
		anos     *int
		condicao string
		corrente bool
		want     string
	}{
		{ptr(1), "", true, "1 ano"},
		{ptr(5), "", false, "5 anos"},
		{nil, "Enquanto estiver vigorando", true, "Enquanto estiver vigorando"},
		{nil, "", true, "não informado na TTDD"},
		{nil, "", false, "não há"},
	} {
		if got := Fase(c.anos, c.condicao, c.corrente); got != c.want {
			t.Errorf("Fase(%v, %q, %v) = %q, quero %q", c.anos, c.condicao, c.corrente, got, c.want)
		}
	}
	elim := ClassificacaoTTDD{DestinacaoFinal: ptr(DestinacaoEliminacao)}
	if elim.Destinacao() != "eliminação" || (ClassificacaoTTDD{}).Destinacao() != "não definida na TTDD" {
		t.Fatal("descrição da destinação")
	}
	if total, ok := (ClassificacaoTTDD{FaseCorrenteAnos: ptr(2), FaseIntermAnos: ptr(3)}).PrazoTotalAnos(); !ok || total != 5 {
		t.Fatalf("prazo total 2+3: %d %v", total, ok)
	}
	if total, ok := (ClassificacaoTTDD{FaseCorrenteAnos: ptr(2)}).PrazoTotalAnos(); !ok || total != 2 {
		t.Fatalf("sem fase intermediária conta só a corrente: %d %v", total, ok)
	}
	for _, c := range []ClassificacaoTTDD{
		{FaseCorrenteCondicao: "Enquanto estiver vigorando", FaseIntermAnos: ptr(5)},
		{FaseCorrenteAnos: ptr(1), FaseIntermCondicao: "30 dias após"},
	} {
		if _, ok := c.PrazoTotalAnos(); ok {
			t.Fatalf("prazo por condição não tem total em anos: %+v", c)
		}
	}
}

func TestCodigoTTDDValido(t *testing.T) {
	for _, ok := range []string{"2.0", "12.0.04", "2.0.01.00", "2.0.01.00.11", "12.0.04.06.01-2"} {
		if !CodigoTTDDValido(ok) {
			t.Errorf("%q deveria ser válido", ok)
		}
	}
	for _, ruim := range []string{"", "2", "2.1.01", "2.0.1", "2.0.01.00.00.00", "abc", "2.0.01.00.00-x"} {
		if CodigoTTDDValido(ruim) {
			t.Errorf("%q deveria ser inválido", ruim)
		}
	}
}

func serieOficial() ClassificacaoTTDD {
	data := time.Date(2025, 8, 25, 0, 0, 0, 0, time.UTC)
	return ClassificacaoTTDD{Codigo: "3.0.01.00.00", Descritor: "Processo de Empenho", FaseCorrenteAnos: ptr(2), FaseIntermAnos: ptr(3),
		DestinacaoFinal: ptr(DestinacaoGuardaPermanente), Observacoes: "Conferir com a contabilidade",
		Subfuncao: &SubfuncaoTTDD{Codigo: "3.0.01.00", Nome: "Contabilidade", Recomendacao: "Transferir após aprovação do TCE-MT.",
			Funcao: FuncaoTTDD{Codigo: "3.0.01", Nome: "Finanças",
				Orgao: OrgaoTTDD{Prefixo: "3.0", Nome: "Secretaria Municipal de Fazenda", EdicaoDiario: "6.275", DataPublicacao: &data, Versao: "II"}}}}
}

func TestSinteseTTDD(t *testing.T) {
	s := SinteseTTDD(serieOficial())
	for _, want := range []string{"3.0.01.00.00 — Processo de Empenho", "Secretaria Municipal de Fazenda › Finanças › Contabilidade",
		"fase corrente 2 anos; fase intermediária 3 anos; destinação final: guarda permanente", "Prazo total de guarda antes da destinação: 5 ano(s)",
		"Observações: Conferir", "Recomendação da subfunção: Transferir", "versão II, Diário Oficial nº 6.275 de 25/08/2025"} {
		if !strings.Contains(s, want) {
			t.Fatalf("síntese sem %q:\n%s", want, s)
		}
	}
	semFonte := SinteseTTDD(ClassificacaoTTDD{Codigo: "x", Descritor: "y", FaseCorrenteCondicao: "Enquanto estiver vigorando"})
	if strings.Contains(semFonte, "Fonte") || !strings.Contains(semFonte, "fase intermediária não há") || !strings.Contains(semFonte, "não definida na TTDD") {
		t.Fatalf("síntese sem hierarquia: %s", semFonte)
	}
	o := serieOficial()
	o.Subfuncao.Funcao.Orgao = OrgaoTTDD{Nome: "Órgão"}
	if s := Temporalidade(o); !strings.Contains(s, "Fonte: TTDD da Órgão.") {
		t.Fatalf("fonte sem publicação: %s", s)
	}
}

// Série revogada: a síntese avisa antes dos prazos (com e sem a edição).
func TestTemporalidadeRevogada(t *testing.T) {
	c := serieOficial()
	if !c.Vigente() || strings.Contains(Temporalidade(c), "revogada") {
		t.Fatal("série vigente não tem aviso")
	}
	em := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	c.RevogadaEm, c.RevogadaEdicao = &em, "6.400"
	if c.Vigente() || !strings.HasPrefix(Temporalidade(c), "ATENÇÃO: série revogada em 01/03/2027 (Diário Oficial nº 6.400) —") {
		t.Fatalf("aviso de revogação: %s", Temporalidade(c))
	}
	c.RevogadaEdicao = ""
	if !strings.HasPrefix(Temporalidade(c), "ATENÇÃO: série revogada em 01/03/2027 —") {
		t.Fatalf("revogação sem edição: %s", Temporalidade(c))
	}
}

func TestRelevanciaTTDD(t *testing.T) {
	c := serieOficial()
	if r := RelevanciaTTDD(c, "Qual o prazo de guarda do processo de empenho?"); r < LimiarRelevancia {
		t.Fatalf("pergunta coberta abaixo do limiar: %.2f", r)
	}
	if r := RelevanciaTTDD(c, "3.0.01.00.00"); r < LimiarRelevancia {
		t.Fatalf("código exato abaixo do limiar: %.2f", r)
	}
	if r := RelevanciaTTDD(c, "vacinação de cães"); r != 0 {
		t.Fatalf("pergunta sem relação: %.2f", r)
	}
	if RelevanciaTTDD(c, "de a o") != 0 {
		t.Fatal("pergunta vazia")
	}
}
