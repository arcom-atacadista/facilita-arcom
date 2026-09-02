package servidor

import (
	"encoding/json"
	"net/http"

	"meu-projeto/internal/db"
)

// saude é o liveness — GET /api/health. Só responde 200 se o processo está
// de pé. NUNCA toca banco/rede: um Postgres lento não pode derrubar o
// container por causa disto (evita restart em cascata).
func (s *Servidor) saude(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// pronto é o readiness — GET /api/ready. Verifica se as dependências
// configuradas (Postgres/Redis) respondem; 503 se não. É este endpoint que o
// healthcheck do docker-compose e um load balancer devem consultar antes de
// mandar tráfego real.
func (s *Servidor) pronto(w http.ResponseWriter, r *http.Request) error {
	if err := db.Pronto(r.Context(), s.db, s.redis); err != nil {
		return &ErroDominio{
			Status: http.StatusServiceUnavailable,
			Titulo: "Indisponível",
			Detail: "Uma dependência (banco de dados ou cache) não está respondendo.",
			Codigo: "indisponivel",
		}
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
