# Merlin Code: guía para agentes

## Propósito y estado del producto

Merlin Code es una aplicación de escritorio local-first para asistir en tareas de
programación dentro de una carpeta elegida explícitamente por la persona usuaria.
Está construida con Go + Wails v2 en el backend y React 18 + TypeScript + Vite +
Tailwind en el frontend.

La interfaz, la gestión de proyectos, el sandbox de archivos, **la integración base
de proveedores de IA con almacenamiento seguro en llavero nativo, el streaming en tiempo
real en el chat y la persistencia local de sesiones y mensajes en SQLite están completamente implementados**.
El chat realiza llamadas HTTP SSE reales a través de Go hacia los proveedores configurados
(Google Gemini, OpenAI, Anthropic y DeepSeek), transmitiendo tokens, cadena de pensamiento
(reasoning/thinking) y estados de conexión en vivo a la interfaz sin bloquear el WebView.
El historial de conversaciones, métricas de tokens y calificaciones de usuario (like/dislike)
se almacenan en SQLite local sin dependencias de CGO.
El siguiente paso en la evolución es conectar el bucle autónomo del agente y las herramientas
del workspace.

## Arquitectura y flujo de datos

```text
React (frontend/src) -- Wails bindings --> internal/app (fachada modular)
                                              |
      +------------------------------+--------+---------------------+------------------------------+
      |                              |                              |                              |
internal/workspace.Service   internal/window.Service         internal/ai.Service            internal/session.Service
proyecto y sistema archivos  estado de ventana y paneles     proveedores IA y llavero       SQLite local (merlin.db)
                                                                    |                       sesiones, mensajes y feedback
                                              +---------------------+---------------------+
                                              |                                           |
                                   internal/ai/catalog.go                      internal/ai/transport/
                                   modelos declarativos (JSON)                 HTTP seguro y sanitización
                                                                                          |
                                                                               internal/platform/keyring/
                                                                               llavero nativo (Credential Manager, etc.)
```

- `main.go`: único archivo Go en la raíz del proyecto; arranque de Wails, ventana
  sin bordes y assets embebidos desde `frontend/dist`.
- `internal/app`: fachada pública de Go para Wails dividida en módulos cohesivos
  (`app.go`, `window.go`, `workspace.go`, `ai.go`, `chat.go`, `session.go`). Permanece delgada:
  delega a los servicios correspondientes (`workspace`, `window`, `ai`, `session`) y no contiene
  reglas de negocio.
- `internal/domain`: entidades y errores sin dependencias de infraestructura.
- `internal/workspace`: proyecto activo, árbol, lectura/escritura y explorador
  del sistema.
- `internal/window`: persistencia de tamaño, maximizado y paneles en el directorio
  de configuración del usuario (`merlincode/window.json`).
- `internal/session`: persistencia local-first de sesiones y mensajes en SQLite puro
  utilizando `modernc.org/sqlite` (sin CGO) en `%APPDATA%/merlincode/merlin.db`.
  Gestiona claves foráneas, borrado en cascada, WAL mode y feedback de mensajes.
- `internal/ai`: servicio orquestador de proveedores de IA, llavero seguro del SO
  y catálogo de modelos.
  - `internal/ai/configs/`: archivos JSON embebidos (`gemini.json`, `openai.json`,
    `anthropic.json`, `deepseek.json`) que definen de forma declarativa el modelo
    orquestador y los modelos disponibles de cada proveedor sin quemarlos en Go.
  - `internal/ai/catalog.go`: carga y consulta los catálogos JSON en tiempo de ejecución.
  - `internal/ai/transport/`: cliente HTTP hardened con límite de payloads (2 MiB),
    bloqueo de redirecciones externas y sanitización estricta de errores para no
    filtrar URLs ni tokens.
  - `internal/ai/providers/`: adaptadores independientes para Google, OpenAI, Anthropic
    y DeepSeek.
  - `internal/platform/keyring/`: adaptador al llavero nativo del sistema operativo
    (Credential Manager en Windows, Keychain en macOS, Secret Service en Linux) con
    fallback en memoria para pruebas.
  - `internal/platform/config/`: gestión segura de carpetas privadas de configuración.
- `frontend/src/features`: implementación activa, organizada por funcionalidad:
  `chat`, `projects`, `sessions`, `artifacts`, `settings` y `layout`.
- `frontend/src/components`: componentes reutilizables como iconos (`components/Icons.tsx`).
  Para trabajo nuevo, usa `features/` y evita aumentar la duplicación.
- `frontend/wailsjs`: bindings generados por Wails. No los edites manualmente.

## Comportamiento implementado

- Se puede elegir una carpeta, validar que existe y que permite escritura, y marcarla
  como proyecto activo.
- El explorador muestra un árbol de profundidad máxima cuatro y excluye directorios
  como `.git`, `node_modules`, `dist`, `.idea`, `.vscode` y `build`.
- Las operaciones `ReadProjectFile` y `WriteProjectFile` existen en Go, pero la UI
  todavía no las usa para editar archivos. El árbol sí abre rutas en el explorador
  nativo.
- Proyectos y preferencias de UI se guardan en `localStorage` y configuración nativa.
- **Persistencia completa de sesiones y mensajes con SQLite:** Las sesiones y mensajes ya
  no son volátiles en memoria; se guardan en la base de datos local SQLite (`merlin.db`)
  usando `modernc.org/sqlite` (sin CGO). Se registran id de sesión, proveedor, modelo,
  tokens de prompt y respuesta, duración en segundos, contenido, cadena de pensamiento,
  estado y calificación (`like` / `dislike`). Al alternar sesiones en el panel lateral,
  el historial completo se recupera automáticamente.
- **Protección contra condiciones de carrera durante streaming:** Al iniciar una conversación
  nueva o el primer mensaje de una sesión, el frontend resguarda el flujo en memoria
  (`currentStreamingSessionIdRef`, `skipNextLoadSessionIdRef`) evitando que una consulta
  asíncrona a SQLite sobreescriba los mensajes temporales mientras se reciben tokens en vivo.
- **Seguridad de credenciales de IA:** Las API keys y Admin keys se almacenan
  **exclusivamente en el llavero nativo del sistema operativo**. El frontend solo
  conoce el estado booleano de configuración (`configured: true/false`, `verified: true/false`)
  y una representación enmascarada (`sk-p••••••••UiWA`).
- **Catálogo declarativo de modelos:** Los modelos no están hardcodeados en el código
  de Go; se leen desde `internal/ai/configs/*.json` con alias automáticos (p. ej. `gemini` -> `google`).
- **Prueba de conexión:** El panel de configuración permite enviar un mensaje real de
  prueba al modelo orquestador para confirmar la conexión de punta a punta, diagnosticando
  de forma específica y segura causas de error como falta de créditos/saldo (`insufficient_quota`),
  modelos inexistentes o límites de velocidad (*Rate Limit*).
- **Interfaz limpia de credenciales:** Si una clave ya está configurada, la UI muestra
  únicamente la tarjeta de estado verificado con el botón de "Eliminar"; el input de
  entrada y botón de guardar se ocultan automáticamente hasta que el usuario decida
  eliminar la credencial activa.
- **Streaming en tiempo real en el Chat:** Las consultas al chat usan Server-Sent Events (SSE)
  en el backend Go y eventos de Wails (`chat:stream`). Se soportan los modelos de Google Gemini,
  OpenAI, Anthropic y DeepSeek con detección en vivo de tokens de pensamiento (`reasoning_content`,
  `thinking`, `parts[].thought`) reflejados en el componente `ThoughtChain`.
- **Selector contextual de proveedores:** Se ubica en la barra superior (`Header`) al lado del selector de proyecto. Muestra dinámicamente los proveedores que cuentan con API key configurada y verificada en el llavero nativo, permitiendo alternar de proveedor al instante. La interfaz no expone los modelos: el backend de Go resuelve automáticamente el modelo orquestador configurado en los JSON (`internal/ai/configs/`).
- **Cancelación y control de flujo:** Las peticiones en vuelo pueden ser canceladas por el usuario
  mediante el botón "Detener" en el input del chat, liberando los recursos de streaming y marcando el mensaje adecuadamente.
- **Métricas, código y feedback:** Las respuestas terminadas muestran el tiempo de resolución en segundos, conteo de tokens de entrada/salida consumidos, bloques de código con botón para copiar al portapapeles y botones de feedback (me gusta / no me gusta) persistidos en SQLite.

## Reglas de seguridad

El confinamiento del proyecto y la seguridad de credenciales son propiedades centrales e innegociables.

### Confinamiento de archivos del proyecto
- Toda lectura, escritura, creación, borrado, ejecución o apertura de rutas debe validar primero que permanece dentro del proyecto activo. Usa `workspace.ValidatePathInProject` y devuelve los errores de `internal/domain` cuando corresponda.
- La validación bloquea path traversal léxico (`..`), pero cualquier cambio que toque archivos debe tratar la evasión por symlink/junction como riesgo y cubrirla con pruebas antes de ampliar permisos.
- Mantén el requisito de consentimiento: no se debe acceder a una carpeta hasta que la persona usuaria la haya seleccionado o activado explícitamente.

### Invariantes de seguridad de credenciales de IA
- **Nunca guardar secretos en frontend**: ni en `localStorage` ni en estado de cliente expuesto. Las API keys viven exclusivamente en el llavero nativo del sistema operativo (`Credential Manager` en Windows, `Keychain` en macOS, `Secret Service` en Linux).
- **Nunca poner claves en URLs**: peticiones HTTP a proveedores (como Google Gemini) deben enviar credenciales estrictamente mediante encabezados (`x-goog-api-key`, `Authorization`, `x-api-key`), jamás en parámetros de consulta (`?key=`).
- **Nunca devolver errores internos sin sanitizar**: no concatenar `err.Error()` directamente hacia la interfaz; usar `sanitizeTransportError` para reportar causas clasificadas y seguras sin URLs, tokens ni detalles sensibles de transporte.
- **Nunca persistir campos secretos en JSON ni en SQLite**: los structs de dominio deben marcar secretos con `json:"-"`, la base de datos `merlin.db` guarda únicamente metadatos de sesión y mensajes de texto sin API keys.
- **Nunca ignorar errores del llavero**: las operaciones de consulta, almacenamiento y eliminación en el llavero nativo deben verificar y propagar sus errores a la interfaz para no reportar éxitos falsos.
- **Protección de red y WebView**: bloquear redirecciones entre hosts distintos o no seguras, limitar payloads recibidos (2 MiB), fijar tokens y tamaño de mensajes de prueba (4 KiB), forzar Content Security Policy estricto y servir todas las fuentes tipográficas localmente.

## Convenciones de implementación

- Formatea Go con `go fmt ./...`; conserva los `sync.RWMutex` de los servicios con
  estado compartido.
- Añade tipos Go a `internal/domain`, lógica a un servicio y solo expón lo necesario
  a través de `app.go`.
- En frontend usa TypeScript estricto y módulos de `features/`; evita `any` salvo en
  límites inevitables de una API externa.
- Reutiliza los tokens de `frontend/src/style.css`, la configuración de Tailwind y
  los iconos SVG existentes. El aspecto deseado es oscuro, compacto y técnico;
  `DESIGN.md` es la referencia visual.
- No almacenes secretos, contenido de archivos de usuario ni rutas privadas en logs,
  telemetría o datos de ejemplo.
- `build/bin`, `node_modules` y `frontend/dist` están ignorados. No agregues sus
  artefactos generados al control de versiones.
- Respeta cambios locales existentes, especialmente los archivos binarios de diseño
  bajo `styles/`; no los reviertas para completar trabajo no relacionado.

## Validación

Desde la raíz:

```powershell
go test ./...
cd frontend
npm run build
```

Si el ejecutable global de `npm` no está disponible pero `frontend/node_modules` ya
existe, las verificaciones equivalentes son:

```powershell
cd frontend
node .\node_modules\typescript\bin\tsc --noEmit
node .\node_modules\vite\bin\vite.js build
```

Para ejecutar la aplicación de escritorio en desarrollo se necesita Wails CLI v2:

```powershell
wails dev
```

Las pruebas actuales cubren principalmente el sandbox básico, concurrencia de
servicios, persistencia de sesiones en SQLite y persistencia de ventana. Al modificar seguridad de rutas, añade casos
para rutas absolutas, `..`, enlaces simbólicos/junctions y errores de permisos.

## Ruta de evolución recomendada

1. **[Completado] Adaptador de proveedores de IA y seguridad de credenciales:**
   - Clientes HTTP seguros con `internal/ai/transport` para OpenAI, Anthropic, Google Gemini y DeepSeek.
   - Almacenamiento seguro en llavero nativo del sistema (`internal/platform/keyring`).
   - Catálogo declarativo de modelos en JSON (`internal/ai/configs/*.json`).
   - Diagnóstico detallado de errores (cuota, saldo, modelos y límites de tasa).
2. **[Completado] Streaming en el chat:**
   - Integración de Server-Sent Events (SSE) y emisión de eventos Wails (`chat:stream`).
   - Soporte para streaming de tokens de respuesta y cadena de pensamiento (reasoning) para OpenAI, Anthropic, Gemini y DeepSeek.
   - Selector visual de proveedor en la barra superior (`Header`) al lado del proyecto con los proveedores verificados en el llavero nativo.
   - Cancelación fluida y sin bloqueos en Go mediante goroutines y `context.WithCancel`.
3. **[Completado] Persistencia de sesiones y mensajes en SQLite:**
   - Persistencia local-first con `modernc.org/sqlite` (sin CGO ni dependencias nativas).
   - Tablas relacionales `sessions` y `messages` con `ON DELETE CASCADE` y modo WAL.
   - Guardado automático de tokens (prompt y completion), duración en segundos, modelo, proveedor y feedback (like/dislike).
   - Protección contra condiciones de carrera para no pisar el streaming en curso al crear nuevas sesiones.
4. **Bucle de agente y herramientas de workspace:**
   - Implementar herramientas que interactúen con `workspace.Service` (lectura, búsqueda, edición con vista previa y consentimiento explícito del usuario antes de aplicar cambios).
5. **Endurecimiento adicional del sandbox:**
   - Añadir validación estricta contra evasión por symlinks/junctions y límites de tamaño por archivo.
6. **Limpieza de componentes duplicados:**
   - Consolidar los componentes heredados de `frontend/src/components` tras confirmar que no tienen consumidores activos en `frontend/src/features`.
