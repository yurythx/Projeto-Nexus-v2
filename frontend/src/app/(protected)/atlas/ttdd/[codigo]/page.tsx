"use client";

import { Archive, Clock, FolderClock } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";

import { AcoesPagina } from "@/components/atlas/AcoesPagina";
import { CalculadoraTemporalidade } from "@/components/atlas/CalculadoraTemporalidade";
import { SeloVigencia } from "@/components/atlas/SeloVigencia";
import { Trilha } from "@/components/atlas/Trilha";
import { WorkflowList } from "@/components/atlas/WorkflowList";
import { destinacao, fase } from "@/components/atlas/labels";
import { DataState } from "@/components/nexus/DataState";
import { Badge } from "@/components/ui/Badge";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import type { ClassificacaoTTDD, Workflow } from "@/lib/nexus/types";

function Prazo({
  titulo,
  icone,
  children,
}: {
  titulo: string;
  icone: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-surface-border bg-surface p-4">
      <dt className="flex items-center gap-1.5 text-xs font-semibold text-muted">
        {icone} {titulo}
      </dt>
      <dd className="mt-1 text-lg font-semibold text-foreground">{children}</dd>
    </div>
  );
}

/** Procedimentos enquadrados nesta série (filtro codigo_ttdd). */
function Procedimentos({ codigo }: { codigo: string }) {
  const list = useApiPage<Workflow>(
    withQuery("v1/atlas/workflows", { codigo_ttdd: codigo, page_size: 20 }),
  );
  const items = list.data?.items ?? [];
  return (
    <section aria-labelledby="atlas-serie-procs">
      <h2 id="atlas-serie-procs" className="mb-3 text-lg font-bold text-foreground">
        Procedimentos que geram esta série
      </h2>
      <DataState
        loading={list.isLoading}
        error={list.error}
        onRetry={() => void list.mutate()}
        empty={items.length === 0}
        emptyTitle="Nenhum procedimento cadastrado nesta série"
      >
        <WorkflowList items={items} />
      </DataState>
    </section>
  );
}

/** Página da série documental: prazos, destinação, recomendação, fonte
 * oficial, calculadora e os procedimentos que a produzem. */
export default function SeriePage() {
  const { codigo: bruto } = useParams<{ codigo: string }>();
  const codigo = decodeURIComponent(bruto);
  const serie = useApiQuery<ClassificacaoTTDD>(`v1/atlas/ttdd/${encodeURIComponent(codigo)}`);
  const c = serie.data;
  const sub = c?.subfuncao;
  const dest = c && destinacao(c.destinacao_final);
  const total =
    c && c.fase_corrente_anos !== null && c.fase_interm_anos !== null
      ? c.fase_corrente_anos + c.fase_interm_anos
      : null;

  return (
    <div className="flex flex-col gap-6">
      <Trilha
        itens={[
          { label: "Atlas", href: "/atlas" },
          { label: "Tabela de Temporalidade", href: "/atlas/ttdd" },
          ...(sub
            ? [
                {
                  label: sub.funcao.orgao.nome,
                  href: `/atlas/ttdd?codigo=${sub.funcao.orgao.prefixo}`,
                },
                { label: sub.funcao.nome, href: `/atlas/ttdd?codigo=${sub.funcao.codigo}` },
                { label: sub.nome, href: `/atlas/ttdd?codigo=${sub.codigo}` },
              ]
            : []),
          { label: codigo },
        ]}
      />
      <DataState
        loading={serie.isLoading}
        error={serie.error}
        empty={!c}
        emptyTitle="Série não encontrada na TTDD"
        onRetry={() => void serie.mutate()}
      >
        {c && dest && (
          <>
            <header className="flex flex-col gap-3">
              <span className="font-mono text-sm font-bold text-primary">{c.codigo}</span>
              <h1 className="text-2xl font-bold text-foreground">{c.descritor}</h1>
              {sub && (
                <p className="text-sm text-muted">
                  {sub.funcao.nome} › {sub.nome}
                </p>
              )}
              <SeloVigencia orgao={sub?.funcao.orgao} />
              <AcoesPagina
                rotulo="Perguntar sobre esta série"
                pergunta={`Por quanto tempo guardar "${c.descritor}" (${c.codigo}) e qual a destinação?`}
              />
            </header>

            <dl className="grid gap-3 sm:grid-cols-3">
              <Prazo titulo="Fase corrente" icone={<Clock size={14} aria-hidden="true" />}>
                {fase(c.fase_corrente_anos, c.fase_corrente_condicao, true)}
              </Prazo>
              <Prazo
                titulo="Fase intermediária"
                icone={<FolderClock size={14} aria-hidden="true" />}
              >
                {fase(c.fase_interm_anos, c.fase_interm_condicao, false)}
              </Prazo>
              <Prazo titulo="Destinação final" icone={<Archive size={14} aria-hidden="true" />}>
                <Badge tone={dest.tone}>{dest.label}</Badge>
              </Prazo>
            </dl>
            {total !== null && (
              <p className="text-sm text-muted">
                Guarda total antes da destinação:{" "}
                <strong className="text-foreground">
                  {total === 1 ? "1 ano" : `${total} anos`}
                </strong>
                .
              </p>
            )}

            {(c.observacoes || sub?.recomendacao) && (
              <div className="flex flex-col gap-2">
                {c.observacoes && (
                  <p className="text-sm text-foreground">
                    <strong>Observações:</strong> {c.observacoes}
                  </p>
                )}
                {sub?.recomendacao && (
                  <p className="rounded-md bg-warning/10 px-3 py-2 text-sm text-foreground">
                    <strong>Recomendação da subfunção:</strong> {sub.recomendacao}
                  </p>
                )}
              </div>
            )}

            <div className="max-w-xl">
              <CalculadoraTemporalidade serie={c} />
            </div>

            <Procedimentos codigo={c.codigo} />

            <p className="text-xs text-muted print:hidden">
              <Link
                href={sub ? `/atlas/ttdd?codigo=${encodeURIComponent(sub.codigo)}` : "/atlas/ttdd"}
                className="text-primary hover:underline"
              >
                Ver as demais séries {sub ? `de ${sub.nome}` : "da tabela"}
              </Link>
            </p>
          </>
        )}
      </DataState>
    </div>
  );
}
