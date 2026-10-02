export type DestinacaoFinal = "GUARDA_PERMANENTE" | "ELIMINACAO";
export type NivelAcesso = "PUBLICO" | "RESTRITO" | "SIGILOSO";
export type FormatoDocumento = "NATO_DIGITAL" | "EXTERNO_DIGITALIZADO";
export type TipoAssinatura = "INDIVIDUAL" | "CONJUNTA_MULTINIVEL" | "EM_BLOCO";

export interface ClassificacaoTTDD {
  codigo: string;
  descritor: string;
  fase_corrente_anos: number;
  fase_interm_anos: number;
  destinacao_final: DestinacaoFinal;
  observacoes?: string;
  created_at: string;
}

export interface EtapaDocumento {
  id: string;
  etapa_id: string;
  nome_documento: string;
  obrigatorio: boolean;
  formato: FormatoDocumento;
  tipo_assinatura: TipoAssinatura;
  exige_conferencia_copia: boolean;
  modelo_minuta_padrao_url?: string;
}

export interface EtapaTransicao {
  id: string;
  origem_etapa_id: string;
  destino_etapa_id: string;
  condicao_transicao: string;
  is_devolucao_diligencia: boolean;
  descricao_diligencia?: string;
}

export interface Etapa {
  id: string;
  workflow_id: string;
  ordem: number;
  unidade_administrativa: string;
  nome_setor: string;
  atribuicoes_setor: string;
  prazo_sla_em_dias: number;
  manter_aberto_apos_remessa: boolean;
  documentos?: EtapaDocumento[];
  transicoes?: EtapaTransicao[];
}

export interface Workflow {
  id: string;
  tenant_id: string;
  codigo_processual: string;
  titulo: string;
  objetivo: string;
  publico_alvo: string;
  versao: number;
  ativo: boolean;
  nivel_acesso: NivelAcesso;
  hipotese_legal_restricao?: string;
  codigo_ttdd: string;
  classificacao?: ClassificacaoTTDD;
  etapas?: Etapa[];
  created_at: string;
  updated_at: string;
}

export interface ProceduralChatResponse {
  answer: string;
  score: number;
  refused: boolean;
  workflows?: Workflow[];
  generated_at: string;
}
