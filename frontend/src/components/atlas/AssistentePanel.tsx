"use client";

import { Bot, Send, Shield } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { ApiError, apiClient } from "@/lib/api/client";
import type { AtlasFonte, AtlasResposta } from "@/lib/nexus/types";

type Mensagem =
  | { id: number; autor: "usuario"; texto: string }
  | { id: number; autor: "assistente"; texto: string; resposta?: AtlasResposta; erro?: boolean };

const MODO: Record<AtlasResposta["mode"], string> = {
  ia: "Redigida por IA a partir dos procedimentos homologados",
  sintese: "Síntese direta dos procedimentos homologados",
  recusada: "Sem procedimento homologado que sustente a resposta",
};

const SUGESTOES = [
  "Quais peças são obrigatórias no pregão eletrônico?",
  "Quais os prazos da TTDD para dispensa de licitação?",
  "Como transferir material permanente entre unidades?",
  "Qual o prazo de guarda da pasta funcional do servidor?",
];

function Fontes({ fontes }: { fontes: AtlasFonte[] }) {
  if (fontes.length === 0) return null;
  return (
    <p className="mt-1">
      Fontes:{" "}
      {fontes.map((f, i) => (
        <span key={`${f.tipo}:${f.codigo}`}>
          {i > 0 && ", "}
          <a
            href={
              f.tipo === "procedimento"
                ? `/atlas?procedimento=${f.id}`
                : `/atlas?ttdd=${encodeURIComponent(f.codigo)}`
            }
            title={f.titulo}
            className="font-mono text-primary hover:underline"
          >
            {f.codigo}
          </a>
          {f.tipo === "ttdd" && <span className="sr-only"> (série da TTDD)</span>}
        </span>
      ))}
    </p>
  );
}

/** Assistente procedural (atlas:read). Respostas só a partir de
 * procedimentos homologados acima do limiar de relevância; abaixo dele, a
 * recusa padronizada. */
export function AssistentePanel() {
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [mensagens, setMensagens] = useState<Mensagem[]>([
    {
      id: 0,
      autor: "assistente",
      texto:
        "Olá! Respondo com base exclusivamente nos procedimentos homologados e na Tabela de Temporalidade. Pergunte sobre etapas, prazos, peças exigidas ou destinação de documentos.",
    },
  ]);

  async function enviar(texto?: string) {
    const pergunta = (texto ?? input).trim();
    if (pergunta.length < 3 || loading) return;
    const base = Date.now();
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
    <div className="grid gap-6 lg:grid-cols-12">
      <section
        aria-label="Conversa com o assistente"
        className="flex h-[600px] flex-col rounded-lg border border-surface-border bg-surface shadow-sm lg:col-span-8"
      >
        <header className="flex items-center gap-2 border-b border-surface-border px-4 py-3">
          <Bot size={18} aria-hidden="true" className="text-primary" />
          <h3 className="font-bold text-foreground">Assistente procedural</h3>
        </header>

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
                className={`max-w-[85%] rounded-lg px-4 py-3 text-sm leading-relaxed ${
                  msg.autor === "usuario"
                    ? "bg-primary text-primary-foreground"
                    : msg.autor === "assistente" && (msg.erro || msg.resposta?.refused)
                      ? "border border-warning/40 bg-warning/10 text-foreground"
                      : "border border-surface-border bg-surface-hover text-foreground"
                }`}
              >
                <span className="sr-only">
                  {msg.autor === "usuario" ? "Você: " : "Assistente: "}
                </span>
                <div className="whitespace-pre-wrap">{msg.texto}</div>
                {msg.autor === "assistente" && msg.resposta && (
                  <div className="mt-3 border-t border-surface-border pt-2 text-[11px] text-muted">
                    <p>
                      {MODO[msg.resposta.mode]} · relevância {(msg.resposta.score * 100).toFixed(0)}
                      %
                    </p>
                    <Fontes fontes={msg.resposta.sources} />
                  </div>
                )}
              </div>
            </div>
          ))}
          {loading && (
            <p className="text-sm text-muted">Consultando os procedimentos homologados…</p>
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
              placeholder="Pergunte sobre etapas, prazos ou documentos de um procedimento…"
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

      <aside className="flex flex-col gap-4 lg:col-span-4">
        <div className="rounded-lg border border-primary/20 bg-primary/5 p-4">
          <h3 className="flex items-center gap-2 text-sm font-bold text-foreground">
            <Shield size={16} aria-hidden="true" className="text-primary" />
            Como o assistente responde
          </h3>
          <p className="mt-2 text-xs leading-relaxed text-muted">
            Só responde quando um procedimento homologado cobre a pergunta (relevância mínima de
            65%). Sem essa base, devolve uma recusa padronizada — nunca uma orientação inventada.
            Confira sempre as fontes citadas.
          </p>
        </div>
        <div className="rounded-lg border border-surface-border bg-surface p-4">
          <h3 className="text-xs font-bold uppercase tracking-wider text-muted">Sugestões</h3>
          <ul className="mt-3 flex flex-col gap-2">
            {SUGESTOES.map((s) => (
              <li key={s}>
                <button
                  type="button"
                  disabled={loading}
                  onClick={() => void enviar(s)}
                  className="w-full rounded-md border border-surface-border bg-surface-hover p-2.5 text-left text-xs text-foreground transition-colors hover:border-primary/40"
                >
                  {s}
                </button>
              </li>
            ))}
          </ul>
        </div>
      </aside>
    </div>
  );
}
