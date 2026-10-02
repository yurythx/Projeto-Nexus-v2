"use client";

import { Search } from "lucide-react";
import { useState } from "react";

import { DataState } from "@/components/nexus/DataState";
import { Pagination } from "@/components/nexus/Pagination";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@/components/ui/Table";
import { useApiPage, withQuery } from "@/lib/api/swr";
import type { ClassificacaoTTDD } from "@/lib/nexus/types";

import { DESTINACAO } from "./labels";

/** Tabela de Temporalidade e Destinação de Documentos (consulta). */
export function TTDDTable() {
  const [q, setQ] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const list = useApiPage<ClassificacaoTTDD>(
    withQuery("v1/atlas/ttdd", { q: query, page, page_size: 50 }),
  );
  const items = list.data?.items ?? [];

  return (
    <div className="flex flex-col gap-4">
      <form
        role="search"
        className="flex flex-wrap items-end gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          setPage(1);
          setQuery(q.trim());
        }}
      >
        <Input
          id="atlas-ttdd-q"
          label="Código ou descritor"
          placeholder="Ex.: 2.0.02 ou pregão"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <Button type="submit" variant="secondary">
          <Search size={14} aria-hidden="true" className="mr-1" /> Buscar
        </Button>
      </form>

      <DataState
        loading={list.isLoading}
        error={list.error}
        onRetry={() => void list.mutate()}
        empty={items.length === 0}
        emptyTitle="Nenhuma classificação encontrada"
      >
        <Table caption="Tabela de Temporalidade e Destinação de Documentos">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Código</TableHeaderCell>
              <TableHeaderCell>Descritor</TableHeaderCell>
              <TableHeaderCell>Fase corrente</TableHeaderCell>
              <TableHeaderCell>Fase intermediária</TableHeaderCell>
              <TableHeaderCell>Destinação final</TableHeaderCell>
              <TableHeaderCell>Observações</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {items.map((c) => (
              <TableRow key={c.codigo}>
                <TableCell className="font-mono text-xs font-bold text-primary">
                  {c.codigo}
                </TableCell>
                <TableCell className="font-medium">{c.descritor}</TableCell>
                <TableCell className="text-xs">{c.fase_corrente_anos} ano(s)</TableCell>
                <TableCell className="text-xs">{c.fase_interm_anos} ano(s)</TableCell>
                <TableCell>
                  <Badge tone={DESTINACAO[c.destinacao_final].tone}>
                    {DESTINACAO[c.destinacao_final].label}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs text-muted">{c.observacoes || "—"}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination meta={list.data?.meta} onPage={setPage} />
      </DataState>
    </div>
  );
}
