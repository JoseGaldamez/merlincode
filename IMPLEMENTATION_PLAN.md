# Plan de implementación: seguridad de credenciales de proveedores de IA

## Objetivo

Endurecer la integración con proveedores de IA para que las API keys no puedan filtrarse mediante URLs, errores, archivos, almacenamiento del navegador o redirecciones; que las operaciones del llavero nativo sean verificables; y que las compilaciones sean reproducibles y utilicen herramientas sin vulnerabilidades conocidas aplicables.

Este plan cubre la integración actual con Anthropic, OpenAI, Google y DeepSeek.

## Estado de validación (2026-09-13)

Resultado: **implementación parcial; conservar este archivo**. El checklist final fue
corregido tras contrastarlo con el código, las pruebas, el estado de Git y las
compilaciones locales.

Validaciones que pasan:

- `go test -count=1 ./...`
- `go vet ./...`
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
- `npm run build`
- `wails build` (compilación de producción sin limpieza previa)

Pendientes o validaciones no concluyentes:

- `npm audit` reporta 3 vulnerabilidades en dependencias de desarrollo: 1 alta y
  2 moderadas, principalmente por Vite 3.2.11 y esbuild 0.15.x.
- `npm ci` y `wails build -clean` no pudieron validarse mientras estaban activos
  `wails`, `merlincode-dev.exe` y `esbuild.exe`, porque Windows bloqueó el reemplazo
  de esos ejecutables. Deben repetirse con el entorno de desarrollo cerrado.
- Wails permanece en `v2.12.0` en `go.mod`, la CLI y CI; el plan exige `v2.15.0`.
- `.github/workflows/ci.yml`, `frontend/package-lock.json` y `frontend/wailsjs/`
  todavía están sin seguimiento en Git. Un checkout limpio no recibe esos archivos;
  además, el job frontend no genera bindings antes de compilar.
- `wails.json` todavía usa `npm install`, por lo que el build de Wails no aplica
  `npm ci` como instalación reproducible.
- `persistedProviderMetadata` conserva `AccountInfo`; en DeepSeek ese texto contiene
  el saldo, por lo que aún se persiste información financiera.
- `resolveConfigFilePath`, `loadAndMigrate`, `hydrateFromKeyring` y algunos llamados
  a `persistCredentialsLocked` descartan errores. Los fallos de lectura del llavero,
  migración, permisos o persistencia no siempre llegan a la interfaz.
- La migración no restringe primero los permisos del archivo heredado, no corrige
  permisos de archivos/directorios existentes y no documenta el plazo de
  compatibilidad del formato anterior.
- En respuestas HTTP no exitosas se reenvía `error.message` del proveedor con solo
  truncado, sin una sanitización por lista permitida.
- Faltan pruebas específicas para el encabezado de Google y ausencia de la clave en
  la URL, limpieza de `localStorage`, migración, `ErrNotFound`, borrado parcial,
  concurrencia, límite de salida y cancelación durante el cierre.
- CSP y llaveros nativos aún requieren las pruebas manuales multiplataforma descritas
  en la fase 6.
- No existe en la aplicación ni en la documentación para usuarios la recomendación
  de rotar claves de Google que pudieron haberse expuesto en URLs.

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

- `internal/aiproviders/google.go`

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

- `internal/aiproviders/validator.go`
- `internal/aiproviders/anthropic.go`
- `internal/aiproviders/openai.go`
- `internal/aiproviders/google.go`
- `internal/aiproviders/deepseek.go`

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

- `internal/aiproviders/validator.go`

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
- `internal/aiproviders/service.go`

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

- `internal/aiproviders/keyring.go`

Cambios:

1. Introducir una interfaz `SecretStore` con operaciones `Get`, `Set` y `Delete`.
2. Implementar `OSKeyringStore` usando `go-keyring`.
3. Inyectar el almacén en `Service`; usar un almacén falso en pruebas.
4. Diferenciar secreto inexistente de error de acceso al llavero.
5. No convertir errores reales de lectura en un simple resultado “no configurado”.

### 3.3 Eliminación coherente de claves

Archivos:

- `internal/aiproviders/keyring.go`
- `internal/aiproviders/service.go`
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

- `internal/aiproviders/service.go`
- Nuevo archivo opcional `internal/aiproviders/migration.go`

Cambios:

1. Crear un DTO exclusivamente para leer archivos antiguos con `apiKey` y `adminApiKey`.
2. Restringir los permisos del archivo existente antes de procesarlo.
3. Guardar cada secreto en el llavero y comprobar el resultado.
4. Reescribir el archivo sin secretos únicamente después de una migración confirmada.
5. Si falla el llavero, conservar la única copia, no registrarla y mostrar un error de migración.
6. Documentar durante cuánto tiempo se mantendrá compatibilidad con el formato heredado.

### 3.5 Persistencia atómica y permisos

Archivo:

- `internal/aiproviders/service.go`

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

- `internal/aiproviders/service.go`
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

- [x] Google autentica mediante `x-goog-api-key`.
- [ ] Ningún error público expone errores internos directamente ni mediante `%w`.
- [ ] Los mensajes de error devueltos por los proveedores se sanitizan mediante una lista permitida.
- [x] Redirects entre hosts no reciben credenciales.
- [x] Todas las respuestas HTTP tienen límite.
- [x] `merlin_app_settings` elimina claves heredadas.
- [x] Los secretos tienen `json:"-"`.
- [ ] El archivo de metadata no contiene balances ni claves.
- [ ] Los errores del llavero se propagan en lectura, escritura, eliminación y migración.
- [x] Se detectan secretos huérfanos.
- [x] La migración no borra la única copia ante un fallo.
- [ ] La migración restringe permisos antes de leer y reporta sus fallos a la interfaz.
- [ ] Directorio y metadata utilizan y corrigen permisos restrictivos sin ignorar errores.
- [x] La escritura de metadata es atómica.
- [ ] Go de CI/release está fijado explícitamente en 1.26.5 o posterior.
- [ ] Wails está alineado en `v2.15.0` en módulo, CLI y CI.
- [ ] `package-lock.json` está versionado y CI/releases usan `npm ci`.
- [ ] Los bindings Wails se generan o versionan de forma reproducible antes del typecheck.
- [x] `govulncheck` forma parte de CI.
- [x] Modelo, tamaño, concurrencia y salida se validan en Go.
- [ ] `npm audit` no reporta vulnerabilidades conocidas aplicables.
- [ ] Las pruebas de aceptación faltantes de las fases 1, 2, 3 y 5 están implementadas.
- [ ] Fuentes locales y CSP funcionan en desarrollo y en el binario en pruebas manuales.
- [ ] Pruebas Go, frontend y Wails pasan desde un clon limpio.
- [ ] Usuarios que probaron Google con errores de red reciben recomendación de rotar la clave.
