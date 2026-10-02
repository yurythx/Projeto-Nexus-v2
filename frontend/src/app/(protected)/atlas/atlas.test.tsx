import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtlasPage from "./page";

const MOCK_WORKFLOWS = [
  {
    id: "wf-1",
    tenant_id: "nexus",
    codigo_processual: "ADM.LIC.001",
    titulo: "Processo Licitatório de Pregão Eletrônico",
    objetivo: "Contratação de bens e serviços comuns",
    publico_alvo: "Secretarias e Pregoeiros",
    versao: 1,
    ativo: true,
    nivel_acesso: "PUBLICO",
    codigo_ttdd: "2.0.02.00.07",
    classificacao: {
      codigo: "2.0.02.00.07",
      descritor: "Processos relativos a Pregão Presencial / Eletrônico",
      fase_corrente_anos: 1,
      fase_interm_anos: 1,
      destinacao_final: "GUARDA_PERMANENTE",
      created_at: "2026-10-01T00:00:00Z",
    },
    etapas: [
      {
        id: "et-1",
        workflow_id: "wf-1",
        ordem: 1,
        unidade_administrativa: "SEC/DEMANDANTE",
        nome_setor: "Setor Demandante",
        atribuicoes_setor: "Elaboração do Termo de Referência",
        prazo_sla_em_dias: 5,
        manter_aberto_apos_remessa: true,
        documentos: [
          {
            id: "doc-1",
            etapa_id: "et-1",
            nome_documento: "Termo de Referência (TR)",
            obrigatorio: true,
            formato: "NATO_DIGITAL",
            tipo_assinatura: "CONJUNTA_MULTINIVEL",
            exige_conferencia_copia: false,
          },
        ],
        transicoes: [],
      },
    ],
    created_at: "2026-10-01T00:00:00Z",
    updated_at: "2026-10-01T00:00:00Z",
  },
];

const MOCK_TTDD = [
  {
    codigo: "2.0.02.00.07",
    descritor: "Processos relativos a Pregão Presencial / Eletrônico",
    fase_corrente_anos: 1,
    fase_interm_anos: 1,
    destinacao_final: "GUARDA_PERMANENTE",
    observacoes: "Processo integral de licitação",
    created_at: "2026-10-01T00:00:00Z",
  },
  {
    codigo: "2.0.01.00.01",
    descritor: "Organogramas",
    fase_corrente_anos: 1,
    fase_interm_anos: 1,
    destinacao_final: "ELIMINACAO",
    observacoes: "Versões anteriores",
    created_at: "2026-10-01T00:00:00Z",
  },
];

describe("Módulo Atlas (Frontend)", () => {
  beforeEach(() => resetNavigation());

  it("renderiza catálogo de fluxos SEI e permite inspecionar etapas e peças", async () => {
    mockBackend({
      ...identityRoutes(),
      "GET v1/atlas/workflows": { data: MOCK_WORKFLOWS },
      "GET v1/atlas/ttdd": { data: MOCK_TTDD },
    });

    renderApp(<AtlasPage />);

    expect(await screen.findByText("Atlas — Catálogo Normativo & Procedural")).toBeInTheDocument();
    expect(screen.getByText("ADM.LIC.001")).toBeInTheDocument();
    expect(screen.getByText("Processo Licitatório de Pregão Eletrônico")).toBeInTheDocument();

    // Clica no card do workflow para ver detalhes da trilha SEI
    await userEvent.click(screen.getByText("ADM.LIC.001"));

    expect(await screen.findByText("Trilha de Tramitação SEI (1 Etapas)")).toBeInTheDocument();
    expect(screen.getByText("Setor Demandante")).toBeInTheDocument();
    expect(screen.getByText("Termo de Referência (TR)")).toBeInTheDocument();
    expect(screen.getByText(/Regra SEI: Unidade mantém processo aberto/)).toBeInTheDocument();
  });

  it("permite navegar para a aba TTDD e inspecionar códigos de temporalidade", async () => {
    mockBackend({
      ...identityRoutes(),
      "GET v1/atlas/workflows": { data: MOCK_WORKFLOWS },
      "GET v1/atlas/ttdd": { data: MOCK_TTDD },
    });

    renderApp(<AtlasPage />);

    // Clica na aba TTDD
    await userEvent.click(await screen.findByRole("button", { name: /Tabela de Temporalidade/ }));

    expect(await screen.findByText("2.0.02.00.07")).toBeInTheDocument();
    expect(screen.getByText("GUARDA_PERMANENTE")).toBeInTheDocument();
    expect(screen.getByText("2.0.01.00.01")).toBeInTheDocument();
    expect(screen.getByText("ELIMINACAO")).toBeInTheDocument();
  });

  it("permite enviar pergunta ao assistente de IA com resposta canônica", async () => {
    mockBackend({
      ...identityRoutes(),
      "GET v1/atlas/workflows": { data: MOCK_WORKFLOWS },
      "GET v1/atlas/ttdd": { data: MOCK_TTDD },
      "POST v1/atlas/chat": {
        data: {
          answer: "Orientação para Pregão Eletrônico: instruir com DFD, ETP e TR.",
          score: 0.88,
          refused: false,
          workflows: MOCK_WORKFLOWS,
          generated_at: "2026-10-01T00:00:00Z",
        },
      },
    });

    renderApp(<AtlasPage />);

    // Clica na aba de IA
    await userEvent.click(await screen.findByRole("button", { name: /Assistente Procedural IA/ }));

    const input = screen.getByPlaceholderText(/Pergunte sobre como tramitar um processo/);
    await userEvent.type(input, "Como tramitar Pregão?");
    await userEvent.click(screen.getByRole("button", { name: "" })); // Botão de envio (ícone Send)

    expect(await screen.findByText("Orientação para Pregão Eletrônico: instruir com DFD, ETP e TR.")).toBeInTheDocument();
    expect(screen.getByText("Relevância Factual:")).toBeInTheDocument();
  });
});
