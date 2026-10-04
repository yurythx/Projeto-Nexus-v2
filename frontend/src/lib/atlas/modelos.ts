import type { UUID } from "@/lib/nexus/types";

/** Formatos aceitos na biblioteca (o backend confere o conteúdo). */
export const FORMATOS_MODELO = ".docx,.odt,.pdf,.doc,.rtf,.xlsx,.ods";

/** Limite do arquivo de um modelo (o mesmo do backend). */
export const MAX_MODELO_BYTES = 10 * 1024 * 1024;

/** Download pela API (mesma origem): a versão atual ou uma específica. */
export function urlArquivoModelo(id: UUID, versao?: number): string {
  const base = `/api/backend/v1/atlas/modelos/${encodeURIComponent(id)}/arquivo`;
  return versao ? `${base}?versao=${versao}` : base;
}

/** Extensão em maiúsculas para o selo do formato ("DOCX"). */
export function extensao(nome: string): string {
  const i = nome.lastIndexOf(".");
  return i < 0 ? "" : nome.slice(i + 1).toUpperCase();
}

/** Normaliza para comparar nomes (sem acento, minúsculas). */
function chave(s: string): string {
  return s
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .trim();
}

/** Sugere o modelo de mesmo nome da peça (sem acento e caixa). */
export function sugerirModelo<T extends { id: UUID; nome: string }>(
  peca: string,
  modelos: T[],
): T | undefined {
  const k = chave(peca);
  return k ? modelos.find((m) => chave(m.nome) === k) : undefined;
}
