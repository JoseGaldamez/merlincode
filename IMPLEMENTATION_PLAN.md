# Plan de implementación: seguridad de credenciales de proveedores de IA

## Objetivo

Endurecer la integración con proveedores de IA para que las API keys no puedan filtrarse mediante URLs, errores, archivos, almacenamiento del navegador o redirecciones; que las operaciones del llavero nativo sean verificables; y que las compilaciones sean reproducibles y utilicen herramientas sin vulnerabilidades conocidas aplicables.

Este plan cubre la integración actual con Anthropic, OpenAI, Google y DeepSeek.

## Estado de validación (2026-09-14)

Las correcciones de código de esta revisión están implementadas. Conservar el plan
hasta completar las comprobaciones manuales multiplataforma y ejecutar los workflows
sobre el commit que contenga estos cambios.

Validaciones realizadas en Windows:

- `go test -count=1 ./...` y `go vet ./...`: pasan, incluidos sandbox/junctions,
  carreras lógicas del llavero, cancelación, límites y errores SSE con centinelas.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...`: sin vulnerabilidades encontradas.
- `npm ci --include=dev`, `npm test` (9 pruebas), typecheck y build: pasan.
- `npm audit`: cero vulnerabilidades reportadas, incluidas las nuevas dependencias
  DOMPurify y jsdom (esta última solo para pruebas).
- `wails build -clean`: pasa con Wails 2.15.0. Los bindings regenerados no tienen
  diferencias de contenido con los versionados.
- Windows Credential Manager: prueba real con cuenta sintética aislada; guardado,
  lectura, borrado y comprobación de ausencia pasan sin usar claves de usuario.
- ACL de Windows: prueba real de reparación de permisos heredados; solo se conceden
  permisos al usuario actual y a LocalSystem. En Unix se mantienen 0700/0600.

Correcciones adicionales al plan original:

- Lectura, escritura y árbol utilizan un handle `os.Root` del proyecto seleccionado.
  Se rechazan enlaces/junctions, rutas externas y archivos especiales; los archivos
  tienen un límite de 4 MiB. El explorador valida primero la ruta. La comprobación
  de permisos crea un temporal exclusivo y no sobrescribe nombres predecibles.
- El HTML del chat se sanitiza con una lista limitada de etiquetas y atributos,
  sin estilos, eventos, formularios, SVG ni esquemas de enlace fuera de HTTP(S).
- Se elimina la configuración heredada del navegador también cuando coexiste con v2.
- El streaming valida modelos y roles, limita la entrada a 256 KiB/200 mensajes,
  la salida y el cuerpo SSE a 2 MiB y la duración a cinco minutos. Se admite una
  generación simultánea; la cancelación conserva su registro hasta finalizar.
  Todos los proveedores solicitan como máximo 4096 tokens de salida.
- Los errores de streaming se sanitizan antes de la UI y SQLite; no se registran
  cuerpos del proveedor ni errores internos de transporte.
- Guardados y borrados del llavero se serializan junto con su estado en memoria.
  Se reconcilian operaciones parciales y se conservan las claves admin al renovar
  la clave principal. Persistir con una ruta vacía falla antes de tocar permisos.
- La CSP de desarrollo autoriza el preámbulo de React con un nonce aleatorio por
  arranque del servidor, limitado a loopback. Producción no permite scripts inline.
- CI incluye pruebas con detector de carreras, vet, govulncheck, npm audit, typecheck,
  build y compilación limpia de Wails. Go 1.26.6, Wails 2.15.0, Node 22.13.1 y npm
  10.9.2 están fijados. Se documentó y mostró en ajustes la rotación de claves Google.

Validaciones que siguen pendientes (no confundir con defectos corregidos):

- `go test -race` local requiere CGO y un compilador C; este equipo no lo tiene.
  El control está configurado en los runners de CI.
- Ejecutar CI/release desde un checkout limpio del futuro commit. No se ha publicado
  ni creado ningún commit o release durante esta revisión.
- Verificación visual de la CSP y fuentes en el WebView nativo, tanto en desarrollo
  como en producción. Hay pruebas automáticas del HTML/CSP, pero no sustituyen esa
  comprobación visual ni certifican todas las plataformas.
- macOS Keychain y Linux Secret Service (disponible/no disponible), instalación nueva
  y actualización real desde una versión antigua. La migración y sus fallos cuentan
  con pruebas automáticas usando almacenes aislados.

## Condiciones para publicar

Antes de generar un release deben completarse, como mínimo, las fases 1 a 4. El release se considera seguro cuando:

- Ninguna clave aparece en URLs, mensajes de error, logs, JSON o `localStorage`.
- El frontend nunca recibe una API key completa desde Go.
- Un fallo del llavero nativo llega al usuario y no se informa un éxito falso.
- Un checkout limpio instala dependencias, genera bindings, ejecuta el typecheck y compila.
- CI y releases utilizan Go 1.26.5 o una versión posterior corregida.
- Las pruebas de seguridad usan claves centinela y comprueban explícitamente su ausencia.

## Fase 1: eliminar filtraciones inmediatas

### 1.1 Autenticación de Google mediante encabezado

Archivos:

- `internal/ai/providers/google/client.go`

Cambios:

1. Eliminar el parámetro `?key=` de los endpoints de listado de modelos y `generateContent`.
2. Construir URLs que nunca contengan secretos.
3. Enviar la credencial únicamente mediante el encabezado `x-goog-api-key`.
4. Mantener los nombres de modelo y demás parámetros validados o escapados independientemente.

Criterios de aceptación:

- Una clave centinela no aparece en `req.URL.String()`.
- El servidor de prueba recibe la clave en `x-goog-api-key`.
- Los errores de transporte no contienen la clave.

### 1.2 Errores seguros para todos los proveedores

Archivos:

- `internal/ai/transport/transport.go`
- `internal/ai/providers/anthropic/client.go`
- `internal/ai/providers/openai/client.go`
- `internal/ai/providers/google/client.go`
- `internal/ai/providers/deepseek/client.go`

Cambios:

1. Crear un helper común que convierta errores internos en categorías públicas seguras, por ejemplo timeout, conexión fallida, respuesta inválida o servicio no disponible.
2. Dejar de concatenar `err.Error()` en resultados enviados al frontend.
3. No mostrar URLs completas, encabezados, cuerpos de solicitud ni respuestas sin filtrar.
4. Si se añade logging interno, registrar solamente proveedor, operación, código HTTP y una causa sanitizada. Nunca registrar claves ni encabezados de autenticación.

Criterios de aceptación:

- Los resultados públicos no contienen URLs ni claves centinela.
- La UI conserva mensajes comprensibles sin detalles internos sensibles.

### 1.3 Cliente HTTP endurecido

Archivo:

- `internal/ai/transport/transport.go`

Cambios:

1. Configurar `CheckRedirect` para bloquear redirecciones o permitir únicamente HTTPS hacia el mismo host exacto.
2. No reenviar `Authorization`, `x-api-key` ni `x-goog-api-key` a otro origen.
3. Mantener un timeout global y añadir timeouts de contexto por operación.
4. Crear un helper `decodeJSONLimited` que limite el cuerpo de respuesta, por ejemplo a 2 MiB, y detecte respuestas truncadas.
5. Usar el helper en todos los endpoints de validación, uso, modelos y mensajes.

Pruebas:

- Un servidor A redirige a un servidor B y B no recibe credenciales.
- Una respuesta mayor al límite falla de forma controlada.
- Timeouts y cancelaciones producen mensajes sanitizados.

## Fase 2: eliminar secretos heredados del navegador

Archivo:

- `frontend/src/features/settings/hooks/useSettingsForm.ts`

Riesgo existente:

Versiones anteriores guardaban `apiKey` dentro de `merlin_app_settings`. Aunque el campo ya no existe en el tipo actual, expandir el objeto completo con `{...parsed}` conserva propiedades desconocidas en tiempo de ejecución y puede volver a escribirlas.

Cambios:

1. Reemplazar la expansión del JSON por una deserialización mediante lista permitida.
2. Copiar únicamente preferencias conocidas: proveedor, modelo, temperatura, streaming, tema, idioma, autoguardado y telemetría.
3. Ignorar explícitamente propiedades desconocidas.
4. Detectar campos heredados como `apiKey`, `adminApiKey` y `apiEndpoint`.
5. Reescribir inmediatamente `merlin_app_settings` con el objeto sanitizado.
6. No migrar automáticamente una clave heredada hacia Go. Eliminarla y solicitar que el usuario vuelva a introducirla en la pantalla protegida.
7. Usar la misma función de serialización permitida cada vez que se guarden preferencias.

Pruebas:

- Cargar preferencias heredadas con una clave centinela.
- Verificar que la clave no entra al estado React.
- Verificar que desaparece de `localStorage`.
- Guardar nuevamente y comprobar que no reaparece.

## Fase 3: almacenamiento robusto en el llavero nativo

### 3.1 Separar secretos y metadatos

Archivos:

- `internal/domain/aiprovider.go`
- `internal/ai/service.go`

Cambios:

1. Marcar `APIKey` y `AdminAPIKey` con `json:"-"`.
2. Crear un DTO privado, por ejemplo `persistedProviderMetadata`, sin campos secretos.
3. Persistir únicamente ID, estado de verificación, fecha y modelos necesarios.
4. No persistir balances ni información financiera en `AccountInfo`; obtenerla bajo demanda y mantenerla en memoria.
5. Eliminar la redacción manual como única barrera contra serialización accidental.

Pruebas:

- `json.Marshal` de una credencial nunca incluye API keys.
- Leer el archivo persistido y asegurar que no contiene claves centinela.
- Reiniciar el servicio y comprobar que el estado se reconstruye desde metadata y llavero.

### 3.2 Abstraer el llavero

Archivo:

- `internal/platform/keyring/keyring.go`

Cambios:

1. Introducir una interfaz `SecretStore` con operaciones `Get`, `Set` y `Delete`.
2. Implementar `OSKeyringStore` usando `go-keyring`.
3. Inyectar el almacén en `Service`; usar un almacén falso en pruebas.
4. Diferenciar secreto inexistente de error de acceso al llavero.
5. No convertir errores reales de lectura en un simple resultado “no configurado”.

### 3.3 Eliminación coherente de claves

Archivos:

- `internal/platform/keyring/keyring.go`
- `internal/ai/service.go`
- `app.go`
- Hook y componentes de ajustes del frontend.

Cambios:

1. Hacer que `deleteKeyringSecret`, `ClearKey` y `ClearAdminKey` devuelvan errores.
2. Actualizar la fachada Wails para propagar esos errores.
3. Mostrar en la UI si la eliminación no pudo completarse.
4. Volver a consultar el llavero después de una eliminación parcial y reflejar el estado real.
5. Eliminar metadata solamente cuando el secreto correspondiente haya desaparecido.
6. Durante el arranque, consultar las cuentas de todos los proveedores conocidos aunque no tengan metadata. Esto permite encontrar secretos huérfanos.

Pruebas:

- Eliminación exitosa de clave normal y admin.
- `ErrNotFound` se trata como éxito idempotente.
- Un fallo real mantiene un estado coherente y llega al frontend.
- Una clave huérfana se detecta y puede eliminarse.

### 3.4 Migración segura del formato anterior

Archivos:

- `internal/ai/service.go`
- Nuevo archivo opcional `internal/ai/service.go`

Cambios:

1. Crear un DTO exclusivamente para leer archivos antiguos con `apiKey` y `adminApiKey`.
2. Restringir los permisos del archivo existente antes de procesarlo.
3. Guardar cada secreto en el llavero y comprobar el resultado.
4. Reescribir el archivo sin secretos únicamente después de una migración confirmada.
5. Si falla el llavero, conservar la única copia, no registrarla y mostrar un error de migración.
6. Documentar durante cuánto tiempo se mantendrá compatibilidad con el formato heredado.

### 3.5 Persistencia atómica y permisos

Archivo:

- `internal/ai/service.go`

Cambios:

1. Crear el directorio de configuración con modo `0700` cuando el sistema lo soporte.
2. Escribir metadata con modo `0600`.
3. Escribir primero en un archivo temporal dentro del mismo directorio.
4. Sincronizar, cerrar y renombrar atómicamente sobre el archivo final.
5. Corregir permisos de archivos existentes.
6. Propagar errores de `MkdirAll`, `Marshal`, escritura, sincronización y renombrado.

## Fase 4: toolchain y cadena de suministro

Archivos:

- `go.mod`
- `.gitignore`
- `frontend/package-lock.json`
- Configuración de CI y documentación de desarrollo.

Cambios:

1. Fijar Go 1.26.5 o superior para CI y releases mediante `toolchain` o configuración equivalente.
2. Alinear la CLI Wails con `github.com/wailsapp/wails/v2 v2.15.0`.
3. Dejar de ignorar `frontend/package-lock.json`.
4. Regenerar y versionar el lockfile con una versión documentada de Node/npm.
5. Usar `npm ci` en CI y releases.
6. Resolver el tratamiento de `frontend/wailsjs`:
   - Opción preferida: versionarlo para que el frontend pueda compilar desde un checkout limpio.
   - Alternativa: generarlo en CI con una CLI Wails fijada antes del typecheck y comprobar que la generación sea determinista.
7. Añadir controles obligatorios de CI:
   - `go test ./...`
   - `go vet ./...`
   - `govulncheck ./...`
   - `npm ci`
   - Typecheck y build del frontend.
   - `wails build`

Criterio de aceptación:

- Un clon limpio compila sin depender de archivos generados manualmente ni de versiones flotantes.

## Fase 5: limitar abuso y consumo accidental

Archivos:

- `internal/ai/service.go`
- `app.go`
- Implementaciones de cada proveedor.

Cambios:

1. Validar en Go que el modelo solicitado pertenece al proveedor seleccionado y está dentro de los modelos permitidos.
2. Limitar el tamaño del mensaje de prueba, por ejemplo a 4 KiB.
3. Fijar `max_tokens` o `maxOutputTokens` en todos los proveedores, incluido Google.
4. Limitar el tamaño del texto devuelto al frontend.
5. Permitir una sola petición de prueba simultánea por proveedor.
6. Añadir un cooldown breve contra clics repetidos o automatización accidental.
7. Reemplazar `context.Background()` por el contexto de vida de la aplicación más `context.WithTimeout`.
8. Cancelar solicitudes pendientes durante el cierre de la aplicación.

Pruebas:

- Modelo arbitrario rechazado antes de hacer una llamada HTTP.
- Mensaje excesivo rechazado.
- Segunda petición simultánea bloqueada.
- Cancelación y timeout no dejan operaciones colgadas.

## Fase 6: defensa del WebView y validación del release

Archivos:

- `frontend/index.html`
- Recursos y estilos del frontend.
- `AGENTS.md`

Cambios:

1. Servir Geist y JetBrains Mono desde recursos locales.
2. Eliminar solicitudes runtime a Google Fonts.
3. Añadir una Content Security Policy restrictiva y compatible con Wails.
4. Mantener scripts, estilos, fuentes, imágenes y conexiones limitados a los orígenes estrictamente necesarios.
5. Probar la política tanto en desarrollo como en el binario de producción.
6. Documentar en `AGENTS.md` las invariantes de seguridad:
   - Nunca guardar secretos en frontend.
   - Nunca poner claves en URLs.
   - Nunca devolver errores internos sin sanitizar.
   - Nunca persistir campos secretos en JSON.
   - Nunca ignorar errores del llavero.

Validación manual:

- Windows Credential Manager.
- macOS Keychain.
- Linux Secret Service con sesión gráfica y con servicio no disponible.
- Instalación nueva y actualización desde una versión con configuración heredada.

## Estrategia de commits

1. `security: stop API keys leaking through URLs and errors`
2. `security: purge legacy browser-stored credentials`
3. `security: make keyring operations explicit and testable`
4. `security: separate secrets from persisted provider metadata`
5. `build: pin secure toolchains and restore dependency lockfile`
6. `security: add request limits and WebView defenses`
7. `test: enforce credential security invariants in CI`

## Checklist final

- [x] Google autentica mediante `x-goog-api-key`, incluido streaming.
- [x] Errores públicos de proveedores sanitizados en la fachada, incluido streaming.
- [x] Mensajes HTTP del proveedor clasificados sin reenviar cuerpos a UI o logs.
- [x] Redirects entre hosts no reciben credenciales.
- [x] Respuestas JSON y cuerpos SSE con límite acumulado.
- [x] Preferencias del navegador sin secretos heredados, incluida coexistencia con v2.
- [x] Los secretos tienen `json:"-"`.
- [x] Metadata sin balances ni claves.
- [x] Errores del llavero propagados en lectura, escritura, eliminación y migración.
- [x] Secretos huérfanos detectados.
- [x] Migración conserva la única copia ante fallos.
- [x] Permisos restringidos antes de leer y errores de migración visibles.
- [x] Permisos reparados con 0700/0600 en Unix y ACL privadas en Windows.
- [x] Metadata escrita mediante temporal, sincronización y reemplazo.
- [x] Go de CI/release fijado en 1.26.6.
- [x] Wails alineado en v2.15.0 en módulo, CLI y release/CI.
- [x] Lockfile versionado y builds utilizan `npm ci`.
- [x] Bindings versionados y regeneración sin diferencias de contenido.
- [x] `govulncheck`, `go vet`, pruebas, auditoría npm y builds configurados en CI.
- [x] Modelo, tamaño, concurrencia, duración y salida validados en Go.
- [x] `npm audit` y `govulncheck` sin vulnerabilidades reportadas en esta revisión.
- [x] Pruebas de aceptación de las fases 1, 2, 3 y 5 implementadas.
- [x] Sandbox endurecido con root handle y pruebas de escape/junctions.
- [x] HTML del chat sanitizado y probado con contenido malicioso.
- [x] Recomendación de rotar claves Google publicada en README y ajustes locales.
- [ ] Detector de carreras ejecutado satisfactoriamente en CI.
- [ ] Fuentes y CSP verificadas visualmente en WebView de desarrollo y producción.
- [ ] Pruebas Go, frontend y Wails verificadas desde checkout limpio en CI.
- [ ] Llavero y actualización real verificados también en macOS y Linux.
