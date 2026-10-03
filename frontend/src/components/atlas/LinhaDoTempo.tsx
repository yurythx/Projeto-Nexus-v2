import {
  AlertTriangle,
  ArrowRight,
  Clock,
  CornerDownLeft,
  ExternalLink,
  FileDigit,
  FileText,
} from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import type { Etapa } from "@/lib/nexus/types";

import { ASSINATURA, FORMATO } from "./labels";

/** Prazo previsto: soma dos prazos (SLA) das etapas, em dias. */
export const prazoTotalDias = (etapas: Etapa[]) =>
  etapas.reduce((t, e) => t + e.prazo_sla_em_dias, 0);

/** Percurso do procedimento como linha do tempo vertical: setor, prazo e
 * transições de cada etapa (desvios de diligência destacados). */
export function LinhaDoTempo({ etapas }: { etapas: Etapa[] }) {
  return (
    <ol className="relative flex flex-col gap-6 border-l-2 border-primary/30 pl-6">
      {etapas.map((e) => (
        <li key={e.id} className="relative">
          <span
            aria-hidden="true"
            className="absolute -left-[2.05rem] top-0 flex h-7 w-7 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground"
          >
            {e.ordem}
          </span>
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h3 className="font-semibold text-foreground">
              <span className="sr-only">Etapa {e.ordem}: </span>
              {e.nome_setor}{" "}
              <span className="font-mono text-xs font-normal text-muted">
                ({e.unidade_administrativa})
              </span>
            </h3>
            <Badge tone="neutral" className="gap-1">
              <Clock size={12} aria-hidden="true" /> {e.prazo_sla_em_dias}{" "}
              {e.prazo_sla_em_dias === 1 ? "dia" : "dias"}
            </Badge>
          </div>
          <p className="mt-1 text-sm text-muted">{e.atribuicoes_setor}</p>
          {e.manter_aberto_apos_remessa && (
            <p className="mt-2 flex items-center gap-1 text-xs font-medium text-warning">
              <AlertTriangle size={12} aria-hidden="true" /> A unidade mantém o processo aberto após
              a remessa
            </p>
          )}
          {e.transicoes.length > 0 && (
            <ul className="mt-2 flex flex-col gap-1 text-xs">
              {e.transicoes.map((t) => (
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
          )}
        </li>
      ))}
    </ol>
  );
}

/** Checklist das peças de todas as etapas, com o modelo quando houver. */
export function ChecklistDocumentos({ etapas }: { etapas: Etapa[] }) {
  const pecas = etapas.flatMap((e) => e.documentos.map((d) => ({ ...d, etapa: e })));
  if (pecas.length === 0)
    return <p className="text-sm text-muted">O procedimento não lista peças obrigatórias.</p>;
  return (
    <ul className="flex flex-col gap-2">
      {pecas.map((d) => (
        <li
          key={d.id}
          className="flex flex-wrap items-start justify-between gap-2 rounded-md border border-surface-border p-2.5 text-sm"
        >
          <span className="flex items-start gap-2">
            {d.formato === "NATO_DIGITAL" ? (
              <FileDigit size={16} aria-hidden="true" className="mt-0.5 shrink-0 text-primary" />
            ) : (
              <FileText size={16} aria-hidden="true" className="mt-0.5 shrink-0 text-muted" />
            )}
            <span>
              <span className="font-medium text-foreground">{d.nome_documento}</span>
              <span className="block text-xs text-muted">
                Etapa {d.etapa.ordem} · {FORMATO[d.formato]} · assinatura{" "}
                {ASSINATURA[d.tipo_assinatura].toLowerCase()}
              </span>
            </span>
          </span>
          <span className="flex flex-wrap items-center gap-1">
            <Badge tone={d.obrigatorio ? "info" : "neutral"}>
              {d.obrigatorio ? "Obrigatória" : "Opcional"}
            </Badge>
            {d.exige_conferencia_copia && <Badge tone="warning">Conferência da cópia</Badge>}
            {d.modelo_minuta_padrao_url && (
              <a
                href={d.modelo_minuta_padrao_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
              >
                Modelo <ExternalLink size={11} aria-hidden="true" />
                <span className="sr-only">(abre em nova janela)</span>
              </a>
            )}
          </span>
        </li>
      ))}
    </ul>
  );
}
