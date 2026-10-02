package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-nexus/internal/platform/database"
)

var (
	ErrWorkflowNotFound       = errors.New("workflow não encontrado")
	ErrClassificacaoNotFound  = errors.New("classificação TTDD não encontrada")
	ErrEtapaNotFound          = errors.New("etapa não encontrada")
	ErrInvalidWorkflowVersion = errors.New("versão do workflow inválida ou já existente")
	ErrInvalidInput           = errors.New("dados de entrada inválidos")
)

type Repository interface {
	// Classificação TTDD
	ListClassificacoes(ctx context.Context, dbtx database.DBTX, query string, limit, offset int) ([]ClassificacaoTTDD, int, error)
	GetClassificacaoByCodigo(ctx context.Context, dbtx database.DBTX, codigo string) (*ClassificacaoTTDD, error)
	SaveClassificacao(ctx context.Context, dbtx database.DBTX, c *ClassificacaoTTDD) error

	// Workflows
	ListWorkflows(ctx context.Context, dbtx database.DBTX, tenantID string, query string, codigoTTDD string, limit, offset int) ([]Workflow, int, error)
	GetWorkflowByID(ctx context.Context, dbtx database.DBTX, id uuid.UUID) (*Workflow, error)
	GetWorkflowByCodigo(ctx context.Context, dbtx database.DBTX, tenantID, codigoProcessual string, versao int) (*Workflow, error)
	SaveWorkflow(ctx context.Context, dbtx database.DBTX, w *Workflow) error

	// Busca textual de procedimentos para alimentar busca semântica/híbrida
	SearchProceduralGrounding(ctx context.Context, dbtx database.DBTX, tenantID string, query string, limit int) ([]Workflow, error)
}
