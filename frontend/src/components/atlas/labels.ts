import type {
  DestinacaoFinal,
  FormatoDocumento,
  NivelAcesso,
  TipoAssinatura,
} from "@/lib/nexus/types";

type Tone = "neutral" | "success" | "warning" | "danger" | "info";

export const NIVEL_ACESSO: Record<NivelAcesso, { label: string; tone: Tone }> = {
  PUBLICO: { label: "Público", tone: "neutral" },
  RESTRITO: { label: "Restrito", tone: "warning" },
  SIGILOSO: { label: "Sigiloso", tone: "danger" },
};

export const DESTINACAO: Record<DestinacaoFinal, { label: string; tone: Tone }> = {
  GUARDA_PERMANENTE: { label: "Guarda permanente", tone: "success" },
  ELIMINACAO: { label: "Eliminação", tone: "warning" },
};

export const FORMATO: Record<FormatoDocumento, string> = {
  NATO_DIGITAL: "Nato-digital",
  EXTERNO_DIGITALIZADO: "Externo digitalizado",
};

export const ASSINATURA: Record<TipoAssinatura, string> = {
  INDIVIDUAL: "Individual",
  CONJUNTA_MULTINIVEL: "Conjunta (multinível)",
  EM_BLOCO: "Em bloco",
};
