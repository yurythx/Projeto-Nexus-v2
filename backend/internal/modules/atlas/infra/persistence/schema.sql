-- ==============================================================================
-- MÓDULO ATLAS: ESQUEMA DDL CANÔNICO (PostgreSQL + pgx/v5)
-- Padrão SEI de Processo Administrativo Eletrônico e TTDD (CCPAD/CONARQ)
-- ==============================================================================

-- 1. Classificação Documental e Temporalidade (TTDD / CCPAD)
CREATE TABLE IF NOT EXISTS atlas_classificacao_ttdd (
    codigo VARCHAR(32) PRIMARY KEY, -- Ex: '02.01.03.01' ou '2.0.01.00.00'
    descritor VARCHAR(255) NOT NULL,
    fase_corrente_anos INT NOT NULL DEFAULT 1,
    fase_interm_anos INT NOT NULL DEFAULT 0,
    destinacao_final VARCHAR(32) NOT NULL CHECK (destinacao_final IN ('GUARDA_PERMANENTE', 'ELIMINACAO')),
    observacoes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Tabela Mestra de Workflows / Tipologias de Processo
CREATE TABLE IF NOT EXISTS atlas_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    codigo_processual VARCHAR(64) NOT NULL, -- Ex: 'ADM.FIN.021'
    titulo VARCHAR(255) NOT NULL,
    objetivo TEXT NOT NULL,
    publico_alvo VARCHAR(255) NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    nivel_acesso VARCHAR(32) NOT NULL CHECK (nivel_acesso IN ('PUBLICO', 'RESTRITO', 'SIGILOSO')),
    hipotese_legal_restricao VARCHAR(255),
    codigo_ttdd VARCHAR(32) NOT NULL REFERENCES atlas_classificacao_ttdd(codigo) ON UPDATE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_atlas_workflow_versao UNIQUE (tenant_id, codigo_processual, versao)
);
CREATE INDEX IF NOT EXISTS idx_atlas_wf_tenant ON atlas_workflows(tenant_id, ativo);

-- 3. Etapas do Fluxo (Nós de Tramitação)
CREATE TABLE IF NOT EXISTS atlas_etapas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES atlas_workflows(id) ON DELETE CASCADE,
    ordem INT NOT NULL,
    unidade_administrativa VARCHAR(64) NOT NULL, -- Ex: 'SEFIN/CONTAB'
    nome_setor VARCHAR(150) NOT NULL,
    atribuicoes_setor TEXT NOT NULL,
    prazo_sla_em_dias INT NOT NULL DEFAULT 2,
    manter_aberto_apos_remessa BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_atlas_etapa_ordem UNIQUE (workflow_id, ordem)
);
CREATE INDEX IF NOT EXISTS idx_atlas_etapa_wf ON atlas_etapas(workflow_id);

-- 4. Peças e Minutas Exigidas por Etapa
CREATE TABLE IF NOT EXISTS atlas_etapa_documentos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    etapa_id UUID NOT NULL REFERENCES atlas_etapas(id) ON DELETE CASCADE,
    nome_documento VARCHAR(150) NOT NULL,
    obrigatorio BOOLEAN NOT NULL DEFAULT TRUE,
    formato VARCHAR(32) NOT NULL CHECK (formato IN ('NATO_DIGITAL', 'EXTERNO_DIGITALIZADO')),
    tipo_assinatura VARCHAR(32) NOT NULL CHECK (tipo_assinatura IN ('INDIVIDUAL', 'CONJUNTA_MULTINIVEL', 'EM_BLOCO')),
    exige_conferencia_copia BOOLEAN NOT NULL DEFAULT FALSE,
    modelo_minuta_padrao_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_atlas_docs_etapa ON atlas_etapa_documentos(etapa_id);

-- 5. Regras de Transição e Trilhas de Diligência
CREATE TABLE IF NOT EXISTS atlas_etapa_transicoes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    origem_etapa_id UUID NOT NULL REFERENCES atlas_etapas(id) ON DELETE CASCADE,
    destino_etapa_id UUID NOT NULL REFERENCES atlas_etapas(id) ON DELETE CASCADE,
    condicao_transicao VARCHAR(255) NOT NULL,
    is_devolucao_diligencia BOOLEAN NOT NULL DEFAULT FALSE,
    descricao_diligencia TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
