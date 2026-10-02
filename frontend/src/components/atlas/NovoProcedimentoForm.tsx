"use client";

import { Plus, Trash2 } from "lucide-react";
import { useState, type FormEvent } from "react";

import { useAction } from "@/components/nexus/useAction";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage } from "@/lib/api/swr";
import type { ClassificacaoTTDD, NivelAcesso, Workflow } from "@/lib/nexus/types";

import { NIVEL_ACESSO } from "./labels";

interface EtapaForm {
  key: number;
  nome_setor: string;
  unidade_administrativa: string;
  atribuicoes_setor: string;
  prazo_sla_em_dias: number;
  manter_aberto_apos_remessa: boolean;
  /** Uma peça por linha (nato-digital, assinatura individual, obrigatória). */
  pecas: string;
}

const novaEtapa = (key: number): EtapaForm => ({
  key,
  nome_setor: "",
  unidade_administrativa: "",
  atribuicoes_setor: "",
  prazo_sla_em_dias: 5,
  manter_aberto_apos_remessa: false,
  pecas: "",
});

/** Cadastro de procedimento (atlas:manage): cabeçalho, enquadramento na
 * TTDD e as etapas em ordem. Regras detalhadas (formatos, assinaturas,
 * transições) seguem a API — o formulário cobre o caso comum. */
export function NovoProcedimentoForm({ onDone }: { onDone: (wf: Workflow) => void }) {
  const ttdd = useApiPage<ClassificacaoTTDD>("v1/atlas/ttdd?page_size=200");
  const { run, pending } = useAction();
  const [nivel, setNivel] = useState<NivelAcesso>("PUBLICO");
  const [etapas, setEtapas] = useState<EtapaForm[]>([novaEtapa(1)]);

  function update(key: number, patch: Partial<EtapaForm>) {
    setEtapas((list) => list.map((e) => (e.key === key ? { ...e, ...patch } : e)));
  }

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const res = await run(
      () =>
        apiClient.post<Workflow>("v1/atlas/admin/workflows", {
          codigo_processual: String(fd.get("codigo_processual") ?? "").trim(),
          titulo: String(fd.get("titulo") ?? "").trim(),
          objetivo: String(fd.get("objetivo") ?? "").trim(),
          publico_alvo: String(fd.get("publico_alvo") ?? "").trim(),
          nivel_acesso: nivel,
          hipotese_legal_restricao:
            nivel === "PUBLICO" ? "" : String(fd.get("hipotese_legal") ?? "").trim(),
          codigo_ttdd: String(fd.get("codigo_ttdd") ?? ""),
          etapas: etapas.map((etapa, i) => ({
            ordem: i + 1,
            nome_setor: etapa.nome_setor.trim(),
            unidade_administrativa: etapa.unidade_administrativa.trim(),
            atribuicoes_setor: etapa.atribuicoes_setor.trim(),
            prazo_sla_em_dias: etapa.prazo_sla_em_dias,
            manter_aberto_apos_remessa: etapa.manter_aberto_apos_remessa,
            documentos: etapa.pecas
              .split("\n")
              .map((p) => p.trim())
              .filter(Boolean)
              .map((nome) => ({
                nome_documento: nome,
                obrigatorio: true,
                formato: "NATO_DIGITAL",
                tipo_assinatura: "INDIVIDUAL",
              })),
            transicoes: [],
          })),
        }),
      "Procedimento cadastrado",
    );
    if (res) onDone(res.data);
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <Input
          id="atlas-codigo"
          name="codigo_processual"
          label="Código processual *"
          placeholder="Ex.: ADM.FIN.021"
          required
          maxLength={64}
        />
        <Select
          id="atlas-nivel"
          label="Nível de acesso sugerido *"
          value={nivel}
          onChange={(e) => setNivel(e.target.value as NivelAcesso)}
          options={Object.entries(NIVEL_ACESSO).map(([value, n]) => ({ value, label: n.label }))}
        />
      </div>
      {nivel !== "PUBLICO" && (
        <Input
          id="atlas-hipotese"
          name="hipotese_legal"
          label="Hipótese legal da restrição *"
          placeholder="Ex.: LAI, art. 31"
          required
          maxLength={255}
        />
      )}
      <Input
        id="atlas-titulo"
        name="titulo"
        label="Título do procedimento *"
        required
        maxLength={255}
      />
      <Textarea
        id="atlas-objetivo"
        name="objetivo"
        label="Objetivo *"
        rows={2}
        required
        maxLength={5000}
      />
      <div className="grid gap-3 sm:grid-cols-2">
        <Input
          id="atlas-publico"
          name="publico_alvo"
          label="Público-alvo *"
          placeholder="Ex.: Secretarias municipais"
          required
          maxLength={255}
        />
        <Select
          id="atlas-ttdd"
          name="codigo_ttdd"
          label="Classificação TTDD *"
          required
          placeholder="Selecione…"
          options={(ttdd.data?.items ?? []).map((t) => ({
            value: t.codigo,
            label: `${t.codigo} — ${t.descritor}`,
          }))}
        />
      </div>

      <fieldset className="flex flex-col gap-3 border-t border-surface-border pt-3">
        <legend className="text-xs font-semibold uppercase tracking-wide text-foreground">
          Etapas, na ordem de tramitação
        </legend>
        {etapas.map((etapa, i) => (
          <div key={etapa.key} className="rounded-lg border border-surface-border p-3">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-sm font-semibold text-foreground">Etapa {i + 1}</span>
              {etapas.length > 1 && (
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  aria-label={`Remover etapa ${i + 1}`}
                  onClick={() => setEtapas((list) => list.filter((e) => e.key !== etapa.key))}
                >
                  <Trash2 size={14} aria-hidden="true" />
                </Button>
              )}
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <Input
                id={`atlas-setor-${etapa.key}`}
                label="Setor *"
                required
                maxLength={150}
                value={etapa.nome_setor}
                onChange={(e) => update(etapa.key, { nome_setor: e.target.value })}
              />
              <Input
                id={`atlas-sigla-${etapa.key}`}
                label="Sigla da unidade *"
                placeholder="Ex.: SEMAD/LIC"
                required
                maxLength={64}
                value={etapa.unidade_administrativa}
                onChange={(e) => update(etapa.key, { unidade_administrativa: e.target.value })}
              />
            </div>
            <div className="mt-3 grid gap-3 sm:grid-cols-[1fr_8rem]">
              <Input
                id={`atlas-atrib-${etapa.key}`}
                label="Atribuições *"
                required
                maxLength={5000}
                value={etapa.atribuicoes_setor}
                onChange={(e) => update(etapa.key, { atribuicoes_setor: e.target.value })}
              />
              <Input
                id={`atlas-prazo-${etapa.key}`}
                label="Prazo (dias) *"
                type="number"
                min={0}
                max={3650}
                required
                value={etapa.prazo_sla_em_dias}
                onChange={(e) => update(etapa.key, { prazo_sla_em_dias: Number(e.target.value) })}
              />
            </div>
            <div className="mt-3">
              <Textarea
                id={`atlas-pecas-${etapa.key}`}
                label="Peças exigidas (uma por linha)"
                rows={2}
                value={etapa.pecas}
                onChange={(e) => update(etapa.key, { pecas: e.target.value })}
              />
            </div>
            <label className="mt-2 flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                className="h-4 w-4 accent-primary"
                checked={etapa.manter_aberto_apos_remessa}
                onChange={(e) =>
                  update(etapa.key, { manter_aberto_apos_remessa: e.target.checked })
                }
              />
              Manter o processo aberto na unidade após a remessa
            </label>
          </div>
        ))}
        <div>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() =>
              setEtapas((list) => [...list, novaEtapa(Math.max(...list.map((e) => e.key)) + 1)])
            }
          >
            <Plus size={14} aria-hidden="true" className="mr-1" /> Adicionar etapa
          </Button>
        </div>
      </fieldset>

      <div className="flex justify-end border-t border-surface-border pt-3">
        <Button type="submit" loading={pending}>
          Cadastrar procedimento
        </Button>
      </div>
    </form>
  );
}
