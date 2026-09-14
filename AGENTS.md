# Merlin Code: guía para agentes

## Propósito y estado del producto

Merlin Code es una aplicación de escritorio local-first para asistir en tareas de
programación dentro de una carpeta elegida explícitamente por la persona usuaria.
Está construida con Go + Wails v2 en el backend y React 18 + TypeScript + Vite +
Tailwind en el frontend.

La interfaz, la gestión de proyectos y el sandbox de archivos están implementados.
La integración de IA **no** lo está todavía: el chat llama a `App.Greet`, que devuelve
un saludo de prueba. Los proveedores, modelos, streaming, telemetría, artefactos y
costes mostrados en la UI son configuración o datos simulados; no existen clientes
HTTP de proveedores ni un bucle de agente real. No presentes estas partes como
funcionalidad terminada ni supongas que una API key configurada se está usando.

## Arquitectura y flujo de datos

```text
React (frontend/src) -- Wails bindings --> app.go (fachada)
                                              |
                         +--------------------+--------------------+
                         |                                         |
              internal/workspace.Service                internal/window.Service
              proyecto y sistema de archivos            estado de ventana y paneles
```

- `main.go`: arranque de Wails, ventana sin bordes y assets embebidos desde
  `frontend/dist`.
- `app.go`: única fachada pública de Go para Wails. Debe permanecer delgada:
  delega a servicios, no contiene reglas de negocio.
- `internal/domain`: entidades y errores sin dependencias de infraestructura.
- `internal/workspace`: proyecto activo, árbol, lectura/escritura y explorador
  del sistema.
- `internal/window`: persistencia de tamaño, maximizado y paneles en el directorio
  de configuración del usuario (`merlincode/window.json`).
- `frontend/src/features`: implementación activa, organizada por funcionalidad:
  `chat`, `projects`, `sessions`, `artifacts`, `settings` y `layout`.
- `frontend/src/components`: hay componentes heredados duplicados. Los iconos de
  `components/Icons.tsx` se reutilizan; para trabajo nuevo, usa `features/` y evita
  aumentar la duplicación.
- `frontend/wailsjs`: bindings generados por Wails. No los edites manualmente.

## Comportamiento implementado

- Se puede elegir una carpeta, validar que existe y que permite escritura, y marcarla
  como proyecto activo.
- El explorador muestra un árbol de profundidad máxima cuatro y excluye directorios
  como `.git`, `node_modules`, `dist`, `.idea`, `.vscode` y `build`.
- Las operaciones `ReadProjectFile` y `WriteProjectFile` existen en Go, pero la UI
  todavía no las usa para editar archivos. El árbol sí abre rutas en el explorador
  nativo.
- Proyectos y preferencias de UI se guardan mayoritariamente en `localStorage`.
  Las sesiones, mensajes y artefactos viven solo en memoria: el cambio de sesión no
  recupera otra conversación y la opción de auto-guardado aún no tiene efecto real.
- La configuración de IA, incluida la API key, se guarda sin cifrar en `localStorage`.
  No la muevas ni la registres en logs; una integración real debe migrarla a un
  almacén seguro del sistema operativo.

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
- **Nunca persistir campos secretos en JSON**: los structs de dominio deben marcar secretos con `json:"-"`, la metadata se guarda mediante DTOs atómicos con permisos `0600`/`0700` y sin datos de balance financiero.
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
servicios y persistencia de ventana. Al modificar seguridad de rutas, añade casos
para rutas absolutas, `..`, enlaces simbólicos/junctions y errores de permisos.

## Ruta de evolución recomendada

1. Endurecer el sandbox contra symlinks/junctions y definir límites de tamaño y tipos
   de archivo.
2. Crear un adaptador de proveedores de IA con secretos seguros, streaming y manejo
   explícito de errores/cancelación.
3. Implementar herramientas del agente que usen el servicio de workspace, con vista
   previa y aprobación antes de escribir.
4. Persistir sesiones, mensajes, artefactos y el proyecto activo de forma coherente.
5. Consolidar los componentes heredados duplicados tras comprobar que no tienen
   consumidores.
