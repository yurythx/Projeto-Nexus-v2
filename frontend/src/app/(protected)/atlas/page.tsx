"use client";

import { Plus, Search, Workflow as WorkflowIcon } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";

import { AssistentePanel } from "@/components/atlas/AssistentePanel";
import { NovoProcedimentoForm } from "@/components/atlas/NovoProcedimentoForm";
import { TTDDTable } from "@/components/atlas/TTDDTable";
import { WorkflowDetail } from "@/components/atlas/WorkflowDetail";
import { WorkflowList } from "@/components/atlas/WorkflowList";
import { DataState } from "@/components/nexus/DataState";
import { PageHeader } from "@/components/nexus/PageHeader";
import { Pagination } from "@/components/nexus/Pagination";
import { SectionTabsInline } from "@/components/nexus/SectionTabsInline";
import { Button } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { useApiPage, withQuery } from "@/lib/api/swr";
import { useNexus } from "@/lib/nexus/NexusProvider";
import type { Workflow } from "@/lib/nexus/types";

type Aba = "procedimentos" | "ttdd" | "assistente";

/** Aba de procedimentos: busca, lista paginada e detalhe do selecionado
 * (?procedimento=<id> — também é o link da Busca Global). */
function Procedimentos({ canManage }: { canManage: boolean }) {
  const params = useSearchParams();
  const router = useRouter();
  const selectedId = params.get("procedimento");
  const [q, setQ] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  // A gestão lista pela rota administrativa (inclui os desativados).
  const base = canManage ? "v1/atlas/admin/workflows" : "v1/atlas/workflows";
  const list = useApiPage<Workflow>(withQuery(base, { q: query, page, page_size: 20 }));
  const items = list.data?.items ?? [];

  const select = (id: string) => router.replace(`/atlas?procedimento=${id}`);

  return (
    <div className="grid gap-6 lg:grid-cols-12">
      <div className="flex flex-col gap-4 lg:col-span-5">
        <form
          role="search"
          className="flex items-end gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setPage(1);
            setQuery(q.trim());
          }}
        >
          <div className="flex-1">
            <Input
              id="atlas-q"
              label="Código, título ou objetivo"
              value={q}
              onChange={(e) => setQ(e.target.value)}
            />
          </div>
          <Button type="submit" variant="secondary" aria-label="Buscar procedimentos">
            <Search size={14} aria-hidden="true" />
          </Button>
        </form>
        <DataState
          loading={list.isLoading}
          error={list.error}
          onRetry={() => void list.mutate()}
          empty={items.length === 0}
          emptyTitle="Nenhum procedimento encontrado"
          emptyDescription="Revise os termos da busca."
        >
          <WorkflowList items={items} selectedId={selectedId} onSelect={select} />
          <Pagination meta={list.data?.meta} onPage={setPage} />
        </DataState>
      </div>

      <div className="lg:col-span-7">
        {selectedId ? (
          <WorkflowDetail
            key={selectedId}
            id={selectedId}
            canManage={canManage}
            onChanged={() => void list.mutate()}
          />
        ) : (
          <div className="flex h-96 flex-col items-center justify-center rounded-lg border border-dashed border-surface-border p-8 text-center text-muted">
            <WorkflowIcon size={40} aria-hidden="true" />
            <p className="mt-3 font-semibold text-foreground">Selecione um procedimento</p>
            <p className="mt-1 max-w-sm text-xs">
              Veja as etapas, os prazos e as peças exigidas em cada setor.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

function Atlas() {
  const { can } = useNexus();
  const router = useRouter();
  // ?ttdd=<código>: link de uma fonte do assistente — abre a TTDD filtrada.
  const buscaTTDD = useSearchParams().get("ttdd") ?? "";
  const canManage = can("atlas:manage");
  const canAsk = can("atlas:read");
  const [aba, setAba] = useState<Aba>(buscaTTDD ? "ttdd" : "procedimentos");
  const [creating, setCreating] = useState(false);
  // Remonta a lista após um cadastro (busca de novo).
  const [versao, setVersao] = useState(0);

  const tabs: { value: Aba; label: string }[] = [
    { value: "procedimentos", label: "Procedimentos" },
    { value: "ttdd", label: "Tabela de Temporalidade (TTDD)" },
    ...(canAsk ? [{ value: "assistente" as const, label: "Assistente procedural" }] : []),
  ];

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atlas"
        title="Procedimentos e temporalidade"
        description="Roteiros canônicos de processo administrativo (padrão SEI), Tabela de Temporalidade e Destinação de Documentos e assistente procedural."
        actions={
          canManage && (
            <Button onClick={() => setCreating(true)}>
              <Plus size={16} aria-hidden="true" className="mr-1" /> Novo procedimento
            </Button>
          )
        }
      />

      <SectionTabsInline tabs={tabs} value={aba} onChange={setAba} label="Seções do Atlas" />

      <div role="tabpanel" aria-label={tabs.find((t) => t.value === aba)?.label}>
        {aba === "procedimentos" && <Procedimentos key={versao} canManage={canManage} />}
        {aba === "ttdd" && <TTDDTable key={buscaTTDD} busca={buscaTTDD} />}
        {aba === "assistente" && canAsk && <AssistentePanel />}
      </div>

      <Dialog
        open={creating}
        onClose={() => setCreating(false)}
        title="Novo procedimento"
        description="Cabeçalho, enquadramento na TTDD e etapas."
        size="lg"
      >
        {creating && (
          <NovoProcedimentoForm
            onDone={(wf) => {
              setCreating(false);
              setAba("procedimentos");
              setVersao((v) => v + 1);
              router.replace(`/atlas?procedimento=${wf.id}`);
            }}
          />
        )}
      </Dialog>
    </div>
  );
}

export default function AtlasPage() {
  return (
    <Suspense>
      <Atlas />
    </Suspense>
  );
}
