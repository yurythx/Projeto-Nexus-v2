"use client";

import {
  AlertTriangle,
  ArrowRight,
  Clock,
  CornerDownLeft,
  ExternalLink,
  FileDigit,
  FileText,
  Workflow as WorkflowIcon,
} from "lucide-react";

import { DataState } from "@/components/nexus/DataState";
import { useAction } from "@/components/nexus/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { Etapa, Workflow } from "@/lib/nexus/types";

import { ASSINATURA, DESTINACAO, FORMATO, NIVEL_ACESSO } from "./labels";

function EtapaCard({ etapa }: { etapa: Etapa }) {
  return (
    <li className="rounded-lg border border-surface-border bg-surface p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <h4 className="flex items-center gap-2 font-semibold text-foreground">
          <span
            className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary"
            aria-hidden="true"
          >
            {etapa.ordem}
          </span>
          <span className="sr-only">Etapa {etapa.ordem}: </span>
          {etapa.nome_setor}
          <span className="font-mono text-xs font-normal text-muted">
            ({etapa.unidade_administrativa})
          </span>
        </h4>
        <Badge tone="neutral" className="gap-1">
          <Clock size={12} aria-hidden="true" /> Prazo: {etapa.prazo_sla_em_dias}{" "}
          {etapa.prazo_sla_em_dias === 1 ? "dia" : "dias"}
        </Badge>
      </div>
      <p className="mt-2 text-sm text-muted">{etapa.atribuicoes_setor}</p>

      {etapa.manter_aberto_apos_remessa && (
        <p className="mt-2 flex items-center gap-1 rounded bg-warning/10 px-2 py-1 text-xs font-medium text-warning">
          <AlertTriangle size={12} aria-hidden="true" />
          Regra SEI: a unidade mantém o processo aberto para acompanhamento após a remessa
        </p>
      )}

      {etapa.documentos.length > 0 && (
        <div className="mt-3 border-t border-surface-border pt-3">
          <h5 className="text-xs font-semibold text-muted">Peças exigidas</h5>
          <ul className="mt-1.5 flex flex-col gap-1.5">
            {etapa.documentos.map((doc) => (
              <li
                key={doc.id}
                className="flex flex-wrap items-center justify-between gap-2 rounded bg-surface-hover px-2.5 py-1.5 text-xs"
              >
                <span className="flex flex-wrap items-center gap-1.5">
                  {doc.formato === "NATO_DIGITAL" ? (
                    <FileDigit size={14} aria-hidden="true" className="text-primary" />
                  ) : (
                    <FileText size={14} aria-hidden="true" className="text-muted" />
                  )}
                  <span className="font-medium text-foreground">{doc.nome_documento}</span>
                  {!doc.obrigatorio && <Badge tone="neutral">Opcional</Badge>}
                  {doc.exige_conferencia_copia && (
                    <Badge tone="warning">Exige conferência da cópia</Badge>
                  )}
                  {doc.modelo_minuta_padrao_url && (
                    <a
                      href={doc.modelo_minuta_padrao_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-primary hover:underline"
                    >
                      Modelo <ExternalLink size={11} aria-hidden="true" />
                      <span className="sr-only">(abre em nova janela)</span>
                    </a>
                  )}
                </span>
                <span className="text-muted">
                  {FORMATO[doc.formato]} · Assinatura{" "}
                  {ASSINATURA[doc.tipo_assinatura].toLowerCase()}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {etapa.transicoes.length > 0 && (
        <div className="mt-3 border-t border-surface-border pt-2 text-xs">
          <h5 className="font-semibold text-muted">Transições</h5>
          <ul className="mt-1 flex flex-col gap-1">
            {etapa.transicoes.map((t) => (
              <li
                key={t.id}
                className={`flex items-start gap-1.5 ${t.is_devolucao_diligencia ? "text-warning" : "text-muted"}`}
              >
                {t.is_devolucao_diligencia ? (
                  <CornerDownLeft size={12} aria-hidden="true" className="mt-0.5 shrink-0" />
                ) : (
                  <ArrowRight size={12} aria-hidden="true" className="mt-0.5 shrink-0" />
                )}
                <span>
                  {t.condicao_transicao} → etapa {t.destino_ordem}
                  {t.is_devolucao_diligencia && (
                    <strong> (diligência: {t.descricao_diligencia})</strong>
                  )}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </li>
  );
}

/** Detalhe do procedimento: busca o percurso completo (a listagem não traz
 * etapas). A gestão (atlas:manage) usa a rota administrativa, que também
 * mostra os desativados, e pode ativar/desativar. */
export function WorkflowDetail({
  id,
  canManage,
  onChanged,
}: {
  id: string;
  canManage: boolean;
  onChanged: () => void;
}) {
  const path = canManage ? `v1/atlas/admin/workflows/${id}` : `v1/atlas/workflows/${id}`;
  const detail = useApiQuery<Workflow>(path);
  const { run, pending } = useAction();
  const wf = detail.data;

  async function toggle(ativo: boolean) {
    const res = await run(
      () =>
        apiClient.post<Workflow>(
          `v1/atlas/admin/workflows/${id}/${ativo ? "ativar" : "desativar"}`,
        ),
      ativo ? "Procedimento ativado" : "Procedimento desativado",
    );
    if (res) {
      await detail.mutate(res.data, { revalidate: false });
      onChanged();
    }
  }

  return (
    <DataState
      loading={detail.isLoading}
      error={detail.error}
      empty={!wf}
      emptyTitle="Procedimento não encontrado"
      onRetry={() => void detail.mutate()}
    >
      {wf && (
        <article
          className="flex flex-col gap-6 rounded-lg border border-surface-border bg-surface p-6 shadow-sm"
          aria-labelledby="atlas-wf-titulo"
        >
          <header>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-mono text-sm font-bold text-primary">
                {wf.codigo_processual} · versão {wf.versao}
              </span>
              <span className="flex flex-wrap gap-1">
                <Badge tone={NIVEL_ACESSO[wf.nivel_acesso].tone}>
                  {NIVEL_ACESSO[wf.nivel_acesso].label}
                </Badge>
                {!wf.ativo && <Badge tone="danger">Inativo</Badge>}
              </span>
            </div>
            <h3 id="atlas-wf-titulo" className="mt-2 text-xl font-bold text-foreground">
              {wf.titulo}
            </h3>
            <p className="mt-2 text-sm text-muted">{wf.objetivo}</p>

            <dl className="mt-4 grid gap-3 rounded-lg bg-surface-hover p-4 text-xs sm:grid-cols-2">
              <div>
                <dt className="font-semibold text-muted">Público-alvo</dt>
                <dd className="text-foreground">{wf.publico_alvo}</dd>
              </div>
              <div>
                <dt className="font-semibold text-muted">Classificação TTDD</dt>
                <dd className="font-mono text-foreground">
                  {wf.codigo_ttdd}
                  {wf.classificacao && ` — ${wf.classificacao.descritor}`}
                </dd>
              </div>
              {wf.classificacao && (
                <div className="sm:col-span-2">
                  <dt className="font-semibold text-muted">Temporalidade</dt>
                  <dd className="text-foreground">
                    {wf.classificacao.fase_corrente_anos} ano(s) na fase corrente,{" "}
                    {wf.classificacao.fase_interm_anos} ano(s) na intermediária —{" "}
                    {DESTINACAO[wf.classificacao.destinacao_final].label.toLowerCase()}
                  </dd>
                </div>
              )}
              {wf.nivel_acesso !== "PUBLICO" && (
                <div className="sm:col-span-2">
                  <dt className="font-semibold text-muted">Hipótese legal da restrição</dt>
                  <dd className="text-foreground">{wf.hipotese_legal_restricao}</dd>
                </div>
              )}
            </dl>

            {canManage && (
              <div className="mt-4 flex justify-end">
                {wf.ativo ? (
                  <Button
                    variant="secondary"
                    size="sm"
                    loading={pending}
                    onClick={() => void toggle(false)}
                  >
                    Desativar procedimento
                  </Button>
                ) : (
                  <Button size="sm" loading={pending} onClick={() => void toggle(true)}>
                    Ativar procedimento
                  </Button>
                )}
              </div>
            )}
          </header>

          <section aria-labelledby="atlas-wf-trilha">
            <h3 id="atlas-wf-trilha" className="flex items-center gap-2 font-bold text-foreground">
              <WorkflowIcon size={16} aria-hidden="true" className="text-primary" />
              Trilha de tramitação ({wf.etapas.length} {wf.etapas.length === 1 ? "etapa" : "etapas"}
              )
            </h3>
            <ol className="mt-4 flex flex-col gap-4">
              {wf.etapas.map((etapa) => (
                <EtapaCard key={etapa.id} etapa={etapa} />
              ))}
            </ol>
          </section>
        </article>
      )}
    </DataState>
  );
}
