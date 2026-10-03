package domain

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// LimiarRelevancia é a relevância mínima (0..1) para o assistente
// responder. Abaixo dela a resposta é a recusa canônica — nada de
// orientação inventada sem procedimento homologado que a sustente.
const LimiarRelevancia = 0.65

// MensagemRecusa é a resposta quando nenhum procedimento homologado
// sustenta a pergunta.
const MensagemRecusa = "Não localizei nenhum fluxo ou procedimento homologado para a sua consulta. " +
	"Verifique a nomenclatura ou a sigla do setor, ou solicite à unidade de gestão documental o cadastramento do fluxo."

// stopwords: palavras sem conteúdo que não contam na cobertura da pergunta
// ("Como tramitar UM processo DE pregão?" mede só tramitar/processo/pregão).
var stopwords = map[string]bool{
	"a": true, "o": true, "as": true, "os": true, "um": true, "uma": true, "uns": true, "umas": true,
	"de": true, "da": true, "do": true, "das": true, "dos": true, "em": true, "na": true, "no": true,
	"nas": true, "nos": true, "por": true, "para": true, "pra": true, "com": true, "sem": true, "e": true,
	"ou": true, "que": true, "qual": true, "quais": true, "como": true, "onde": true, "quando": true,
	"se": true, "ao": true, "aos": true, "sao": true, "ser": true, "sobre": true, "meu": true,
	"minha": true, "seu": true, "sua": true, "eu": true, "voce": true, "posso": true, "devo": true,
	"preciso": true, "fazer": true, "isso": true, "este": true, "esta": true, "esse": true, "essa": true,
	"pelo": true, "pela": true, "entre": true, "mais": true, "tem": true, "ter": true, "ha": true,
}

// Fold devolve o texto em minúsculas e sem acentos.
func Fold(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Termos extrai os termos de conteúdo da pergunta (sem acento, sem
// stopwords, com 2+ caracteres, sem repetição).
func Termos(pergunta string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range strings.FieldsFunc(Fold(pergunta), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.'
	}) {
		t = strings.Trim(t, ".")
		if len([]rune(t)) < 2 || stopwords[t] || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// corpus é o texto pesquisável do procedimento completo.
func corpus(w Workflow) string {
	var b strings.Builder
	b.WriteString(w.Titulo + " " + w.Objetivo + " " + w.PublicoAlvo + " " + w.CodigoProcessual + " " + w.CodigoTTDD)
	if w.Classificacao != nil {
		b.WriteString(" " + w.Classificacao.Descritor)
	}
	for _, e := range w.Etapas {
		b.WriteString(" " + e.NomeSetor + " " + e.UnidadeAdministrativa + " " + e.AtribuicoesSetor)
		for _, d := range e.Documentos {
			b.WriteString(" " + d.NomeDocumento)
		}
	}
	return Fold(b.String())
}

// contem casa o termo inteiro ou, a partir de 6 letras, o radical sem as
// duas últimas (plural e gênero: "documentos" ~ "documento", "licitacoes" ~
// "licitacao").
func contem(texto, termo string) bool {
	if strings.Contains(texto, termo) {
		return true
	}
	r := []rune(termo)
	return len(r) >= 6 && strings.Contains(texto, string(r[:len(r)-2]))
}

// Relevancia mede (0..1) quanto o procedimento sustenta a pergunta: a
// fração dos termos da pergunta presentes no procedimento, com bônus para
// o código exato e para termos do título. Determinística e auditável — o
// limiar não depende do modelo de linguagem.
func Relevancia(w Workflow, pergunta string) float64 {
	termos := Termos(pergunta)
	if len(termos) == 0 {
		return 0
	}
	texto := corpus(w)
	titulo := Fold(w.Titulo)
	codigos := []string{Fold(w.CodigoProcessual), Fold(w.CodigoTTDD)}
	matched := 0
	bonus := 0.0
	for _, t := range termos {
		if contem(texto, t) {
			matched++
		}
		if t == codigos[0] || t == codigos[1] {
			bonus += 0.3
		}
		if contem(titulo, t) {
			bonus += 0.15
		}
	}
	if matched == 0 {
		return 0
	}
	score := float64(matched)/float64(len(termos)) + bonus
	if score > 1 {
		score = 1
	}
	return score
}

// SinteseCanonica redige a orientação diretamente dos dados homologados
// (sem modelo de linguagem): usada quando o assistente de IA não está
// configurado ou falha.
func SinteseCanonica(w Workflow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s)\n\nObjetivo: %s\n", w.Titulo, w.CodigoProcessual, w.Objetivo)
	if w.NivelAcesso != NivelPublico {
		fmt.Fprintf(&b, "Nível de acesso: %s — hipótese legal: %s\n", w.NivelAcesso, w.HipoteseLegal)
	}
	if c := w.Classificacao; c != nil {
		fmt.Fprintf(&b, "\nEnquadramento na TTDD: %s — %s\n", c.Codigo, c.Descritor)
		b.WriteString(Temporalidade(*c))
	}
	b.WriteString("\nFluxo de tramitação:\n")
	for _, e := range w.Etapas {
		fmt.Fprintf(&b, "\n%d. %s (%s) — prazo: %d dia(s)\n   %s\n", e.Ordem, e.NomeSetor, e.UnidadeAdministrativa, e.PrazoSLAEmDias, e.AtribuicoesSetor)
		if e.ManterAbertoAposRemessa {
			b.WriteString("   A unidade mantém o processo aberto para acompanhamento após a remessa.\n")
		}
		for _, d := range e.Documentos {
			req := "opcional"
			if d.Obrigatorio {
				req = "obrigatória"
			}
			conf := ""
			if d.ExigeConferenciaCopia {
				conf = "; exige conferência da cópia"
			}
			fmt.Fprintf(&b, "   • %s — peça %s, %s, assinatura %s%s\n", d.NomeDocumento, req, d.Formato, d.TipoAssinatura, conf)
		}
		for _, t := range e.Transicoes {
			if t.IsDevolucaoDiligencia {
				fmt.Fprintf(&b, "   ↩ %s: devolve à etapa %d em diligência (%s)\n", t.CondicaoTransicao, t.DestinoOrdem, t.DescricaoDiligencia)
			} else {
				fmt.Fprintf(&b, "   → %s: segue para a etapa %d\n", t.CondicaoTransicao, t.DestinoOrdem)
			}
		}
	}
	return b.String()
}

// Temporalidade descreve prazos, destinação, recomendação e fonte de uma
// série da TTDD (uma linha por informação).
func Temporalidade(c ClassificacaoTTDD) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Temporalidade: fase corrente %s; fase intermediária %s; destinação final: %s.\n",
		Fase(c.FaseCorrenteAnos, c.FaseCorrenteCondicao, true), Fase(c.FaseIntermAnos, c.FaseIntermCondicao, false), c.Destinacao())
	if total, ok := c.PrazoTotalAnos(); ok {
		fmt.Fprintf(&b, "Prazo total de guarda antes da destinação: %d ano(s).\n", total)
	}
	if c.Observacoes != "" {
		fmt.Fprintf(&b, "Observações: %s\n", c.Observacoes)
	}
	if s := c.Subfuncao; s != nil {
		if s.Recomendacao != "" {
			fmt.Fprintf(&b, "Recomendação da subfunção: %s\n", s.Recomendacao)
		}
		o := s.Funcao.Orgao
		fonte := fmt.Sprintf("TTDD da %s", o.Nome)
		if o.Versao != "" {
			fonte += ", versão " + o.Versao
		}
		if o.EdicaoDiario != "" {
			fonte += ", Diário Oficial nº " + o.EdicaoDiario
			if o.DataPublicacao != nil {
				fonte += " de " + o.DataPublicacao.Format("02/01/2006")
			}
		}
		fmt.Fprintf(&b, "Fonte: %s.\n", fonte)
	}
	return b.String()
}

// SinteseTTDD responde com os dados oficiais de uma série documental.
func SinteseTTDD(c ClassificacaoTTDD) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n", c.Codigo, c.Descritor)
	if s := c.Subfuncao; s != nil {
		fmt.Fprintf(&b, "Classificação: %s › %s › %s\n", s.Funcao.Orgao.Nome, s.Funcao.Nome, s.Nome)
	}
	b.WriteString(Temporalidade(c))
	return b.String()
}

// RelevanciaTTDD mede (0..1) quanto uma série da TTDD sustenta a pergunta,
// com a mesma regra dos procedimentos: fração dos termos presentes no
// descritor (e na função/subfunção), bônus para o código exato e para
// termos do próprio descritor.
func RelevanciaTTDD(c ClassificacaoTTDD, pergunta string) float64 {
	termos := Termos(pergunta)
	if len(termos) == 0 {
		return 0
	}
	descritor := Fold(c.Descritor)
	texto := descritor + " " + Fold(c.Codigo) + " " + Fold(c.Observacoes)
	if s := c.Subfuncao; s != nil {
		texto += " " + Fold(s.Nome+" "+s.Funcao.Nome)
	}
	matched, bonus := 0, 0.0
	for _, t := range termos {
		if contem(texto, t) {
			matched++
		}
		if t == Fold(c.Codigo) {
			bonus += 0.3
		}
		if contem(descritor, t) {
			bonus += 0.15
		}
	}
	if matched == 0 {
		return 0
	}
	return min(1, float64(matched)/float64(len(termos))+bonus)
}
