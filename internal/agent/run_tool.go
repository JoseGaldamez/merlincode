package agent

import (
	"context"
	"encoding/json"

	"merlincode/internal/domain"
)

// proposeRunTool prepares cards without waiting for approval or launching anything.
// Unlike file approval, a proposal remains usable after the agent turn has ended.
type proposeRunTool struct {
	prepare func(domain.RunRequest) ([]domain.RunProposal, error)
}

// NewRunTool prepares proposals only. Launching still requires the user's Play action.
func NewRunTool(prepare func(domain.RunRequest) ([]domain.RunProposal, error)) ToolExecutor {
	return proposeRunTool{prepare: prepare}
}

func (proposeRunTool) Name() string           { return "propose_app_run" }
func (proposeRunTool) RequiresApproval() bool { return false }
func (proposeRunTool) Definition() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name: "propose_app_run",
		Description: "Merlin Code SÍ puede ejecutar aplicaciones, scripts Python y servidores locales mediante el botón Play. Cuando el usuario solicita ejecutar un archivo o iniciar una aplicación o servidor, usa esta herramienta. " +
			"El análisis es general para cualquier proyecto: primero usa get_tree o list_files para localizar la aplicación y sus archivos de configuración. Si hay package.json, léelo con read_file: analiza scripts, packageManager y dependencias para identificar sus comandos reales, sea React, Angular, Vue, Astro, Next, Nuxt u otro framework. Identifica npm/pnpm/yarn/bun a partir de packageManager y los nombres de los lockfiles (package-lock.json, npm-shrinkwrap.json, pnpm-lock.yaml, yarn.lock, bun.lock o bun.lockb), también en las carpetas superiores del proyecto si es un monorepo. No hace falta leer lockfiles completos. Consulta README y la configuración del framework si hace falta aclarar cómo se inicia (por ejemplo angular.json, vite.config.*, vue.config.* o astro.config.*). " +
			"Para aplicaciones JavaScript/TypeScript pasa directory con la carpeta que contiene package.json y omite entry: ni el manifiesto ni archivos de componentes/páginas son entradas ejecutables del framework. La herramienta vuelve a leer y validar los scripts y el gestor; utiliza el comando que devuelva en lugar de suponer npm o inventar scripts. Si un lanzador no tiene un perfil admitido, explica esa limitación concreta; no modifiques el proyecto para eludirla. " +
			"Los scripts Python de ejecución corta (por ejemplo hola_mundo.py) TAMBIÉN son compatibles: pasa su ruta relativa en entry; no tienen que ser servidores, usar un framework ni declarar un bloque __main__. No sugieras convertirlos en servidores. Analiza pyproject.toml, requirements.txt y las entradas Python según corresponda. También detecta gestores Python (uv, poetry, pipenv), Go (incluidos archivos independientes sin go.mod) y HTML estático. " +
			"Devuelve propuestas con el comando y botón Play; el proceso solo inicia al pulsarlo. Si devuelve propuestas, indica al usuario que pulse Play; no digas que no puedes ejecutarlas ni lo envíes a otra terminal. Si devuelve un error, explica su causa concreta; si falta información del gestor o hay señales ambiguas, solicita aclaración sin modificar los archivos del proyecto. No afirmes que ya se ejecutó. Si hay varias apps, prepara cada carpeta por separado. Nunca uses write_file para crear un script o modificar la configuración con el fin de sortear las restricciones de ejecución.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"directory": map[string]any{"type": "string", "description": "Carpeta relativa al proyecto activo; usa . para su raíz. En proyectos JavaScript/TypeScript/Astro debe contener el package.json analizado."},
			"entry":     map[string]any{"type": "string", "description": "Solo para un archivo de entrada real Python/Go, relativo a directory. Omite este campo para package.json, Astro y demás frameworks JavaScript/TypeScript. Los .py pueden ser scripts de ejecución corta y no necesitan ser servidores. Debe existir; no acepta comandos ni argumentos."},
		}, "required": []string{"directory"}, "additionalProperties": false},
	}
}
func (t proposeRunTool) Execute(ctx context.Context, args string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var request domain.RunRequest
	if err := json.Unmarshal([]byte(args), &request); err != nil {
		return "", &domain.RunPolicyError{Message: "Los argumentos de ejecución no son válidos."}
	}
	proposals, err := t.prepare(request)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(proposals)
	return string(data), err
}
