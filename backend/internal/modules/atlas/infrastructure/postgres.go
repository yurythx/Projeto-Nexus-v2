// Package infrastructure implementa o repositório do Atlas.
package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nexus/internal/domain/pagination"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

// Repository implementa domain.Repository.
type Repository struct{}

// NewRepository cria o repositório.
func NewRepository() *Repository { return &Repository{} }

var _ domain.Repository = (*Repository)(nil)

func wrap(err error) error {
	switch {
	case err == nil:
		return nil
	case database.IsNoRows(err):
		return domain.ErrNotFound
	case database.IsUniqueViolation(err):
		return domain.ErrDuplicate
	}
	return fmt.Errorf("atlas: %w", err)
}

// ---------------------------------------------------------------- TTDD

const ttddCols = `codigo, descritor, fase_corrente_anos, fase_interm_anos, destinacao_final, COALESCE(observacoes, ''), created_at`

func scanTTDD(row pgx.Row) (domain.ClassificacaoTTDD, error) {
	var c domain.ClassificacaoTTDD
	err := row.Scan(&c.Codigo, &c.Descritor, &c.FaseCorrenteAnos, &c.FaseIntermAnos, &c.DestinacaoFinal, &c.Observacoes, &c.CreatedAt)
	return c, err
}

func (r *Repository) ListTTDD(ctx context.Context, db database.DBTX, query string, p pagination.Params) ([]domain.ClassificacaoTTDD, int64, error) {
	where, args := "", []any{}
	if query != "" {
		args = append(args, "%"+query+"%")
		where = ` WHERE nexus_unaccent(codigo || ' ' || descritor) ILIKE nexus_unaccent($1)`
	}
	var total int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM atlas_classificacao_ttdd`+where, args...).Scan(&total); err != nil {
		return nil, 0, wrap(err)
	}
	args = append(args, p.Limit(), p.Offset())
	rows, err := db.Query(ctx, fmt.Sprintf(`SELECT %s FROM atlas_classificacao_ttdd%s ORDER BY codigo LIMIT $%d OFFSET $%d`,
		ttddCols, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, wrap(err)
	}
	defer rows.Close()
	out := []domain.ClassificacaoTTDD{}
	for rows.Next() {
		c, err := scanTTDD(rows)
		if err != nil {
			return nil, 0, wrap(err)
		}
		out = append(out, c)
	}
	return out, total, wrap(rows.Err())
}

func (r *Repository) GetTTDD(ctx context.Context, db database.DBTX, codigo string) (domain.ClassificacaoTTDD, error) {
	c, err := scanTTDD(db.QueryRow(ctx, `SELECT `+ttddCols+` FROM atlas_classificacao_ttdd WHERE codigo = $1`, codigo))
	if database.IsNoRows(err) {
		return c, domain.ErrTTDDNotFound
	}
	return c, wrap(err)
}

func (r *Repository) LockTTDD(ctx context.Context, db database.DBTX, codigo string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM atlas_classificacao_ttdd WHERE codigo = $1 FOR SHARE)`, codigo).Scan(&ok)
	return ok, wrap(err)
}

// ----------------------------------------------------------- Workflows

const wfCols = `w.id, w.codigo_processual, w.titulo, w.objetivo, w.publico_alvo, w.versao, w.ativo, w.nivel_acesso,
	COALESCE(w.hipotese_legal_restricao, ''), w.codigo_ttdd, w.created_by, w.created_at, w.updated_at,
	c.descritor, c.fase_corrente_anos, c.fase_interm_anos, c.destinacao_final, COALESCE(c.observacoes, ''), c.created_at,
	(SELECT COUNT(*) FROM atlas_etapas e WHERE e.workflow_id = w.id)`

const wfFrom = ` FROM atlas_workflows w JOIN atlas_classificacao_ttdd c ON c.codigo = w.codigo_ttdd`

func scanWorkflow(row pgx.Row, extra ...any) (domain.Workflow, error) {
	var w domain.Workflow
	c := &domain.ClassificacaoTTDD{}
	dest := []any{&w.ID, &w.CodigoProcessual, &w.Titulo, &w.Objetivo, &w.PublicoAlvo, &w.Versao, &w.Ativo, &w.NivelAcesso,
		&w.HipoteseLegal, &w.CodigoTTDD, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
		&c.Descritor, &c.FaseCorrenteAnos, &c.FaseIntermAnos, &c.DestinacaoFinal, &c.Observacoes, &c.CreatedAt, &w.TotalEtapas}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return w, err
	}
	c.Codigo = w.CodigoTTDD
	w.Classificacao = c
	w.Etapas = []domain.Etapa{}
	return w, nil
}

func (r *Repository) List(ctx context.Context, db database.DBTX, f domain.Filter, p pagination.Params) ([]domain.Workflow, int64, error) {
	conds, args := []string{}, []any{}
	if !f.IncluirInativos {
		conds = append(conds, "w.ativo")
	}
	if f.CodigoTTDD != "" {
		args = append(args, f.CodigoTTDD)
		conds = append(conds, fmt.Sprintf("w.codigo_ttdd = $%d", len(args)))
	}
	if f.Query != "" {
		args = append(args, f.Query)
		conds = append(conds, fmt.Sprintf("(w.search @@ nexus_search_tsquery('portuguese', $%d) OR w.codigo_processual ILIKE '%%' || $%d || '%%')", len(args), len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*)`+wfFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, wrap(err)
	}
	args = append(args, p.Limit(), p.Offset())
	rows, err := db.Query(ctx, fmt.Sprintf(`SELECT %s%s%s ORDER BY w.codigo_processual, w.versao DESC LIMIT $%d OFFSET $%d`,
		wfCols, wfFrom, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, wrap(err)
	}
	defer rows.Close()
	out := []domain.Workflow{}
	for rows.Next() {
		w, err := scanWorkflow(rows)
		if err != nil {
			return nil, 0, wrap(err)
		}
		out = append(out, w)
	}
	return out, total, wrap(rows.Err())
}

func (r *Repository) Get(ctx context.Context, db database.DBTX, id uuid.UUID, forUpdate bool) (domain.Workflow, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE OF w"
	}
	w, err := scanWorkflow(db.QueryRow(ctx, `SELECT `+wfCols+wfFrom+` WHERE w.id = $1`+lock, id))
	if err != nil {
		return w, wrap(err)
	}
	full, err := r.loadEtapas(ctx, db, []domain.Workflow{w})
	if err != nil {
		return w, err
	}
	return full[0], nil
}

// loadEtapas carrega etapas, peças e transições de vários procedimentos em
// três consultas (sem N+1).
func (r *Repository) loadEtapas(ctx context.Context, db database.DBTX, wfs []domain.Workflow) ([]domain.Workflow, error) {
	if len(wfs) == 0 {
		return wfs, nil
	}
	ids := make([]uuid.UUID, len(wfs))
	idx := map[uuid.UUID]int{}
	for i, w := range wfs {
		ids[i] = w.ID
		idx[w.ID] = i
	}
	rows, err := db.Query(ctx, `SELECT id, workflow_id, ordem, unidade_administrativa, nome_setor, atribuicoes_setor,
		prazo_sla_em_dias, manter_aberto_apos_remessa FROM atlas_etapas WHERE workflow_id = ANY($1) ORDER BY workflow_id, ordem`, ids)
	if err != nil {
		return nil, wrap(err)
	}
	type pos struct{ wf, etapa int }
	etapaPos := map[uuid.UUID]pos{}
	ordemByEtapa := map[uuid.UUID]int{}
	var etapaIDs []uuid.UUID
	for rows.Next() {
		var e domain.Etapa
		var wfID uuid.UUID
		if err := rows.Scan(&e.ID, &wfID, &e.Ordem, &e.UnidadeAdministrativa, &e.NomeSetor, &e.AtribuicoesSetor,
			&e.PrazoSLAEmDias, &e.ManterAbertoAposRemessa); err != nil {
			rows.Close()
			return nil, wrap(err)
		}
		e.Documentos, e.Transicoes = []domain.EtapaDocumento{}, []domain.EtapaTransicao{}
		i := idx[wfID]
		wfs[i].Etapas = append(wfs[i].Etapas, e)
		etapaPos[e.ID] = pos{i, len(wfs[i].Etapas) - 1}
		ordemByEtapa[e.ID] = e.Ordem
		etapaIDs = append(etapaIDs, e.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, wrap(err)
	}
	if len(etapaIDs) == 0 {
		return wfs, nil
	}

	rows, err = db.Query(ctx, `SELECT id, etapa_id, nome_documento, obrigatorio, formato, tipo_assinatura, exige_conferencia_copia,
		COALESCE(modelo_minuta_padrao_url, '') FROM atlas_etapa_documentos WHERE etapa_id = ANY($1) ORDER BY nome_documento`, etapaIDs)
	if err != nil {
		return nil, wrap(err)
	}
	for rows.Next() {
		var d domain.EtapaDocumento
		var etapaID uuid.UUID
		if err := rows.Scan(&d.ID, &etapaID, &d.NomeDocumento, &d.Obrigatorio, &d.Formato, &d.TipoAssinatura,
			&d.ExigeConferenciaCopia, &d.ModeloMinutaPadraoURL); err != nil {
			rows.Close()
			return nil, wrap(err)
		}
		p := etapaPos[etapaID]
		wfs[p.wf].Etapas[p.etapa].Documentos = append(wfs[p.wf].Etapas[p.etapa].Documentos, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, wrap(err)
	}

	rows, err = db.Query(ctx, `SELECT id, origem_etapa_id, destino_etapa_id, condicao_transicao, is_devolucao_diligencia,
		COALESCE(descricao_diligencia, '') FROM atlas_etapa_transicoes WHERE origem_etapa_id = ANY($1) ORDER BY created_at, id`, etapaIDs)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()
	for rows.Next() {
		var t domain.EtapaTransicao
		var origem, destino uuid.UUID
		if err := rows.Scan(&t.ID, &origem, &destino, &t.CondicaoTransicao, &t.IsDevolucaoDiligencia, &t.DescricaoDiligencia); err != nil {
			return nil, wrap(err)
		}
		t.DestinoOrdem = ordemByEtapa[destino]
		p := etapaPos[origem]
		wfs[p.wf].Etapas[p.etapa].Transicoes = append(wfs[p.wf].Etapas[p.etapa].Transicoes, t)
	}
	return wfs, wrap(rows.Err())
}

func (r *Repository) Insert(ctx context.Context, db database.DBTX, w domain.Workflow) error {
	if _, err := db.Exec(ctx, `INSERT INTO atlas_workflows (id, codigo_processual, titulo, objetivo, publico_alvo, versao, ativo,
		nivel_acesso, hipotese_legal_restricao, codigo_ttdd, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10,$11)`,
		w.ID, w.CodigoProcessual, w.Titulo, w.Objetivo, w.PublicoAlvo, w.Versao, w.Ativo, string(w.NivelAcesso), w.HipoteseLegal,
		w.CodigoTTDD, w.CreatedBy); err != nil {
		return wrap(err)
	}
	etapaID := map[int]uuid.UUID{}
	for _, e := range w.Etapas {
		etapaID[e.Ordem] = e.ID
		if _, err := db.Exec(ctx, `INSERT INTO atlas_etapas (id, workflow_id, ordem, unidade_administrativa, nome_setor,
			atribuicoes_setor, prazo_sla_em_dias, manter_aberto_apos_remessa) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			e.ID, w.ID, e.Ordem, e.UnidadeAdministrativa, e.NomeSetor, e.AtribuicoesSetor, e.PrazoSLAEmDias, e.ManterAbertoAposRemessa); err != nil {
			return wrap(err)
		}
		for _, d := range e.Documentos {
			if _, err := db.Exec(ctx, `INSERT INTO atlas_etapa_documentos (id, etapa_id, nome_documento, obrigatorio, formato,
				tipo_assinatura, exige_conferencia_copia, modelo_minuta_padrao_url) VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''))`,
				d.ID, e.ID, d.NomeDocumento, d.Obrigatorio, string(d.Formato), string(d.TipoAssinatura), d.ExigeConferenciaCopia, d.ModeloMinutaPadraoURL); err != nil {
				return wrap(err)
			}
		}
	}
	// Transições depois de todas as etapas: o destino pode vir adiante.
	for _, e := range w.Etapas {
		for _, t := range e.Transicoes {
			if _, err := db.Exec(ctx, `INSERT INTO atlas_etapa_transicoes (id, origem_etapa_id, destino_etapa_id, condicao_transicao,
				is_devolucao_diligencia, descricao_diligencia) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''))`,
				t.ID, e.ID, etapaID[t.DestinoOrdem], t.CondicaoTransicao, t.IsDevolucaoDiligencia, t.DescricaoDiligencia); err != nil {
				return wrap(err)
			}
		}
	}
	return nil
}

func (r *Repository) SetAtivo(ctx context.Context, db database.DBTX, id uuid.UUID, ativo bool) error {
	tag, err := db.Exec(ctx, `UPDATE atlas_workflows SET ativo = $2 WHERE id = $1`, id, ativo)
	if err != nil {
		return wrap(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) Search(ctx context.Context, db database.DBTX, query string, limit int) ([]domain.Workflow, []float64, error) {
	rows, err := db.Query(ctx, `SELECT `+wfCols+`, ts_rank(w.search, q)`+wfFrom+`, nexus_search_tsquery('portuguese', $1) q
		WHERE w.ativo AND w.search @@ q ORDER BY 21 DESC LIMIT $2`, query, limit)
	if err != nil {
		return nil, nil, wrap(err)
	}
	defer rows.Close()
	var out []domain.Workflow
	var ranks []float64
	for rows.Next() {
		var rank float32
		w, err := scanWorkflow(rows, &rank)
		if err != nil {
			return nil, nil, wrap(err)
		}
		out = append(out, w)
		ranks = append(ranks, float64(rank))
	}
	return out, ranks, wrap(rows.Err())
}

// Candidatos: basta UM termo da pergunta casar (OR) — em linguagem natural
// a pergunta tem palavras que o procedimento não usa. A relevância real é
// medida depois, no domínio (domain.Relevancia), sobre o procedimento
// completo; aqui só se recorta o universo pelo índice.
func (r *Repository) Candidatos(ctx context.Context, db database.DBTX, pergunta string, limit int) ([]domain.Workflow, error) {
	rows, err := db.Query(ctx, `WITH q AS (
			SELECT NULLIF(replace(plainto_tsquery('portuguese', nexus_unaccent($1))::text, ' & ', ' | '), '')::tsquery AS q
		), hits AS (
			SELECT w.id, ts_rank(w.search, q.q) AS rank FROM atlas_workflows w, q WHERE w.ativo AND w.search @@ q.q
			UNION ALL
			SELECT e.workflow_id, ts_rank(e.search, q.q) * 0.5 FROM atlas_etapas e, q WHERE e.search @@ q.q
			UNION ALL
			SELECT e.workflow_id, ts_rank(d.search, q.q) * 0.5 FROM atlas_etapa_documentos d
				JOIN atlas_etapas e ON e.id = d.etapa_id, q WHERE d.search @@ q.q
		)
		SELECT `+wfCols+wfFrom+` JOIN (SELECT id, SUM(rank) AS rank FROM hits GROUP BY id) h ON h.id = w.id
		WHERE w.ativo ORDER BY h.rank DESC, w.codigo_processual LIMIT $2`, pergunta, limit)
	if err != nil {
		return nil, wrap(err)
	}
	var out []domain.Workflow
	for rows.Next() {
		w, err := scanWorkflow(rows)
		if err != nil {
			rows.Close()
			return nil, wrap(err)
		}
		out = append(out, w)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, wrap(err)
	}
	return r.loadEtapas(ctx, db, out)
}
