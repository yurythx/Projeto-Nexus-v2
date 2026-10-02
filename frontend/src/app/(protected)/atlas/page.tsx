"use client";

import { useState, type FormEvent } from "react";
import {
  AlertTriangle,
  ArrowRight,
  Bot,
  Clock,
  FileDigit,
  FileText,
  Layers,
  Plus,
  Search,
  Send,
  Shield,
  Sparkles,
  Workflow as WorkflowIcon,
} from "lucide-react";

import { DataState } from "@/components/nexus/DataState";
import { PageHeader } from "@/components/nexus/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { useNexus } from "@/lib/nexus/NexusProvider";
import type {
  ClassificacaoTTDD,
  NivelAcesso,
  ProceduralChatResponse,
  Workflow,
} from "@/types/atlas";

export default function AtlasPage() {
  const { can } = useNexus();
  const canManage = can("atlas:manage") || can("*");

  const [activeTab, setActiveTab] = useState<"workflows" | "ttdd" | "ia">("workflows");

  // Filtros de Workflows
  const [searchWf, setSearchWf] = useState("");
  const [selectedWf, setSelectedWf] = useState<Workflow | null>(null);

  // Filtros de TTDD
  const [searchTTDD, setSearchTTDD] = useState("");

  // Modal de Criação de Procedimento
  const [createOpen, setCreateOpen] = useState(false);
  const [createPending, setCreatePending] = useState(false);
  const [createError, setCreateError] = useState("");

  // Estado do Chat de IA
  const [chatInput, setChatInput] = useState("");
  const [chatLoading, setChatLoading] = useState(false);
  const [chatHistory, setChatHistory] = useState<
    Array<{
      sender: "user" | "bot";
      text: string;
      score?: number;
      refused?: boolean;
      sources?: Workflow[];
    }>
  >([
    {
      sender: "bot",
      text: "Olá, servidor! Sou o Assistente Procedural do Módulo Atlas. Minhas respostas são estritamente fundamentadas na Tabela de Temporalidade e Destinação de Documentos (TTDD) e nos fluxos SEI homologados pela Prefeitura. Como posso orientar seu processo hoje?",
    },
  ]);

  // Consultas à API
  const workflowsQuery = useApiQuery<Workflow[]>(
    `v1/atlas/workflows?q=${encodeURIComponent(searchWf)}`
  );
  const ttddQuery = useApiQuery<ClassificacaoTTDD[]>(
    `v1/atlas/ttdd?q=${encodeURIComponent(searchTTDD)}`
  );

  async function handleSendChat(customQuery?: string) {
    const q = (customQuery ?? chatInput).trim();
    if (!q || chatLoading) return;

    setChatHistory((prev) => [...prev, { sender: "user", text: q }]);
    if (!customQuery) setChatInput("");
    setChatLoading(true);

    try {
      const res = await apiClient.post<ProceduralChatResponse>("v1/atlas/chat", {
        query: q,
        tenant_id: "nexus",
      });

      if (res.data) {
        setChatHistory((prev) => [
          ...prev,
          {
            sender: "bot",
            text: res.data.answer,
            score: res.data.score,
            refused: res.data.refused,
            sources: res.data.workflows,
          },
        ]);
      }
    } catch {
      setChatHistory((prev) => [
        ...prev,
        {
          sender: "bot",
          text: "Houve uma instabilidade ao conectar com o serviço de orientação procedural. Por favor, tente novamente em instantes.",
        },
      ]);
    } finally {
      setChatLoading(false);
    }
  }

  async function handleCreateWorkflow(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setCreateError("");
    setCreatePending(true);

    const fd = new FormData(e.currentTarget);
    const codigoProcessual = String(fd.get("codigo_processual") ?? "").trim();
    const titulo = String(fd.get("titulo") ?? "").trim();
    const objetivo = String(fd.get("objetivo") ?? "").trim();
    const publicoAlvo = String(fd.get("publico_alvo") ?? "").trim();
    const nivelAcesso = (String(fd.get("nivel_acesso") ?? "PUBLICO") as NivelAcesso);
    const codigoTTDD = String(fd.get("codigo_ttdd") ?? "").trim();
    const setorNome = String(fd.get("setor_nome") ?? "").trim();
    const setorUnidade = String(fd.get("setor_unidade") ?? "").trim();
    const setorAtribuicoes = String(fd.get("setor_atribuicoes") ?? "").trim();
    const slaDias = Number(fd.get("sla_dias") ?? 5);

    try {
      await apiClient.post("v1/atlas/workflows", {
        codigo_processual: codigoProcessual,
        titulo: titulo,
        objetivo: objetivo,
        publico_alvo: publicoAlvo,
        nivel_acesso: nivelAcesso,
        codigo_ttdd: codigoTTDD,
        etapas: [
          {
            ordem: 1,
            unidade_administrativa: setorUnidade,
            nome_setor: setorNome,
            atribuicoes_setor: setorAtribuicoes,
            prazo_sla_em_dias: slaDias,
            manter_aberto_apos_remessa: false,
            documentos: [
              {
                nome_documento: "Termo de Abertura / Requerimento Inicial",
                obrigatorio: true,
                formato: "NATO_DIGITAL",
                tipo_assinatura: "INDIVIDUAL",
                exige_conferencia_copia: false,
              },
            ],
            transicoes: [],
          },
        ],
      });

      await workflowsQuery.mutate();
      setCreateOpen(false);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Erro ao cadastrar procedimento.";
      setCreateError(msg);
    } finally {
      setCreatePending(false);
    }
  }

  const workflows = workflowsQuery.data ?? [];
  const ttddList = ttddQuery.data ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Atlas — Catálogo Normativo & Procedural"
        description="Mapeamento canônico de processos SEI, Tabela de Temporalidade (TTDD / CCPAD) e Orientação Procedural com IA sob Grounding Estrito."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Badge tone="info">Padrão SEI Brasil</Badge>
            <Badge tone="success">TTDD CCPAD (Rondonópolis)</Badge>
            <Badge tone="warning">Grounding IA &gt;= 0.65</Badge>
            {canManage && (
              <Button
                variant="primary"
                onClick={() => setCreateOpen(true)}
                className="ml-2 flex items-center gap-1.5"
              >
                <Plus className="h-4 w-4" />
                Novo Procedimento
              </Button>
            )}
          </div>
        }
      />

      {/* Abas de Navegação */}
      <div className="flex border-b border-slate-200">
        <button
          type="button"
          onClick={() => setActiveTab("workflows")}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
            activeTab === "workflows"
              ? "border-blue-600 text-blue-700"
              : "border-transparent text-slate-600 hover:text-slate-900"
          }`}
        >
          <WorkflowIcon className="h-4 w-4" />
          Fluxos SEI & Procedimentos ({workflows.length})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("ttdd")}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
            activeTab === "ttdd"
              ? "border-blue-600 text-blue-700"
              : "border-transparent text-slate-600 hover:text-slate-900"
          }`}
        >
          <Layers className="h-4 w-4" />
          Tabela de Temporalidade (TTDD) ({ttddList.length})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("ia")}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
            activeTab === "ia"
              ? "border-purple-600 text-purple-700"
              : "border-transparent text-slate-600 hover:text-slate-900"
          }`}
        >
          <Sparkles className="h-4 w-4 text-purple-600" />
          Assistente Procedural IA (Grounding Estrito)
        </button>
      </div>

      {/* ABA 1: WORKFLOWS SEI */}
      {activeTab === "workflows" && (
        <div className="grid gap-6 lg:grid-cols-12">
          {/* Coluna da Esquerda: Lista de Workflows */}
          <div className="space-y-4 lg:col-span-5">
            <div className="relative">
              <Search className="absolute left-3 top-3 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Buscar por código, título ou objetivo..."
                value={searchWf}
                onChange={(e) => setSearchWf(e.target.value)}
                className="pl-9"
              />
            </div>

            <DataState
              loading={workflowsQuery.isLoading}
              error={workflowsQuery.error}
              empty={workflows.length === 0}
              emptyTitle="Nenhum procedimento processual localizado"
              emptyDescription="Verifique os termos da busca ou cadastre novos fluxos."
            >
              <div className="space-y-3">
                {workflows.map((wf) => {
                  const isSelected = selectedWf?.id === wf.id;
                  return (
                    <div
                      key={wf.id}
                      onClick={() => setSelectedWf(wf)}
                      className={`cursor-pointer rounded-lg border p-4 transition-all hover:shadow-sm ${
                        isSelected
                          ? "border-blue-500 bg-blue-50/50 shadow-sm ring-1 ring-blue-500"
                          : "border-slate-200 bg-white hover:border-slate-300"
                      }`}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <span className="font-mono text-xs font-bold text-blue-700">
                          {wf.codigo_processual}
                        </span>
                        <Badge tone={wf.nivel_acesso === "PUBLICO" ? "neutral" : "warning"}>
                          {wf.nivel_acesso}
                        </Badge>
                      </div>
                      <h4 className="mt-1 font-semibold text-slate-900">{wf.titulo}</h4>
                      <p className="mt-1 line-clamp-2 text-xs text-slate-600">{wf.objetivo}</p>
                      <div className="mt-3 flex items-center justify-between border-t border-slate-100 pt-2 text-xs text-slate-500">
                        <span className="flex items-center gap-1 font-mono">
                          TTDD: {wf.codigo_ttdd}
                        </span>
                        <span>{wf.etapas?.length ?? 0} etapas</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </DataState>
          </div>

          {/* Coluna da Direita: Detalhe do Fluxo SEI Selecionado */}
          <div className="lg:col-span-7">
            {selectedWf ? (
              <div className="space-y-6 rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
                <div>
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="font-mono text-sm font-bold text-blue-700">
                      {selectedWf.codigo_processual} • Versão {selectedWf.versao}
                    </span>
                    <Badge tone="success">Homologado CCPAD</Badge>
                  </div>
                  <h3 className="mt-2 text-xl font-bold text-slate-900">{selectedWf.titulo}</h3>
                  <p className="mt-2 text-sm text-slate-700">{selectedWf.objetivo}</p>

                  <div className="mt-4 grid gap-3 sm:grid-cols-2 rounded-lg bg-slate-50 p-4 text-xs">
                    <div>
                      <span className="font-semibold text-slate-500">Público-alvo:</span>
                      <p className="text-slate-800">{selectedWf.publico_alvo}</p>
                    </div>
                    <div>
                      <span className="font-semibold text-slate-500">Classificação TTDD:</span>
                      <p className="font-mono text-slate-800">
                        {selectedWf.codigo_ttdd}
                        {selectedWf.classificacao && ` — ${selectedWf.classificacao.descritor}`}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Percurso em Etapas */}
                <div>
                  <h4 className="flex items-center gap-2 font-bold text-slate-900">
                    <WorkflowIcon className="h-4 w-4 text-blue-600" />
                    Trilha de Tramitação SEI ({selectedWf.etapas?.length ?? 0} Etapas)
                  </h4>

                  <div className="mt-4 space-y-4">
                    {selectedWf.etapas?.map((etapa) => (
                      <div
                        key={etapa.id}
                        className="relative rounded-lg border border-slate-200 bg-white p-4 shadow-xs"
                      >
                        <div className="flex items-start justify-between gap-2">
                          <div className="flex items-center gap-2">
                            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-100 text-xs font-bold text-blue-800">
                              {etapa.ordem}
                            </span>
                            <span className="font-semibold text-slate-900">
                              {etapa.nome_setor}
                            </span>
                            <span className="font-mono text-xs text-slate-500">
                              ({etapa.unidade_administrativa})
                            </span>
                          </div>
                          <Badge tone="neutral" className="flex items-center gap-1 text-xs">
                            <Clock className="h-3 w-3" /> SLA: {etapa.prazo_sla_em_dias}d
                          </Badge>
                        </div>

                        <p className="mt-2 text-xs text-slate-600">{etapa.atribuicoes_setor}</p>

                        {etapa.manter_aberto_apos_remessa && (
                          <div className="mt-2 flex items-center gap-1 rounded bg-amber-50 px-2 py-1 text-xs font-medium text-amber-800">
                            <AlertTriangle className="h-3 w-3" />
                            Regra SEI: Unidade mantém processo aberto para acompanhamento após remessa
                          </div>
                        )}

                        {/* Peças / Documentos Obrigatórios */}
                        {etapa.documentos && etapa.documentos.length > 0 && (
                          <div className="mt-3 border-t border-slate-100 pt-3">
                            <span className="text-xs font-semibold text-slate-500">
                              Peças Autuadas Obrigatórias:
                            </span>
                            <div className="mt-1.5 space-y-1.5">
                              {etapa.documentos.map((doc) => (
                                <div
                                  key={doc.id}
                                  className="flex flex-wrap items-center justify-between gap-2 rounded bg-slate-50 px-2.5 py-1.5 text-xs"
                                >
                                  <div className="flex items-center gap-1.5">
                                    {doc.formato === "NATO_DIGITAL" ? (
                                      <FileDigit className="h-3.5 w-3.5 text-blue-600" />
                                    ) : (
                                      <FileText className="h-3.5 w-3.5 text-slate-600" />
                                    )}
                                    <span className="font-medium text-slate-800">
                                      {doc.nome_documento}
                                    </span>
                                    {doc.exige_conferencia_copia && (
                                      <Badge tone="warning" className="text-[10px]">
                                        Exige Atesto de Autenticidade
                                      </Badge>
                                    )}
                                  </div>
                                  <div className="flex items-center gap-2">
                                    <span className="text-[10px] text-slate-500">
                                      Assinatura: {doc.tipo_assinatura}
                                    </span>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {/* Regras de Transição e Diligências */}
                        {etapa.transicoes && etapa.transicoes.length > 0 && (
                          <div className="mt-3 border-t border-slate-100 pt-2 text-xs">
                            <span className="font-semibold text-slate-500">
                              Regras de Transição / Diligência:
                            </span>
                            <ul className="mt-1 space-y-1">
                              {etapa.transicoes.map((t) => (
                                <li
                                  key={t.id}
                                  className={`flex items-start gap-1.5 ${
                                    t.is_devolucao_diligencia
                                      ? "text-amber-800"
                                      : "text-slate-600"
                                  }`}
                                >
                                  <ArrowRight className="mt-0.5 h-3 w-3 shrink-0" />
                                  <span>
                                    {t.condicao_transicao}
                                    {t.is_devolucao_diligencia && (
                                      <span className="font-semibold">
                                        {" "}
                                        [Devolução em Diligência: {t.descricao_diligencia}]
                                      </span>
                                    )}
                                  </span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex h-96 flex-col items-center justify-center rounded-lg border border-dashed border-slate-300 p-8 text-center text-slate-500">
                <WorkflowIcon className="h-10 w-10 text-slate-400" />
                <h4 className="mt-3 font-semibold text-slate-700">Selecione um Procedimento</h4>
                <p className="mt-1 max-w-sm text-xs">
                  Escolha um workflow na listagem ao lado para inspecionar todas as etapas SEI, prazos SLA e peças documentais exigidas.
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ABA 2: TABELA DE TEMPORALIDADE (TTDD) */}
      {activeTab === "ttdd" && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="relative w-full max-w-md">
              <Search className="absolute left-3 top-3 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Buscar por código TTDD (ex: 2.0.02) ou descritor..."
                value={searchTTDD}
                onChange={(e) => setSearchTTDD(e.target.value)}
                className="pl-9"
              />
            </div>
            <span className="text-xs text-slate-500">
              Fonte Oficial: Diário Oficial de Rondonópolis nº 6.017 / CCPAD
            </span>
          </div>

          <div className="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-slate-700">
                <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold text-slate-600">
                  <tr>
                    <th className="px-4 py-3 font-mono">Código</th>
                    <th className="px-4 py-3">Descritor / Tipo Documental</th>
                    <th className="px-4 py-3 text-center">Fase Corrente</th>
                    <th className="px-4 py-3 text-center">Fase Intermediária</th>
                    <th className="px-4 py-3">Destinação Final</th>
                    <th className="px-4 py-3">Observações</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {ttddList.map((item) => (
                    <tr key={item.codigo} className="hover:bg-slate-50/70 transition-colors">
                      <td className="px-4 py-3 font-mono text-xs font-bold text-blue-700">
                        {item.codigo}
                      </td>
                      <td className="px-4 py-3 font-medium text-slate-900">{item.descritor}</td>
                      <td className="px-4 py-3 text-center text-xs">
                        {item.fase_corrente_anos} ano(s)
                      </td>
                      <td className="px-4 py-3 text-center text-xs">
                        {item.fase_interm_anos} ano(s)
                      </td>
                      <td className="px-4 py-3">
                        <Badge
                          tone={
                            item.destinacao_final === "GUARDA_PERMANENTE"
                              ? "success"
                              : "warning"
                          }
                        >
                          {item.destinacao_final}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-xs text-slate-500">
                        {item.observacoes || "—"}
                      </td>
                    </tr>
                  ))}
                  {ttddList.length === 0 && (
                    <tr>
                      <td colSpan={6} className="p-8 text-center text-slate-500">
                        Nenhum item da TTDD localizado para os critérios informados.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* ABA 3: ASSISTENTE PROCEDURAL COM IA */}
      {activeTab === "ia" && (
        <div className="grid gap-6 lg:grid-cols-12">
          {/* Painel do Chat */}
          <div className="flex h-[600px] flex-col rounded-lg border border-slate-200 bg-white shadow-sm lg:col-span-8">
            <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-4 py-3">
              <div className="flex items-center gap-2">
                <Bot className="h-5 w-5 text-purple-600" />
                <span className="font-bold text-slate-900">
                  Assistente Procedural com Grounding Estrito
                </span>
              </div>
              <Badge tone="info" className="text-[10px]">
                Temperatura: 0.05 • Threshold: 0.65
              </Badge>
            </div>

            {/* Mensagens do Chat */}
            <div className="flex-1 space-y-4 overflow-y-auto p-4">
              {chatHistory.map((msg, i) => (
                <div
                  key={i}
                  className={`flex ${msg.sender === "user" ? "justify-end" : "justify-start"}`}
                >
                  <div
                    className={`max-w-[85%] rounded-lg px-4 py-3 text-sm leading-relaxed ${
                      msg.sender === "user"
                        ? "bg-blue-600 text-white"
                        : msg.refused
                        ? "border border-amber-300 bg-amber-50/80 text-amber-900"
                        : "border border-slate-200 bg-slate-50 text-slate-800"
                    }`}
                  >
                    <div className="whitespace-pre-wrap">{msg.text}</div>

                    {msg.sender === "bot" && typeof msg.score === "number" && (
                      <div className="mt-3 flex flex-wrap items-center justify-between border-t border-slate-200/60 pt-2 text-[11px] text-slate-500">
                        <span>
                          Relevância Factual: <strong>{(msg.score * 100).toFixed(1)}%</strong>
                        </span>
                        {msg.sources && msg.sources.length > 0 && (
                          <span>
                            Fontes: {msg.sources.map((s) => s.codigo_processual).join(", ")}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {chatLoading && (
                <div className="flex justify-start">
                  <div className="flex items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-500">
                    <Sparkles className="h-4 w-4 animate-spin text-purple-600" />
                    Consultando matriz procedural do Typesense e validando threshold...
                  </div>
                </div>
              )}
            </div>

            {/* Campo de Entrada */}
            <div className="border-t border-slate-200 p-3">
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  handleSendChat();
                }}
                className="flex gap-2"
              >
                <Input
                  placeholder="Pergunte sobre como tramitar um processo, prazos TTDD ou documentos exigidos..."
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  disabled={chatLoading}
                />
                <Button type="submit" disabled={chatLoading || !chatInput.trim()}>
                  <Send className="h-4 w-4" />
                </Button>
              </form>
            </div>
          </div>

          {/* Painel Lateral de Sugestões e Regras de Segurança */}
          <div className="space-y-4 lg:col-span-4">
            <div className="rounded-lg border border-purple-200 bg-purple-50/60 p-4">
              <h4 className="flex items-center gap-2 font-bold text-purple-900 text-sm">
                <Shield className="h-4 w-4 text-purple-600" />
                Diretrizes de Resposta Canônica
              </h4>
              <p className="mt-2 text-xs leading-relaxed text-purple-800">
                O assistente opera sob <strong>grounding estrito</strong>. Perguntas sem correspondência no catálogo homologado com relevância mínima de 0.65 retornam recusa padronizada, impedindo qualquer alucinação.
              </p>
            </div>

            <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-xs">
              <h4 className="text-xs font-bold uppercase tracking-wider text-slate-500">
                Consultas Rápidas Sugeridas
              </h4>
              <div className="mt-3 space-y-2">
                {[
                  "Como tramitar um processo de Pregão Eletrônico e quais peças são obrigatórias?",
                  "Quais os prazos da TTDD para processos de Dispensa de Licitação?",
                  "Qual o procedimento e documentos para Transferência de Material Permanente?",
                  "Como protocolar um pedido de licença para viagens internacionais não homologadas?",
                ].map((sug, idx) => (
                  <button
                    key={idx}
                    type="button"
                    onClick={() => handleSendChat(sug)}
                    className="w-full text-left rounded-md border border-slate-100 bg-slate-50/70 p-2.5 text-xs text-slate-700 transition hover:border-blue-300 hover:bg-blue-50/50"
                  >
                    {sug}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* MODAL DE CRIAÇÃO DE PROCEDIMENTO (atlas:manage) */}
      <Dialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Cadastrar Novo Procedimento Canônico"
        description="Defina a tipologia SEI, enquadramento na TTDD e a etapa inaugural do percurso."
        size="lg"
      >
        <form onSubmit={handleCreateWorkflow} className="space-y-4">
          {createError && (
            <div className="rounded-md border border-danger/20 bg-danger/10 p-3 text-xs text-danger">
              {createError}
            </div>
          )}

          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              id="codigo_processual"
              name="codigo_processual"
              label="Código Processual *"
              placeholder="Ex: ADM.FIN.021"
              required
            />
            <Select
              id="nivel_acesso"
              name="nivel_acesso"
              label="Nível de Acesso *"
              defaultValue="PUBLICO"
              required
              options={[
                { value: "PUBLICO", label: "Público (amplo acesso)" },
                { value: "RESTRITO", label: "Restrito (hipótese legal)" },
                { value: "SIGILOSO", label: "Sigiloso (credenciamento nominal)" },
              ]}
            />
          </div>

          <Input
            id="titulo"
            name="titulo"
            label="Título do Procedimento *"
            placeholder="Ex: Prestação de Contas de Adiantamento Financeiro"
            required
          />

          <Textarea
            id="objetivo"
            name="objetivo"
            label="Objetivo do Procedimento *"
            placeholder="Descreva a finalidade administrativa e a fundamentação geral..."
            required
            rows={2}
          />

          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              id="publico_alvo"
              name="publico_alvo"
              label="Público-alvo *"
              placeholder="Ex: Secretarias Municipais e Supridos"
              required
            />
            <Select
              id="codigo_ttdd"
              name="codigo_ttdd"
              label="Código TTDD Vinculado *"
              required
              options={ttddList.map((t) => ({
                value: t.codigo,
                label: `${t.codigo} — ${t.descritor}`,
              }))}
            />
          </div>

          <div className="border-t border-slate-200 pt-3">
            <h5 className="font-semibold text-xs text-slate-800 uppercase tracking-wide">
              Etapa 1: Unidade Inaugural / Demandante
            </h5>
            <div className="mt-2 grid gap-3 sm:grid-cols-2">
              <Input
                id="setor_nome"
                name="setor_nome"
                label="Nome do Setor *"
                placeholder="Ex: Gabinete ou Setor Requisitante"
                required
              />
              <Input
                id="setor_unidade"
                name="setor_unidade"
                label="Sigla da Unidade *"
                placeholder="Ex: SEC/DEMANDANTE"
                required
              />
            </div>
            <div className="mt-2 grid gap-3 sm:grid-cols-2">
              <Input
                id="sla_dias"
                name="sla_dias"
                label="Prazo SLA (dias) *"
                type="number"
                defaultValue={5}
                required
              />
              <Input
                id="setor_atribuicoes"
                name="setor_atribuicoes"
                label="Atribuições da Etapa *"
                placeholder="Ex: Formalização da demanda e anexação de comprovantes"
                required
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t border-slate-200">
            <Button variant="secondary" type="button" onClick={() => setCreateOpen(false)}>
              Cancelar
            </Button>
            <Button variant="primary" type="submit" disabled={createPending}>
              {createPending ? "Salvando..." : "Cadastrar Procedimento"}
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}
