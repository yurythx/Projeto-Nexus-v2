package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

// Cobertura monta o painel em quatro consultas (órgãos, contagens, peças sem
// modelo, modelos sem uso).
func (r *Repository) Cobertura(ctx context.Context, db database.DBTX) (domain.Cobertura, error) {
	out := domain.Cobertura{Orgaos: []domain.CoberturaOrgao{}, PecasSemModelo: []domain.PecaSemModelo{}, ModelosSemUso: []domain.ModeloSemUso{}}
	err := coletar(ctx, db, `SELECT o.prefixo, o.nome, count(c.codigo),
		count(c.codigo) FILTER (WHERE EXISTS (SELECT 1 FROM atlas_workflows w WHERE w.ativo AND w.codigo_ttdd = c.codigo)),
		count(c.codigo) FILTER (WHERE EXISTS (SELECT 1 FROM atlas_ttdd_modelos tm JOIN atlas_modelos m ON m.id = tm.modelo_id AND m.ativo
			WHERE tm.codigo = c.codigo))
		FROM atlas_ttdd_orgaos o
		JOIN atlas_ttdd_funcoes f ON f.orgao_prefixo = o.prefixo
		JOIN atlas_ttdd_subfuncoes s ON s.funcao_codigo = f.codigo
		JOIN atlas_classificacao_ttdd c ON c.subfuncao_codigo = s.codigo AND c.revogada_em IS NULL
		GROUP BY o.prefixo, o.nome ORDER BY split_part(o.prefixo, '.', 1)::int`, func(rows pgx.Rows) error {
		var o domain.CoberturaOrgao
		err := rows.Scan(&o.Prefixo, &o.Nome, &o.Series, &o.SeriesComProcedimento, &o.SeriesComModelo)
		out.Orgaos = append(out.Orgaos, o)
		return err
	})
	if err != nil {
		return out, err
	}
	err = coletar(ctx, db, `SELECT w.id, w.codigo_processual, w.titulo, e.ordem, d.nome_documento
		FROM atlas_etapa_documentos d JOIN atlas_etapas e ON e.id = d.etapa_id JOIN atlas_workflows w ON w.id = e.workflow_id
		WHERE w.ativo AND NOT EXISTS (SELECT 1 FROM atlas_modelos m WHERE m.id = d.modelo_id AND m.ativo)
			AND NOT EXISTS (SELECT 1 FROM atlas_ttdd_modelos tm JOIN atlas_modelos m ON m.id = tm.modelo_id AND m.ativo
				WHERE tm.codigo = w.codigo_ttdd)
		ORDER BY w.codigo_processual, e.ordem, d.nome_documento`, func(rows pgx.Rows) error {
		var p domain.PecaSemModelo
		err := rows.Scan(&p.WorkflowID, &p.CodigoProcessual, &p.Titulo, &p.Etapa, &p.Peca)
		out.PecasSemModelo = append(out.PecasSemModelo, p)
		return err
	})
	if err != nil {
		return out, err
	}
	err = coletar(ctx, db, `SELECT m.id, m.nome FROM atlas_modelos m
		WHERE m.ativo
			AND NOT EXISTS (SELECT 1 FROM atlas_etapa_documentos d JOIN atlas_etapas e ON e.id = d.etapa_id
				JOIN atlas_workflows w ON w.id = e.workflow_id AND w.ativo WHERE d.modelo_id = m.id)
			AND NOT EXISTS (SELECT 1 FROM atlas_ttdd_modelos tm WHERE tm.modelo_id = m.id)
		ORDER BY lower(m.nome)`, func(rows pgx.Rows) error {
		var m domain.ModeloSemUso
		err := rows.Scan(&m.ID, &m.Nome)
		out.ModelosSemUso = append(out.ModelosSemUso, m)
		return err
	})
	if err != nil {
		return out, err
	}
	var procedimentos, pecas, modelos, rascunhos, emValidacao int
	if err := db.QueryRow(ctx, `SELECT (SELECT count(*) FROM atlas_workflows WHERE ativo),
		(SELECT count(*) FROM atlas_etapa_documentos d JOIN atlas_etapas e ON e.id = d.etapa_id
			JOIN atlas_workflows w ON w.id = e.workflow_id WHERE w.ativo),
		(SELECT count(*) FROM atlas_modelos WHERE ativo),
		(SELECT count(*) FROM atlas_workflows WHERE situacao = 'RASCUNHO'),
		(SELECT count(*) FROM atlas_workflows WHERE situacao = 'EM_VALIDACAO')`).Scan(&procedimentos, &pecas, &modelos, &rascunhos, &emValidacao); err != nil {
		return out, wrap(err)
	}
	out.Somar(procedimentos, pecas, modelos)
	out.Totais.Rascunhos, out.Totais.EmValidacao = rascunhos, emValidacao
	return out, nil
}

// coletar roda a consulta e passa cada linha a ler.
func coletar(ctx context.Context, db database.DBTX, sql string, ler func(pgx.Rows) error) error {
	rows, err := db.Query(ctx, sql)
	if err != nil {
		return wrap(err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := ler(rows); err != nil {
			return wrap(err)
		}
	}
	return wrap(rows.Err())
}
