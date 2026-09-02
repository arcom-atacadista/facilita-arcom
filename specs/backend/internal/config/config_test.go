package config_test

import (
	"testing"

	"meu-projeto/internal/config"
)

func TestLoadFalhaComJWTSecretCurto(t *testing.T) {
	t.Setenv("JWT_SECRET", "curto-demais")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load deveria falhar com JWT_SECRET < 32 caracteres, mas não falhou")
	}
}

func TestLoadUsaDefaultDePorta(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load falhou sem nenhuma env obrigatória faltando: %v", err)
	}
	if cfg.Port != "3000" {
		t.Errorf("Port = %q, quer default 3000", cfg.Port)
	}
}
