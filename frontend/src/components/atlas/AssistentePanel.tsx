"use client";

import Link from "next/link";
import { Send, Shield } from "lucide-react";
import { useRef, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { ApiError, apiClient } from "@/lib/api/client";
import type { AtlasFonte, AtlasResposta } from "@/lib/nexus/types";

type Mensagem =
  | { id: number; autor: "usuario"; texto: string }
  | { id: number; autor: "assistente"; texto: string; resposta?: AtlasResposta; erro?: boolean };

const MODO: Record<AtlasResposta["mode"], string> = {
  ia: "Redigida por IA a partir das fontes homologadas",
  sintese: "Síntese direta das fontes homologadas",
  recusada: "Sem fonte homologada que sustente a resposta",
};

const SUGESTOES = [
  "Quais peças são obrigatórias no pregão eletrônico?",
  "Qual o prazo de guarda da pasta funcional do servidor?",
  "Como transferir material permanente entre unidades?",
];

/** Link da fonte: procedimento ou série da TTDD (página própria). */
export function hrefFonte(f: AtlasFonte): string {
  return f.tipo === "procedimento"
    ? `/atlas/procedimentos/${f.id}`
    : `/atlas/ttdd/${encodeURIComponent(f.codigo)}`;
}

function Fontes({ fontes, onNavegar }: { fontes: AtlasFonte[]; onNavegar?: () => void }) {
  if (fontes.length === 0) return null;
  return (
    <p className="mt-1">
      Fontes:{" "}
      {fontes.map((f, i) => (
        <span key={`${f.tipo}:${f.codigo}`}>
          {i > 0 && ", "}
          <Link
            href={hrefFonte(f)}
            title={f.titulo}
            onClick={onNavegar}
            className="font-mono text-primary hover:underline"
          >
            {f.codigo}
          </Link>
          <span className="sr-only">
            {f.tipo === "ttdd" ? " (série da TTDD)" : " (procedimento)"}
          </span>
        </span>
      ))}
    </p>
  );
}

/** Assistente procedural (atlas:read): só responde a partir de procedimentos
 * homologados e de séries da TTDD oficial acima do limiar de relevância;
 * abaixo dele, a recusa padronizada. */
export function AssistentePanel({
  perguntaInicial = "",
  onNavegar,
}: {
  perguntaInicial?: string;
  onNavegar?: () => void;
}) {
  const [input, setInput] = useState(perguntaInicial);
  const [loading, setLoading] = useState(false);
  // Ids das mensagens (0 é a de boas-vindas).
  const seq = useRef(1);
  const [mensagens, setMensagens] = useState<Mensagem[]>([
    {
      id: 0,
      autor: "assistente",
      texto:
        "Olá! Respondo só com base nos procedimentos homologados e na Tabela de Temporalidade oficial. Pergunte sobre etapas, prazos de guarda, peças exigidas ou destinação de documentos.",
    },
  ]);

  async function enviar(texto?: string) {
    const pergunta = (texto ?? input).trim();
    if (pergunta.length < 3 || loading) return;
    const base = seq.current;
    seq.current += 2;
    setMensagens((m) => [...m, { id: base, autor: "usuario", texto: pergunta }]);
    if (!texto) setInput("");
    setLoading(true);
    try {
      const { data } = await apiClient.post<AtlasResposta>("v1/atlas/chat", { query: pergunta });
      setMensagens((m) => [
        ...m,
        { id: base + 1, autor: "assistente", texto: data.answer, resposta: data },
      ]);
    } catch (err) {
      const texto =
        err instanceof ApiError
          ? err.message
          : "Não foi possível consultar o assistente agora. Tente novamente em instantes.";
      setMensagens((m) => [...m, { id: base + 1, autor: "assistente", texto, erro: true }]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <section aria-label="Conversa com o assistente" className="flex min-h-0 flex-1 flex-col">
      <p className="flex items-start gap-2 border-b border-surface-border bg-primary/5 px-4 py-2 text-xs text-muted">
        <Shield size={14} aria-hidden="true" className="mt-0.5 shrink-0 text-primary" />
        Só responde quando uma fonte homologada cobre a pergunta (relevância mínima de 65%); sem
        ela, recusa — nunca inventa. Confira sempre as fontes citadas.
      </p>

      <div
        className="flex-1 space-y-4 overflow-y-auto p-4"
        role="log"
        aria-live="polite"
        aria-busy={loading}
      >
        {mensagens.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.autor === "usuario" ? "justify-end" : "justify-start"}`}
          >
            <div
              className={`max-w-[90%] rounded-lg px-4 py-3 text-sm leading-relaxed ${
                msg.autor === "usuario"
                  ? "bg-primary text-primary-foreground"
                  : msg.autor === "assistente" && (msg.erro || msg.resposta?.refused)
                    ? "border border-warning/40 bg-warning/10 text-foreground"
                    : "border border-surface-border bg-surface-hover text-foreground"
              }`}
            >
              <span className="sr-only">{msg.autor === "usuario" ? "Você: " : "Assistente: "}</span>
              <div className="whitespace-pre-wrap">{msg.texto}</div>
              {msg.autor === "assistente" && msg.resposta && (
                <div className="mt-3 border-t border-surface-border pt-2 text-[11px] text-muted">
                  <p>
                    {MODO[msg.resposta.mode]} · relevância {(msg.resposta.score * 100).toFixed(0)}%
                  </p>
                  <Fontes fontes={msg.resposta.sources} onNavegar={onNavegar} />
                </div>
              )}
            </div>
          </div>
        ))}
        {loading && <p className="text-sm text-muted">Consultando as fontes homologadas…</p>}
        {mensagens.length === 1 && !loading && (
          <div>
            <h3 className="text-xs font-bold uppercase tracking-wider text-muted">Sugestões</h3>
            <ul className="mt-2 flex flex-col gap-2">
              {SUGESTOES.map((s) => (
                <li key={s}>
                  <button
                    type="button"
                    onClick={() => void enviar(s)}
                    className="w-full rounded-md border border-surface-border bg-surface-hover p-2.5 text-left text-xs text-foreground transition-colors hover:border-primary/40"
                  >
                    {s}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <form
        className="flex gap-2 border-t border-surface-border p-3"
        onSubmit={(e) => {
          e.preventDefault();
          void enviar();
        }}
      >
        <label htmlFor="atlas-pergunta" className="sr-only">
          Sua pergunta
        </label>
        <div className="flex-1">
          <Input
            id="atlas-pergunta"
            placeholder="Pergunte sobre etapas, prazos de guarda ou documentos…"
            value={input}
            maxLength={500}
            onChange={(e) => setInput(e.target.value)}
            disabled={loading}
          />
        </div>
        <Button
          type="submit"
          aria-label="Enviar pergunta"
          disabled={loading || input.trim().length < 3}
        >
          <Send size={16} aria-hidden="true" />
        </Button>
      </form>
    </section>
  );
}
