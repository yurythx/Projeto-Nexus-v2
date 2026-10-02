package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository {
	return &PostgresRepository{}
}

// -----------------------------------------------------------------------------
// Classificação TTDD
// -----------------------------------------------------------------------------

func (r *PostgresRepository) ListClassificacoes(ctx context.Context, dbtx database.DBTX, query string, limit, offset int) ([]domain.ClassificacaoTTDD, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	whereClause := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if strings.TrimSpace(query) != "" {
		whereClause += fmt.Sprintf(" AND (codigo ILIKE $%d OR descritor ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+query+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM atlas_classificacao_ttdd %s", whereClause)
	var total int
	if err := dbtx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contagem de classificações ttdd: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT codigo, descritor, fase_corrente_anos, fase_interm_anos, destinacao_final, observacoes, created_at
		FROM atlas_classificacao_ttdd
		%s
		ORDER BY codigo ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := dbtx.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("consulta de classificações ttdd: %w", err)
	}
	defer rows.Close()

	var list []domain.ClassificacaoTTDD
	for rows.Next() {
		var c domain.ClassificacaoTTDD
		var obs sql.NullString
		if err := rows.Scan(&c.Codigo, &c.Descritor, &c.FaseCorrenteAnos, &c.FaseIntermAnos, &c.DestinacaoFinal, &obs, &c.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan classificação ttdd: %w", err)
		}
		if obs.Valid {
			c.Observacoes = obs.String
		}
		list = append(list, c)
	}

	return list, total, nil
}

func (r *PostgresRepository) GetClassificacaoByCodigo(ctx context.Context, dbtx database.DBTX, codigo string) (*domain.ClassificacaoTTDD, error) {
	query := `
		SELECT codigo, descritor, fase_corrente_anos, fase_interm_anos, destinacao_final, observacoes, created_at
		FROM atlas_classificacao_ttdd
		WHERE codigo = $1
	`
	var c domain.ClassificacaoTTDD
	var obs sql.NullString
	err := dbtx.QueryRow(ctx, query, codigo).Scan(&c.Codigo, &c.Descritor, &c.FaseCorrenteAnos, &c.FaseIntermAnos, &c.DestinacaoFinal, &obs, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrClassificacaoNotFound
		}
		return nil, fmt.Errorf("obter classificação ttdd %q: %w", codigo, err)
	}
	if obs.Valid {
		c.Observacoes = obs.String
	}
	return &c, nil
}

func (r *PostgresRepository) SaveClassificacao(ctx context.Context, dbtx database.DBTX, c *domain.ClassificacaoTTDD) error {
	query := `
		INSERT INTO atlas_classificacao_ttdd (codigo, descritor, fase_corrente_anos, fase_interm_anos, destinacao_final, observacoes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (codigo) DO UPDATE SET
			descritor = EXCLUDED.descritor,
			fase_corrente_anos = EXCLUDED.fase_corrente_anos,
			fase_interm_anos = EXCLUDED.fase_interm_anos,
			destinacao_final = EXCLUDED.destinacao_final,
			observacoes = EXCLUDED.observacoes
	`
	_, err := dbtx.Exec(ctx, query, c.Codigo, c.Descritor, c.FaseCorrenteAnos, c.FaseIntermAnos, string(c.DestinacaoFinal), c.Observacoes)
	if err != nil {
		return fmt.Errorf("salvar classificação ttdd: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Workflows SEI
// -----------------------------------------------------------------------------

func (r *PostgresRepository) ListWorkflows(ctx context.Context, dbtx database.DBTX, tenantID string, query string, codigoTTDD string, limit, offset int) ([]domain.Workflow, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	whereClause := "WHERE w.ativo = TRUE"
	args := []any{}
	argIdx := 1

	if strings.TrimSpace(tenantID) != "" {
		whereClause += fmt.Sprintf(" AND w.tenant_id = $%d", argIdx)
		args = append(args, tenantID)
		argIdx++
	}

	if strings.TrimSpace(codigoTTDD) != "" {
		whereClause += fmt.Sprintf(" AND w.codigo_ttdd = $%d", argIdx)
		args = append(args, codigoTTDD)
		argIdx++
	}

	if strings.TrimSpace(query) != "" {
		whereClause += fmt.Sprintf(" AND (w.titulo ILIKE $%d OR w.codigo_processual ILIKE $%d OR w.objetivo ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+query+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM atlas_workflows w %s", whereClause)
	var total int
	if err := dbtx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contagem de workflows: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			w.id, w.tenant_id, w.codigo_processual, w.titulo, w.objetivo, w.publico_alvo,
			w.versao, w.ativo, w.nivel_acesso, w.hipotese_legal_restricao, w.codigo_ttdd,
			w.created_at, w.updated_at,
			c.descritor, c.fase_corrente_anos, c.fase_interm_anos, c.destinacao_final, c.observacoes
		FROM atlas_workflows w
		JOIN atlas_classificacao_ttdd c ON c.codigo = w.codigo_ttdd
		%s
		ORDER BY w.codigo_processual ASC, w.versao DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := dbtx.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("consulta de workflows: %w", err)
	}
	defer rows.Close()

	var list []domain.Workflow
	for rows.Next() {
		var w domain.Workflow
		var c domain.ClassificacaoTTDD
		var hipLegal, ttddObs sql.NullString

		err := rows.Scan(
			&w.ID, &w.TenantID, &w.CodigoProcessual, &w.Titulo, &w.Objetivo, &w.PublicoAlvo,
			&w.Versao, &w.Ativo, &w.NivelAcesso, &hipLegal, &w.CodigoTTDD,
			&w.CreatedAt, &w.UpdatedAt,
			&c.Descritor, &c.FaseCorrenteAnos, &c.FaseIntermAnos, &c.DestinacaoFinal, &ttddObs,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan workflow: %w", err)
		}
		if hipLegal.Valid {
			w.HipoteseLegal = hipLegal.String
		}
		c.Codigo = w.CodigoTTDD
		if ttddObs.Valid {
			c.Observacoes = ttddObs.String
		}
		w.Classificacao = &c
		list = append(list, w)
	}

	return list, total, nil
}

func (r *PostgresRepository) GetWorkflowByID(ctx context.Context, dbtx database.DBTX, id uuid.UUID) (*domain.Workflow, error) {
	query := `
		SELECT 
			w.id, w.tenant_id, w.codigo_processual, w.titulo, w.objetivo, w.publico_alvo,
			w.versao, w.ativo, w.nivel_acesso, w.hipotese_legal_restricao, w.codigo_ttdd,
			w.created_at, w.updated_at,
			c.descritor, c.fase_corrente_anos, c.fase_interm_anos, c.destinacao_final, c.observacoes
		FROM atlas_workflows w
		JOIN atlas_classificacao_ttdd c ON c.codigo = w.codigo_ttdd
		WHERE w.id = $1
	`
	var w domain.Workflow
	var c domain.ClassificacaoTTDD
	var hipLegal, ttddObs sql.NullString

	err := dbtx.QueryRow(ctx, query, id).Scan(
		&w.ID, &w.TenantID, &w.CodigoProcessual, &w.Titulo, &w.Objetivo, &w.PublicoAlvo,
		&w.Versao, &w.Ativo, &w.NivelAcesso, &hipLegal, &w.CodigoTTDD,
		&w.CreatedAt, &w.UpdatedAt,
		&c.Descritor, &c.FaseCorrenteAnos, &c.FaseIntermAnos, &c.DestinacaoFinal, &ttddObs,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("obter workflow %s: %w", id, err)
	}
	if hipLegal.Valid {
		w.HipoteseLegal = hipLegal.String
	}
	c.Codigo = w.CodigoTTDD
	if ttddObs.Valid {
		c.Observacoes = ttddObs.String
	}
	w.Classificacao = &c

	// Carrega etapas
	etapas, err := r.loadEtapas(ctx, dbtx, w.ID)
	if err != nil {
		return nil, err
	}
	w.Etapas = etapas

	return &w, nil
}

func (r *PostgresRepository) GetWorkflowByCodigo(ctx context.Context, dbtx database.DBTX, tenantID, codigoProcessual string, versao int) (*domain.Workflow, error) {
	query := `
		SELECT id FROM atlas_workflows
		WHERE tenant_id = $1 AND codigo_processual = $2 AND versao = $3
	`
	var id uuid.UUID
	err := dbtx.QueryRow(ctx, query, tenantID, codigoProcessual, versao).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("obter workflow por código: %w", err)
	}
	return r.GetWorkflowByID(ctx, dbtx, id)
}

func (r *PostgresRepository) loadEtapas(ctx context.Context, dbtx database.DBTX, workflowID uuid.UUID) ([]domain.Etapa, error) {
	etapasQuery := `
		SELECT id, workflow_id, ordem, unidade_administrativa, nome_setor, atribuicoes_setor,
		       prazo_sla_em_dias, manter_aberto_apos_remessa, created_at
		FROM atlas_etapas
		WHERE workflow_id = $1
		ORDER BY ordem ASC
	`
	rows, err := dbtx.Query(ctx, etapasQuery, workflowID)
	if err != nil {
		return nil, fmt.Errorf("carregar etapas do workflow %s: %w", workflowID, err)
	}
	defer rows.Close()

	var etapas []domain.Etapa
	var etapaIDs []uuid.UUID
	for rows.Next() {
		var e domain.Etapa
		if err := rows.Scan(
			&e.ID, &e.WorkflowID, &e.Ordem, &e.UnidadeAdministrativa, &e.NomeSetor,
			&e.AtribuicoesSetor, &e.PrazoSLAEmDias, &e.ManterAbertoAposRemessa, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan etapa: %w", err)
		}
		etapas = append(etapas, e)
		etapaIDs = append(etapaIDs, e.ID)
	}

	if len(etapas) == 0 {
		return etapas, nil
	}

	// Carrega documentos das etapas
	docsQuery := `
		SELECT id, etapa_id, nome_documento, obrigatorio, formato, tipo_assinatura,
		       exige_conferencia_copia, modelo_minuta_padrao_url, created_at
		FROM atlas_etapa_documentos
		WHERE etapa_id = ANY($1)
		ORDER BY nome_documento ASC
	`
	docRows, err := dbtx.Query(ctx, docsQuery, etapaIDs)
	if err != nil {
		return nil, fmt.Errorf("carregar documentos das etapas: %w", err)
	}
	defer docRows.Close()

	docsByEtapa := make(map[uuid.UUID][]domain.EtapaDocumento)
	for docRows.Next() {
		var d domain.EtapaDocumento
		var modeloURL sql.NullString
		if err := docRows.Scan(
			&d.ID, &d.EtapaID, &d.NomeDocumento, &d.Obrigatorio, &d.Formato, &d.TipoAssinatura,
			&d.ExigeConferenciaCopia, &modeloURL, &d.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan documento: %w", err)
		}
		if modeloURL.Valid {
			d.ModeloMinutaPadraoURL = modeloURL.String
		}
		docsByEtapa[d.EtapaID] = append(docsByEtapa[d.EtapaID], d)
	}

	// Carrega transições
	transQuery := `
		SELECT id, origem_etapa_id, destino_etapa_id, condicao_transicao,
		       is_devolucao_diligencia, descricao_diligencia, created_at
		FROM atlas_etapa_transicoes
		WHERE origem_etapa_id = ANY($1)
	`
	transRows, err := dbtx.Query(ctx, transQuery, etapaIDs)
	if err != nil {
		return nil, fmt.Errorf("carregar transições das etapas: %w", err)
	}
	defer transRows.Close()

	transByOrigem := make(map[uuid.UUID][]domain.EtapaTransicao)
	for transRows.Next() {
		var t domain.EtapaTransicao
		var descDiligencia sql.NullString
		if err := transRows.Scan(
			&t.ID, &t.OrigemEtapaID, &t.DestinoEtapaID, &t.CondicaoTransicao,
			&t.IsDevolucaoDiligencia, &descDiligencia, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transicao: %w", err)
		}
		if descDiligencia.Valid {
			t.DescricaoDiligencia = descDiligencia.String
		}
		transByOrigem[t.OrigemEtapaID] = append(transByOrigem[t.OrigemEtapaID], t)
	}

	for i := range etapas {
		etapas[i].Documentos = docsByEtapa[etapas[i].ID]
		etapas[i].Transicoes = transByOrigem[etapas[i].ID]
	}

	return etapas, nil
}

func (r *PostgresRepository) SaveWorkflow(ctx context.Context, dbtx database.DBTX, w *domain.Workflow) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}

	wfQuery := `
		INSERT INTO atlas_workflows (
			id, tenant_id, codigo_processual, titulo, objetivo, publico_alvo,
			versao, ativo, nivel_acesso, hipotese_legal_restricao, codigo_ttdd,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		ON CONFLICT (tenant_id, codigo_processual, versao) DO UPDATE SET
			titulo = EXCLUDED.titulo,
			objetivo = EXCLUDED.objetivo,
			publico_alvo = EXCLUDED.publico_alvo,
			ativo = EXCLUDED.ativo,
			nivel_acesso = EXCLUDED.nivel_acesso,
			hipotese_legal_restricao = EXCLUDED.hipotese_legal_restricao,
			codigo_ttdd = EXCLUDED.codigo_ttdd,
			updated_at = NOW()
	`
	_, err := dbtx.Exec(ctx, wfQuery,
		w.ID, w.TenantID, w.CodigoProcessual, w.Titulo, w.Objetivo, w.PublicoAlvo,
		w.Versao, w.Ativo, string(w.NivelAcesso), w.HipoteseLegal, w.CodigoTTDD,
	)
	if err != nil {
		return fmt.Errorf("salvar cabeçalho do workflow: %w", err)
	}

	// Remove etapas anteriores para regravação atômica
	_, _ = dbtx.Exec(ctx, "DELETE FROM atlas_etapas WHERE workflow_id = $1", w.ID)

	for _, e := range w.Etapas {
		etapaID := e.ID
		if etapaID == uuid.Nil {
			etapaID = uuid.New()
		}
		etapaQuery := `
			INSERT INTO atlas_etapas (
				id, workflow_id, ordem, unidade_administrativa, nome_setor,
				atribuicoes_setor, prazo_sla_em_dias, manter_aberto_apos_remessa, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`
		_, err := dbtx.Exec(ctx, etapaQuery,
			etapaID, w.ID, e.Ordem, e.UnidadeAdministrativa, e.NomeSetor,
			e.AtribuicoesSetor, e.PrazoSLAEmDias, e.ManterAbertoAposRemessa,
		)
		if err != nil {
			return fmt.Errorf("inserir etapa ordem %d: %w", e.Ordem, err)
		}

		for _, d := range e.Documentos {
			docID := d.ID
			if docID == uuid.Nil {
				docID = uuid.New()
			}
			docQuery := `
				INSERT INTO atlas_etapa_documentos (
					id, etapa_id, nome_documento, obrigatorio, formato,
					tipo_assinatura, exige_conferencia_copia, modelo_minuta_padrao_url, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
			`
			_, err := dbtx.Exec(ctx, docQuery,
				docID, etapaID, d.NomeDocumento, d.Obrigatorio, string(d.Formato),
				string(d.TipoAssinatura), d.ExigeConferenciaCopia, d.ModeloMinutaPadraoURL,
			)
			if err != nil {
				return fmt.Errorf("inserir documento de etapa %s: %w", d.NomeDocumento, err)
			}
		}

		for _, t := range e.Transicoes {
			transID := t.ID
			if transID == uuid.Nil {
				transID = uuid.New()
			}
			transQuery := `
				INSERT INTO atlas_etapa_transicoes (
					id, origem_etapa_id, destino_etapa_id, condicao_transicao,
					is_devolucao_diligencia, descricao_diligencia, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, NOW())
			`
			_, err := dbtx.Exec(ctx, transQuery,
				transID, etapaID, t.DestinoEtapaID, t.CondicaoTransicao,
				t.IsDevolucaoDiligencia, t.DescricaoDiligencia,
			)
			if err != nil {
				return fmt.Errorf("inserir transição de etapa: %w", err)
			}
		}
	}

	return nil
}

func (r *PostgresRepository) SearchProceduralGrounding(ctx context.Context, dbtx database.DBTX, tenantID string, query string, limit int) ([]domain.Workflow, error) {
	if limit <= 0 {
		limit = 5
	}
	searchPattern := "%" + query + "%"
	q := `
		SELECT DISTINCT w.id
		FROM atlas_workflows w
		LEFT JOIN atlas_etapas e ON e.workflow_id = w.id
		LEFT JOIN atlas_etapa_documentos d ON d.etapa_id = e.id
		LEFT JOIN atlas_classificacao_ttdd c ON c.codigo = w.codigo_ttdd
		WHERE w.ativo = TRUE 
		  AND (w.tenant_id = $1 OR $1 = '')
		  AND (
		      w.titulo ILIKE $2 OR
		      w.objetivo ILIKE $2 OR
		      w.codigo_processual ILIKE $2 OR
		      c.descritor ILIKE $2 OR
		      c.codigo ILIKE $2 OR
		      e.nome_setor ILIKE $2 OR
		      e.atribuicoes_setor ILIKE $2 OR
		      d.nome_documento ILIKE $2
		  )
		LIMIT $3
	`
	rows, err := dbtx.Query(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("busca procedural grounding: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}

	var results []domain.Workflow
	for _, id := range ids {
		wf, err := r.GetWorkflowByID(ctx, dbtx, id)
		if err == nil && wf != nil {
			results = append(results, *wf)
		}
	}

	return results, nil
}
