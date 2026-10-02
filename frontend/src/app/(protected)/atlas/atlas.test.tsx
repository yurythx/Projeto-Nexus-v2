import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, page, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtlasPage from "./page";

/** Preenche colando (rápido e estável sob carga) em vez de digitar tecla a tecla. */
async function fill(el: HTMLElement, text: string) {
  await userEvent.click(el);
  await userEvent.paste(text);
}

const CLASSIFICACAO = {
  codigo: "2.0.02.00.07",
  descritor: "Pregão Presencial / Eletrônico",
  fase_corrente_anos: 1,
  fase_interm_anos: 4,
  destinacao_final: "GUARDA_PERMANENTE",
  observacoes: "Processo integral",
  created_at: "2026-10-01T00:00:00Z",
};

const RESUMO = {
  id: "wf-1",
  codigo_processual: "ADM.LIC.001",
  titulo: "Pregão Eletrônico",
  objetivo: "Contratação de bens e serviços comuns",
  publico_alvo: "Secretarias",
  versao: 1,
  ativo: true,
  nivel_acesso: "PUBLICO",
  hipotese_legal_restricao: "",
  codigo_ttdd: "2.0.02.00.07",
  classificacao: CLASSIFICACAO,
  total_etapas: 1,
  etapas: [],
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
};

const DETALHE = {
  ...RESUMO,
  etapas: [
    {
      id: "et-1",
      ordem: 1,
      unidade_administrativa: "SEC/DEM",
      nome_setor: "Setor Demandante",
      atribuicoes_setor: "Elaborar o Termo de Referência",
      prazo_sla_em_dias: 5,
      manter_aberto_apos_remessa: true,
      documentos: [
        {
          id: "doc-1",
          nome_documento: "Termo de Referência",
          obrigatorio: true,
          formato: "NATO_DIGITAL",
          tipo_assinatura: "CONJUNTA_MULTINIVEL",
          exige_conferencia_copia: false,
          modelo_minuta_padrao_url: "",
        },
      ],
      transicoes: [
        {
          id: "tr-1",
          destino_ordem: 2,
          condicao_transicao: "TR aprovado",
          is_devolucao_diligencia: false,
          descricao_diligencia: "",
        },
      ],
    },
  ],
};

const base = (perms: string[]) => ({
  ...identityRoutes({
    permissions: perms,
    scopes: [{ perfil: "p", origem: "manual", permissions: perms }],
  }),
  "GET v1/atlas/workflows": page([RESUMO]),
  "GET v1/atlas/admin/workflows": page([RESUMO]),
  "GET v1/atlas/workflows/wf-1": { data: DETALHE },
  "GET v1/atlas/admin/workflows/wf-1": { data: DETALHE },
  "GET v1/atlas/ttdd": page([
    CLASSIFICACAO,
    {
      ...CLASSIFICACAO,
      codigo: "2.0.01.00.01",
      descritor: "Organogramas",
      destinacao_final: "ELIMINACAO",
    },
  ]),
});

describe("Atlas", () => {
  beforeEach(() => resetNavigation({}, "/atlas"));

  it("consulta pública: lista pela rota pública, sem gestão nem assistente", async () => {
    const backend = mockBackend(base([]));
    renderApp(<AtlasPage />);

    const item = await screen.findByRole("button", { name: /ADM\.LIC\.001/ });
    expect(within(item).getByText("1 etapa")).toBeInTheDocument();
    expect(backend.to("GET v1/atlas/workflows")).toHaveLength(1);
    expect(backend.to("GET v1/atlas/admin/workflows")).toHaveLength(0);
    expect(screen.queryByRole("button", { name: /Novo procedimento/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: /Assistente/ })).not.toBeInTheDocument();

    await userEvent.click(item);
    expect(router.replace).toHaveBeenCalledWith("/atlas?procedimento=wf-1");
  });

  it("detalhe busca o percurso completo (etapas, peças e transições)", async () => {
    resetNavigation({}, "/atlas", "procedimento=wf-1");
    mockBackend(base([]));
    renderApp(<AtlasPage />);

    expect(await screen.findByText("Trilha de tramitação (1 etapa)")).toBeInTheDocument();
    expect(screen.getByText("Setor Demandante")).toBeInTheDocument();
    expect(screen.getByText("Termo de Referência")).toBeInTheDocument();
    expect(screen.getByText(/Assinatura conjunta \(multinível\)/)).toBeInTheDocument();
    expect(screen.getByText(/mantém o processo aberto/)).toBeInTheDocument();
    expect(screen.getByText(/TR aprovado → etapa 2/)).toBeInTheDocument();
    expect(screen.getByText(/4 ano\(s\) na intermediária/)).toBeInTheDocument();
  });

  it("gestão: rota administrativa e desativação do procedimento", async () => {
    resetNavigation({}, "/atlas", "procedimento=wf-1");
    const backend = mockBackend({
      ...base(["atlas:manage"]),
      "POST v1/atlas/admin/workflows/wf-1/desativar": { data: { ...DETALHE, ativo: false } },
    });
    renderApp(<AtlasPage />);

    await userEvent.click(await screen.findByRole("button", { name: "Desativar procedimento" }));
    expect(backend.to("POST v1/atlas/admin/workflows/wf-1/desativar")).toHaveLength(1);
    expect(await screen.findByRole("button", { name: "Ativar procedimento" })).toBeInTheDocument();
    expect(backend.to("GET v1/atlas/admin/workflows").length).toBeGreaterThan(0);
  });

  it("aba TTDD com rótulos de destinação", async () => {
    mockBackend(base([]));
    renderApp(<AtlasPage />);

    await userEvent.click(await screen.findByRole("tab", { name: /Tabela de Temporalidade/ }));
    expect(await screen.findByText("2.0.01.00.01")).toBeInTheDocument();
    expect(screen.getByText("Guarda permanente")).toBeInTheDocument();
    expect(screen.getByText("Eliminação")).toBeInTheDocument();
  });

  it("assistente (atlas:read): envia a pergunta e mostra modo e fontes", async () => {
    const backend = mockBackend({
      ...base(["atlas:read"]),
      "POST v1/atlas/chat": {
        data: {
          answer: "Instrua com DFD, ETP e TR.",
          score: 0.9,
          refused: false,
          mode: "sintese",
          sources: [
            { id: "wf-1", codigo_processual: "ADM.LIC.001", titulo: "Pregão", relevancia: 0.9 },
          ],
          generated_at: "2026-10-01T00:00:00Z",
        },
      },
    });
    renderApp(<AtlasPage />);

    await userEvent.click(await screen.findByRole("tab", { name: /Assistente/ }));
    await userEvent.type(screen.getByLabelText("Sua pergunta"), "Como tramitar o pregão?");
    await userEvent.click(screen.getByRole("button", { name: "Enviar pergunta" }));

    expect(await screen.findByText("Instrua com DFD, ETP e TR.")).toBeInTheDocument();
    expect(
      screen.getByText(/Síntese direta dos procedimentos homologados · relevância 90%/),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "ADM.LIC.001" })).toHaveAttribute(
      "href",
      "/atlas?procedimento=wf-1",
    );
    expect(backend.to("POST v1/atlas/chat")[0]!.body).toEqual({ query: "Como tramitar o pregão?" });
  });

  it("assistente mostra a mensagem do backend quando a consulta falha", async () => {
    mockBackend({
      ...base(["atlas:read"]),
      "POST v1/atlas/chat": {
        status: 429,
        error: { code: "RATE_LIMITED", message: "Muitas consultas; aguarde um minuto." },
      },
    });
    renderApp(<AtlasPage />);

    await userEvent.click(await screen.findByRole("tab", { name: /Assistente/ }));
    await userEvent.type(screen.getByLabelText("Sua pergunta"), "pregão eletrônico");
    await userEvent.click(screen.getByRole("button", { name: "Enviar pergunta" }));
    expect(await screen.findByText("Muitas consultas; aguarde um minuto.")).toBeInTheDocument();
  });

  it("cadastro: várias etapas na ordem e peças uma por linha", async () => {
    const backend = mockBackend({
      ...base(["atlas:manage"]),
      "POST v1/atlas/admin/workflows": { status: 201, data: { ...DETALHE, id: "wf-2" } },
    });
    renderApp(<AtlasPage />);

    await userEvent.click(await screen.findByRole("button", { name: /Novo procedimento/ }));
    await fill(screen.getByLabelText("Código processual *"), "adm.fin.021");
    await userEvent.selectOptions(screen.getByLabelText("Nível de acesso sugerido *"), "RESTRITO");
    await fill(screen.getByLabelText("Hipótese legal da restrição *"), "LAI, art. 31");
    await fill(screen.getByLabelText("Título do procedimento *"), "Adiantamento");
    await fill(screen.getByLabelText("Objetivo *"), "Prestar contas");
    await fill(screen.getByLabelText("Público-alvo *"), "Supridos");
    await screen.findByRole("option", { name: /2\.0\.02\.00\.07/ });
    await userEvent.selectOptions(screen.getByLabelText("Classificação TTDD *"), "2.0.02.00.07");

    const setores = () => screen.getAllByLabelText("Setor *");
    await fill(setores()[0]!, "Gabinete");
    await fill(screen.getAllByLabelText("Sigla da unidade *")[0]!, "GAB");
    await fill(screen.getAllByLabelText("Atribuições *")[0]!, "Solicitar");
    await fill(
      screen.getAllByLabelText("Peças exigidas (uma por linha)")[0]!,
      "Requerimento\nComprovante",
    );
    await userEvent.click(screen.getByRole("button", { name: /Adicionar etapa/ }));
    await fill(setores()[1]!, "Contabilidade");
    await fill(screen.getAllByLabelText("Sigla da unidade *")[1]!, "SEFIN");
    await fill(screen.getAllByLabelText("Atribuições *")[1]!, "Analisar");

    await userEvent.click(screen.getByRole("button", { name: "Cadastrar procedimento" }));
    const body = backend.to("POST v1/atlas/admin/workflows")[0]!.body as {
      codigo_processual: string;
      nivel_acesso: string;
      hipotese_legal_restricao: string;
      etapas: { ordem: number; nome_setor: string; documentos: { nome_documento: string }[] }[];
    };
    expect(body.codigo_processual).toBe("adm.fin.021");
    expect(body.nivel_acesso).toBe("RESTRITO");
    expect(body.hipotese_legal_restricao).toBe("LAI, art. 31");
    expect(body.etapas.map((e) => [e.ordem, e.nome_setor])).toEqual([
      [1, "Gabinete"],
      [2, "Contabilidade"],
    ]);
    expect(body.etapas[0]!.documentos.map((d) => d.nome_documento)).toEqual([
      "Requerimento",
      "Comprovante",
    ]);
    expect(router.replace).toHaveBeenCalledWith("/atlas?procedimento=wf-2");
  });
});
