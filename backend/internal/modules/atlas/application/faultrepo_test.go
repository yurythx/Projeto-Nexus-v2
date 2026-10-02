package application_test

// Código no formato de scripts/genfault.py a partir da interface Repository
// do domínio. Regenere se a interface mudar.

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nexus/internal/domain/pagination"
	"github.com/yurythx/projeto-nexus/internal/modules/atlas/domain"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

type innerRepo = domain.Repository

var errBoom = errors.New("falha simulada no repositório")

// faultRepo envolve o repositório real: na chamada failAt devolve erro; depois
// da chamada poisonAt "envenena" a transação (a instrução SQL seguinte — do
// repositório, do outbox ou da auditoria — falha). noTx marca veneno em
// leitura fora de transação, onde não há instrução seguinte na mesma conexão.
type faultRepo struct {
	inner                   innerRepo
	calls, failAt, poisonAt int
	trace                   []string
	noTx                    bool
}

func (fr *faultRepo) hook(name string) error {
	fr.calls++
	fr.trace = append(fr.trace, name)
	if fr.calls == fr.failAt {
		return errBoom
	}
	return nil
}

func (fr *faultRepo) post(ctx context.Context, db database.DBTX) {
	if fr.calls != fr.poisonAt {
		return
	}
	if _, inTx := db.(pgx.Tx); !inTx {
		fr.noTx = true
		return
	}
	_, _ = db.Exec(ctx, `SELECT 1/0`)
}

func (fr *faultRepo) ListTTDD(ctx context.Context, db database.DBTX, query string, p pagination.Params) ([]domain.ClassificacaoTTDD, int64, error) {
	if err := fr.hook("ListTTDD"); err != nil {
		var z0 []domain.ClassificacaoTTDD
		var z1 int64
		return z0, z1, err
	}
	defer fr.post(ctx, db)
	return fr.inner.ListTTDD(ctx, db, query, p)
}

func (fr *faultRepo) GetTTDD(ctx context.Context, db database.DBTX, codigo string) (domain.ClassificacaoTTDD, error) {
	if err := fr.hook("GetTTDD"); err != nil {
		var z0 domain.ClassificacaoTTDD
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.GetTTDD(ctx, db, codigo)
}

func (fr *faultRepo) LockTTDD(ctx context.Context, db database.DBTX, codigo string) (bool, error) {
	if err := fr.hook("LockTTDD"); err != nil {
		var z0 bool
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.LockTTDD(ctx, db, codigo)
}

func (fr *faultRepo) List(ctx context.Context, db database.DBTX, f domain.Filter, p pagination.Params) ([]domain.Workflow, int64, error) {
	if err := fr.hook("List"); err != nil {
		var z0 []domain.Workflow
		var z1 int64
		return z0, z1, err
	}
	defer fr.post(ctx, db)
	return fr.inner.List(ctx, db, f, p)
}

func (fr *faultRepo) Get(ctx context.Context, db database.DBTX, id uuid.UUID, forUpdate bool) (domain.Workflow, error) {
	if err := fr.hook("Get"); err != nil {
		var z0 domain.Workflow
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.Get(ctx, db, id, forUpdate)
}

func (fr *faultRepo) Insert(ctx context.Context, db database.DBTX, w domain.Workflow) error {
	if err := fr.hook("Insert"); err != nil {
		return err
	}
	defer fr.post(ctx, db)
	return fr.inner.Insert(ctx, db, w)
}

func (fr *faultRepo) SetAtivo(ctx context.Context, db database.DBTX, id uuid.UUID, ativo bool) error {
	if err := fr.hook("SetAtivo"); err != nil {
		return err
	}
	defer fr.post(ctx, db)
	return fr.inner.SetAtivo(ctx, db, id, ativo)
}

func (fr *faultRepo) Search(ctx context.Context, db database.DBTX, query string, limit int) ([]domain.Workflow, []float64, error) {
	if err := fr.hook("Search"); err != nil {
		var z0 []domain.Workflow
		var z1 []float64
		return z0, z1, err
	}
	defer fr.post(ctx, db)
	return fr.inner.Search(ctx, db, query, limit)
}

func (fr *faultRepo) Candidatos(ctx context.Context, db database.DBTX, pergunta string, limit int) ([]domain.Workflow, error) {
	if err := fr.hook("Candidatos"); err != nil {
		var z0 []domain.Workflow
		return z0, err
	}
	defer fr.post(ctx, db)
	return fr.inner.Candidatos(ctx, db, pergunta, limit)
}
