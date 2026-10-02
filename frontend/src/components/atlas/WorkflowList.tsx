"use client";

import { Badge } from "@/components/ui/Badge";
import type { Workflow } from "@/lib/nexus/types";

import { NIVEL_ACESSO } from "./labels";

/** Lista de procedimentos: cada item é um botão (teclado e leitor de tela —
 * e-MAG 2.1/WCAG 2.1.1), com aria-pressed marcando o selecionado. */
export function WorkflowList({
  items,
  selectedId,
  onSelect,
}: {
  items: Workflow[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  return (
    <ul className="flex flex-col gap-3" aria-label="Procedimentos">
      {items.map((wf) => {
        const selected = wf.id === selectedId;
        const nivel = NIVEL_ACESSO[wf.nivel_acesso];
        return (
          <li key={wf.id}>
            <button
              type="button"
              aria-pressed={selected}
              onClick={() => onSelect(wf.id)}
              className={`w-full rounded-lg border p-4 text-left transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary ${
                selected
                  ? "border-primary bg-primary/5"
                  : "border-surface-border bg-surface hover:bg-surface-hover"
              }`}
            >
              <span className="flex items-start justify-between gap-2">
                <span className="font-mono text-xs font-bold text-primary">
                  {wf.codigo_processual}
                </span>
                <span className="flex gap-1">
                  {!wf.ativo && <Badge tone="danger">Inativo</Badge>}
                  <Badge tone={nivel.tone}>{nivel.label}</Badge>
                </span>
              </span>
              <span className="mt-1 block font-semibold text-foreground">{wf.titulo}</span>
              <span className="mt-1 line-clamp-2 block text-xs text-muted">{wf.objetivo}</span>
              <span className="mt-3 flex items-center justify-between border-t border-surface-border pt-2 text-xs text-muted">
                <span className="font-mono">TTDD {wf.codigo_ttdd}</span>
                <span>
                  {wf.total_etapas} {wf.total_etapas === 1 ? "etapa" : "etapas"}
                </span>
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
