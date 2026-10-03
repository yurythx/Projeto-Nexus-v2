"use client";

import { Search } from "lucide-react";
import { useState } from "react";

import { DataState } from "@/components/nexus/DataState";
import { Pagination } from "@/components/nexus/Pagination";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@/components/ui/Table";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import type { ClassificacaoTTDD, EstruturaTTDD } from "@/lib/nexus/types";

import { destinacao, fase, fonteTTDD } from "./labels";

/** Tabela de Temporalidade e Destinação de Documentos oficial (consulta):
 * filtro por órgão e função (árvore da TTDD), busca por código ou descritor. */
export function TTDDTable({ busca = "" }: { busca?: string }) {
  const [q, setQ] = useState(busca);
  const [query, setQuery] = useState(busca);
  const [orgao, setOrgao] = useState("");
  const [funcao, setFuncao] = useState("");
  const [page, setPage] = useState(1);
  const estrutura = useApiQuery<EstruturaTTDD[]>("v1/atlas/ttdd/estrutura");
  const org = (estrutura.data ?? []).find((o) => o.prefixo === orgao);
  const list = useApiPage<ClassificacaoTTDD>(
    withQuery("v1/atlas/ttdd", { q: query, codigo: funcao || orgao, page, page_size: 50 }),
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
        <Select
          id="atlas-ttdd-orgao"
          label="Órgão"
          value={orgao}
          onChange={(e) => {
            setOrgao(e.target.value);
            setFuncao("");
            setPage(1);
          }}
          options={[
            { value: "", label: "Todos os órgãos" },
            ...(estrutura.data ?? []).map((o) => ({
              value: o.prefixo,
              label: `${o.prefixo} — ${o.nome} (${o.total})`,
            })),
          ]}
        />
        {org && (
          <Select
            id="atlas-ttdd-funcao"
            label="Função"
            value={funcao}
            onChange={(e) => {
              setFuncao(e.target.value);
              setPage(1);
            }}
            options={[
              { value: "", label: "Todas as funções" },
              ...org.funcoes.map((f) => ({
                value: f.codigo,
                label: `${f.codigo} — ${f.nome} (${f.total})`,
              })),
            ]}
          />
        )}
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

      {org && <p className="text-xs text-muted">Fonte: {fonteTTDD(org)}.</p>}

      <DataState
        loading={list.isLoading}
        error={list.error}
        onRetry={() => void list.mutate()}
        empty={items.length === 0}
        emptyTitle="Nenhuma série encontrada"
      >
        <Table caption="Tabela de Temporalidade e Destinação de Documentos">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Código</TableHeaderCell>
              <TableHeaderCell>Série documental</TableHeaderCell>
              <TableHeaderCell>Fase corrente</TableHeaderCell>
              <TableHeaderCell>Fase intermediária</TableHeaderCell>
              <TableHeaderCell>Destinação final</TableHeaderCell>
              <TableHeaderCell>Observações</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {items.map((c) => {
              const dest = destinacao(c.destinacao_final);
              return (
                <TableRow key={c.codigo}>
                  <TableCell className="whitespace-nowrap font-mono text-xs font-bold text-primary">
                    {c.codigo}
                  </TableCell>
                  <TableCell>
                    <span className="font-medium">{c.descritor}</span>
                    {c.subfuncao && (
                      <span className="block text-xs text-muted">
                        {c.subfuncao.funcao.nome} › {c.subfuncao.nome}
                      </span>
                    )}
                    {c.subfuncao?.recomendacao && (
                      <span className="mt-1 block text-xs text-warning">
                        Recomendação: {c.subfuncao.recomendacao}
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-xs">
                    {fase(c.fase_corrente_anos, c.fase_corrente_condicao, true)}
                  </TableCell>
                  <TableCell className="text-xs">
                    {fase(c.fase_interm_anos, c.fase_interm_condicao, false)}
                  </TableCell>
                  <TableCell>
                    <Badge tone={dest.tone}>{dest.label}</Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted">{c.observacoes || "—"}</TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
        <Pagination meta={list.data?.meta} onPage={setPage} />
      </DataState>
    </div>
  );
}
