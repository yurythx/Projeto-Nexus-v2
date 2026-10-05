# Plano de validação dos procedimentos do Atlas

Para: equipe de gestão documental e quem conduzirá as entrevistas nos
departamentos.

## 1. Objetivo

Validar com cada departamento **como os processos tramitam de verdade** e
**padronizar os documentos** que eles produzem, antes de publicar os
procedimentos no Atlas (consulta interna, site público e assistente).

O ponto de partida são **28 rascunhos** gerados a partir da Tabela de
Temporalidade (TTDD), dos processos mais comuns de cada secretaria. Eles são
**hipóteses**: etapas, setores, prazos e documentos têm de ser confirmados
ou corrigidos nas entrevistas.

## 2. Como funciona no sistema

Cada procedimento passa por três situações:

| Situação | O que significa | Visível para |
|---|---|---|
| **Rascunho** | Gerado, ainda não conversado com o departamento | Só a gestão do Atlas |
| **Em validação** | Entrevistas em andamento | Só a gestão do Atlas |
| **Homologado** | Validado e publicado | Todos (consulta, site público, assistente, busca) |

Regras:

- Rascunho e em validação **não aparecem** na consulta, no site público, na
  busca nem no assistente.
- O rascunho de uma **nova versão** de um procedimento já publicado não tira
  a versão atual do ar, e não avisa ninguém.
- Só a **homologação** publica. Ela substitui a versão anterior e avisa
  quem segue o fluxo e as unidades que participam dele.

Passo a passo:

1. **Importar os rascunhos.** Em Atlas → Importar, envie
   `deploy/atlas/rascunhos-procedimentos.json` com **"Importar como
   rascunho"** marcado (já vem marcado), clique em **Simular**, confira e
   **Aplique**.
2. **Registrar cada entrevista.** Na página do procedimento, cartão
   **Validação** → **Registrar entrevista**. Informe a data, o
   departamento, os participantes (pela **função**, não pelo nome), o que
   foi validado ou corrigido e as pendências. O primeiro registro muda o
   procedimento para **Em validação**.
3. **Corrigir o fluxo.** Corrija pelo botão **Nova versão**. Para corrigir
   vários de uma vez, baixe o arquivo em Atlas → Importar → "Baixar os
   procedimentos em vigor", edite e importe de novo como rascunho.
4. **Ligar os modelos.** Ligue a cada documento o modelo padronizado: em
   **Detalhes** da peça, ou na biblioteca (Atlas → Modelos de documento).
5. **Homologar.** Com o fluxo e os documentos validados, use **Homologar**
   (pede confirmação).
6. **Acompanhar.** Em **Atlas → Cobertura** (rascunhos, em validação, peças
   sem modelo, séries sem fluxo).

As entrevistas ficam registradas pelo **código** do procedimento, então
continuam valendo para as versões seguintes. Elas não se apagam: uma
correção é um novo registro.

## 3. Roteiro da entrevista

Leve a **ficha de validação** impressa (cartão Validação → "Ficha de validação
para imprimir"): ela traz o fluxo proposto item a item, com "confere /
corrigir", a tabela de padronização de cada documento e o fechamento com as
assinaturas. Percorra-a com quem executa o processo e depois registre a
entrevista no sistema.

### 3.1 O processo

- Quem pede? Por qual canal (balcão, sistema, e-mail)?
- Qual é a base legal (lei, decreto, instrução normativa)?
- O processo é **público** ou tem informação **restrita** (dados pessoais,
  saúde, situação fiscal)? Qual hipótese legal?
- A série da TTDD sugerida está certa? O documento final é guardado por
  quanto tempo?

### 3.2 Cada etapa

- **Qual setor** executa? Qual é a **sigla oficial** da unidade? (ver o
  item 5)
- **O que** o setor faz? Quem decide e quem só confere?
- **Prazo** real em dias. Existe prazo legal?
- **Condição para seguir** à próxima etapa.
- O que faz o processo **voltar** (diligência)? Para quem volta?
- A unidade mantém o processo aberto depois de enviar?

### 3.3 Cada documento (padronização)

| Pergunta | Por quê |
|---|---|
| Qual é o **nome oficial** do documento? | O mesmo nome em todos os fluxos permite reaproveitar o modelo |
| É **obrigatório** ou opcional? | Entra no checklist do processo |
| Nasce **no sistema** (nato-digital) ou é **papel digitalizado**? | Define a conferência da cópia |
| Quem **assina**: uma pessoa, várias (conjunta) ou em bloco? | Configura a assinatura no Signum |
| Existe **modelo**? Quem é o dono do modelo? | Vai para a biblioteca de modelos |
| Quais **campos** o documento precisa ter (dados mínimos)? | Padroniza o modelo |
| Há documento **duplicado ou desnecessário** que pode sair? | Simplifica o fluxo |

Peça os **modelos em uso**, em Word ou LibreOffice: eles serão
padronizados e cadastrados na biblioteca.

### 3.4 Fechamento

- Leia o fluxo corrigido para o entrevistado confirmar.
- Registre as pendências (documento que falta, decisão de outra
  secretaria).
- Combine quem valida a versão final (chefia do departamento).

## 4. Depois das entrevistas: a versão consolidada

- Corrija o rascunho com o que foi validado (botão **Nova versão** — já vem
  marcado "Salvar como rascunho") e ligue os modelos padronizados.
- **Homologar** publica a versão consolidada: ela fica disponível na consulta
  interna, no site público (`/procedimentos`), na busca, no assistente e no
  Trâmite (abertura de processo, checklist de peças), com versão, histórico e
  impressão. A versão anterior, se houver, sai do ar e quem segue o fluxo é
  avisado.
- Revisões futuras seguem o mesmo caminho: nova versão como rascunho →
  entrevista → homologar. O homologado nunca volta a rascunho.

## 5. Procedimentos novos

Se a entrevista revelar um processo que não está na lista, cadastre-o em
**Atlas → Novo procedimento** (vem marcado "Salvar como rascunho") ou inclua-o
no arquivo de importação. Ele entra no mesmo ciclo de validação.

O painel **Atlas → Cobertura → Processos da TTDD sem procedimento** lista as
subfunções da TTDD com séries de processo (requerimento, licença, certidão…) e
nenhum procedimento, nem rascunho — é a lista de candidatos, atualizada sozinha
à medida que os procedimentos são criados.

## 6. Checklist antes de homologar

- [ ] Todas as unidades das etapas foram entrevistadas (ou consultadas).
- [ ] As siglas das etapas são as **cadastradas** no sistema.
- [ ] Os nomes dos documentos seguem o padrão da biblioteca.
- [ ] Os documentos obrigatórios têm modelo ligado (ou a pendência está
      registrada).
- [ ] O nível de acesso e a hipótese legal foram confirmados.
- [ ] A série da TTDD foi confirmada pela gestão documental.
- [ ] Nenhuma pendência aberta nas entrevistas.

## 7. Ponto de atenção: siglas das unidades

O cadastro de unidades (Configurações → Unidades) está quase todo com a
sigla **"SEDE"**. Os rascunhos usam siglas descritivas (`SEMAD/COMPRAS`,
`SEGEP/RH`, `SEFAZ/CONTABILIDADE`…), que devem ser **substituídas pelas
siglas oficiais** confirmadas nas entrevistas.

O cadastro também precisa ser corrigido: o aviso de nova versão chega às
unidades **pela sigla**, e com "SEDE" ninguém é avisado.

## 8. Procedimentos prioritários (rascunhos)

**28 procedimentos, 93 etapas e 154 documentos.** A coluna "Entrevistar"
indica quem conversar primeiro.

| Secretaria (TTDD) | Código | Procedimento | Série TTDD | Entrevistar |
|---|---|---|---|---|
| 2.0 Administração | ADM.CMP.004 | Adesão a Ata de Registro de Preços | 2.0.02.00.10 | Compras, Procuradoria |
| 2.0 Administração | ADM.CON.005 | Formalização de contrato e aditivos | 2.0.02.02.12 | Contratos, Procuradoria |
| 2.0 Administração | ADM.CON.006 | Fiscalização de contrato | 2.0.02.02.15 | Fiscais, Contabilidade |
| 2.0 Administração | ADM.CMP.007 | Inexigibilidade de licitação | 2.0.02.01.03 | Compras, Procuradoria |
| 2.0 Administração | ADM.FRO.008 | Infração de trânsito de veículo oficial | 2.0.01.02.15 | Frotas, Folha |
| 2.0 Administração | ADM.SIC.009 | Pedido de acesso à informação (LAI) | 2.0.03.02.00 | SIC |
| 2.0 Gestão de Pessoas | RH.FER.001 | Férias | 2.0.06.03.01 | RH, Folha |
| 2.0 Gestão de Pessoas | RH.LIC.002 | Licença-prêmio | 2.0.06.03.05 | RH, Procuradoria |
| 2.0 Gestão de Pessoas | RH.ADI.003 | Adicional de insalubridade | 2.0.06.02.09 | RH, DESOPEM |
| 2.0 Gestão de Pessoas | RH.APO.004 | Aposentadoria voluntária | 2.0.07.01.01 | RH, IMPRO |
| 2.0 Gestão de Pessoas | RH.PAD.005 | Processo Administrativo Disciplinar *(restrito)* | 2.0.05.02.04 | Comissão de PAD, Procuradoria |
| 2.0 Gestão de Pessoas | RH.EST.006 | Avaliação de estágio probatório | 2.0.08.01.02 | RH, comissão de avaliação |
| 2.0 Gestão de Pessoas | RH.CON.007 | Posse de aprovado em concurso | 2.0.08.00.14 | RH, DESOPEM |
| 3.0 Fazenda | FAZ.DIA.001 | Diárias (concessão e prestação de contas) | 3.0.01.00.05 | Contabilidade |
| 3.0 Fazenda | FAZ.DES.002 | Liquidação e pagamento de despesa | 3.0.01.00.02 | Contabilidade, Tesouraria |
| 3.0 Fazenda | FAZ.PAR.003 | Parcelamento de débitos | 3.0.02.00.03 | Dívida ativa, Atendimento |
| 3.0 Fazenda | FAZ.IPT.004 | Isenção de IPTU | 3.0.02.01.01 | Cadastro imobiliário |
| 3.0 Fazenda | FAZ.ALV.005 | Alvará de funcionamento | 3.0.03.00.04 | Licenciamento, Fiscalização |
| 3.0 Fazenda | FAZ.CND.006 | Certidão negativa de débitos | 3.0.02.00.00 | Atendimento, Dívida ativa |
| 4.0 Controle Interno | CTR.OUV.001 | Manifestação de ouvidoria *(restrito)* | 4.0.01.02.00 | Ouvidoria |
| 6.0 Habitação | HAB.ALV.001 | Alvará de licença para construção | 6.0.02.00.00 | Aprovação de projetos |
| 6.0 Habitação | HAB.HAB.002 | Habite-se | 6.0.02.00.08 | Fiscalização de obras |
| 6.0 Habitação | HAB.REU.003 | REURB-S *(restrito)* | 6.0.01.01.00 | Regularização fundiária |
| 8.0 Meio Ambiente | MAM.LIC.001 | Licenciamento ambiental | 8.0.01.01.00 | Licenciamento ambiental |
| 9.0 Desenvolvimento Econômico | DEC.USO.001 | Uso do solo para profissional liberal | 9.0.01.01.04 | Sala do Empreendedor |
| 10.0 Assistência Social | SAS.OSC.001 | Inscrição de OSC no CMAS | 10.0.02.00.00 | CMAS, Vigilância socioassistencial |
| 12.0 Saúde | SAU.VSA.001 | Licença sanitária | 12.0.10.02.03 | Vigilância sanitária |
| 12.0 Saúde | SAU.TFD.001 | Tratamento Fora do Domicílio *(restrito)* | 12.0.06.01.08 | TFD, Regulação |

Já publicados (devem ser revisados nas mesmas entrevistas):

- `ADM.LIC.001` (Pregão eletrônico);
- `ADM.DIR.002` (Dispensa por valor);
- `ADM.MAT.003` (Carga patrimonial).

## 9. Lacunas a tratar com a gestão documental

- **7.0 Infraestrutura** não tem rascunho: a TTDD só traz a subfunção
  "Gestão Administrativa". Definir com a secretaria quais processos
  tramitam (obras, manutenção, medições).
- **Educação** tem muitas unidades no cadastro, mas **não tem órgão na
  TTDD**. Os documentos escolares precisam de tabela ou enquadramento
  definido pela CCPAD antes de qualquer fluxo.
- Na data da carga dos rascunhos, **50 subfunções** com séries de processo
  ficaram sem procedimento (ex.: descontos em folha, licenças do servidor,
  saúde ocupacional, cargos comissionados, processo seletivo, patrimônio,
  cadastro imobiliário, averbação e cartografia, produção rural, áreas
  industriais). Use o painel de Cobertura para escolher a próxima leva.
- **Saúde (745 séries)** é a maior área: depois dos dois rascunhos,
  priorizar com a secretaria (regulação, farmácia, vigilância
  epidemiológica).

## 10. Proteção de dados nas entrevistas

- Registre os participantes pela **função** ("chefe do RH"), não pelo nome.
  O registro fica guardado e pode ser consultado pela gestão.
- Não anote dados de cidadãos ou servidores citados como exemplo.
- Para regerar os rascunhos (ex.: depois de ajustar o gerador), use
  `node deploy/atlas/gerar-rascunhos.mjs`.
