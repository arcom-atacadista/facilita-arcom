package db

import "embed"

// Migrations embarca os arquivos .sql no binário — o Dockerfile final
// (alpine, sem o código-fonte) não precisa copiar `.sql` separadamente, e o
// `cmd/migrate` sempre roda a partir do binário publicado. Ver
// pressly/goose "Embedding SQL migrations".
//
//go:embed migrations/*.sql
var Migrations embed.FS
