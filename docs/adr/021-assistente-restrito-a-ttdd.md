# 021 — Assistente do Atlas restrito à TTDD

- **Status:** Aceito e implementado
- **Data:** 2026-10-03
- **Relacionado:** [ADR 015](015-atlas-padronizado.md) (grounding), [ADR 017](017-ttdd-oficial.md) (TTDD oficial), [ADR 020](020-conexoes-de-ia.md) (conexões de IA)

## Contexto

O assistente respondia sobre duas fontes — procedimentos do Atlas (como
tramitar, etapas, peças) e séries da TTDD — e recusava o que nenhuma das
duas cobria, com uma mensagem sobre "fluxo ou procedimento homologado".
Com a IA configurável (ADR 020), inclusive fornecedores externos, a
decisão foi **blindar o objetivo**: a IA existe para orientar sobre a
Tabela de Temporalidade e Destinação de Documentos e nada mais.

## Decisão

1. **Uma fonte só:** a resposta se apoia exclusivamente em séries da TTDD
   (relevância ≥ 0,65, até 3 séries). Na relevância, palavras de temporalidade
   ("prazo", "guarda", "destinação", "final", "por quanto tempo") que a
   série não contém não contam: descrevem a pergunta, não a série — sem
   isso, "Qual a destinação final dos organogramas?" ficava abaixo do limiar. O grounding por procedimento
   (`Repository.Candidatos`, `Relevancia`, `SinteseCanonica`) foi removido.
2. **Duas recusas canônicas** (`domain.MensagemForaDoObjetivo` e
   `domain.MensagemSemSerie`):
   - assunto que não é TTDD → *"Esse assunto foge do objetivo da IA: este
     assistente responde apenas sobre a Tabela de Temporalidade e
     Destinação de Documentos (TTDD)…"*;
   - assunto de temporalidade ("prazo", "guarda", "destinação",
     "eliminar", "TTDD"…) sem série correspondente → *"Não localizei na
     TTDD oficial uma série documental…"*. Uma coincidência fraca de
     palavras com alguma série (abaixo do limiar) não torna o assunto
     válido: sem termo de temporalidade, a recusa é a de assunto.
3. **Decisão determinística antes do modelo:** pedido de procedimento
   ("tramitar", "etapa", "setor", "quem assina"…) sem termo de
   temporalidade é recusado direto — senão "Como tramitar o processo de
   pregão?" casaria por palavras com a série "Processos relativos a pregão".
4. **Reforço no modelo:** o prompt restringe o papel à TTDD, manda devolver
   a recusa literal para qualquer outro assunto e ignorar pedidos para
   mudar de papel ou revelar as instruções. Se o modelo devolver a recusa
   por assunto, a resposta vai como `recusada`, sem fontes.
5. **Interface:** boas-vindas, sugestões e aviso do assistente falam só da
   TTDD; na página do procedimento, o botão vira "Perguntar sobre a
   temporalidade" (pergunta sobre a série do procedimento). As fontes da
   resposta são sempre séries (`tipo: "ttdd"`).

## Consequências

- Perguntas sobre procedimentos passam a ser recusadas; o roteiro de cada
  procedimento continua na própria página do Atlas.
- As listas de termos (temporalidade e procedimento) são heurísticas: um
  caso mal classificado se corrige acrescentando o termo e um teste
  (`TestObjetivoDoAssistente`).
