package transport

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nexus/internal/platform/auth"
	"github.com/yurythx/projeto-nexus/internal/platform/kernel"
)

// RegisterRoutes monta as rotas do módulo Atlas nos pontos de montagem
// públicos e autenticados providos pelo Kernel.
func RegisterRoutes(routes kernel.Routes, h *Handlers, logger *slog.Logger) {
	// Rotas Públicas (acesso institucional e consulta cidadã)
	if routes.Public != nil {
		routes.Public.Route("/atlas", func(r chi.Router) {
			r.Get("/ttdd", h.ListTTDD)
			r.Get("/ttdd/{codigo}", h.GetTTDD)
			r.Get("/workflows", h.ListWorkflows)
			r.Get("/workflows/{id}", h.GetWorkflow)
		})
	}

	// Rotas Autenticadas (servidores e gestores)
	if routes.Authed != nil {
		routes.Authed.Route("/atlas", func(r chi.Router) {
			r.Get("/ttdd", h.ListTTDD)
			r.Get("/ttdd/{codigo}", h.GetTTDD)
			r.Get("/workflows", h.ListWorkflows)
			r.Get("/workflows/{id}", h.GetWorkflow)
			
			// Consulta com IA (Grounding Estrito)
			r.Post("/chat", h.ChatProcedural)

			// Gestão de novos workflows (requer atlas:manage)
			r.With(auth.RequirePermission(logger, auth.PermAtlasManage)).Post("/workflows", h.CreateWorkflow)
		})
	}
}
