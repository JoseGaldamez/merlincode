<div align="center">

  <img src="./frontend/src/assets/images/logo.png" alt="Merlin Code Logo" width="110" height="110" />

  # Merlin Code

  **El asistente de programación en pareja e ingeniería de software asistido por IA para escritorio.**  
  *Construido con Go, Wails v2, React 18, Vite y Tailwind CSS.*

  <p>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-teal.svg?style=for-the-badge" alt="License: MIT" /></a>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.26.9-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" /></a>
    <a href="https://wails.io"><img src="https://img.shields.io/badge/Wails-v2.15.0-DF0000?style=for-the-badge&logo=wails&logoColor=white" alt="Wails Version" /></a>
    <a href="https://react.dev"><img src="https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React 18" /></a>
    <a href="https://vitejs.dev"><img src="https://img.shields.io/badge/Vite-7-646CFF?style=for-the-badge&logo=vite&logoColor=white" alt="Vite" /></a>
    <a href="https://tailwindcss.com"><img src="https://img.shields.io/badge/Tailwind-3-38B2AC?style=for-the-badge&logo=tailwind-css&logoColor=white" alt="Tailwind CSS" /></a>
  </p>

  <p>
    <a href="#descripción-del-proyecto">Descripción</a> •
    <a href="#características-principales">Características</a> •
    <a href="#guía-de-instalación-desde-cero">Instalación desde Cero</a> •
    <a href="#ejecución-y-desarrollo">Ejecución</a> •
    <a href="#claves-de-api-de-proveedores-de-ia">Claves de API</a> •
    <a href="#compilación-para-producción">Compilación</a> •
    <a href="#arquitectura-del-proyecto">Arquitectura</a> •
    <a href="#creador">Creador</a> •
    <a href="#licencia">Licencia</a>
  </p>

</div>

---

## Descripción del Proyecto

**Merlin Code** es una aplicación de escritorio multiplataforma (Windows, macOS y Linux) diseñada para actuar como un compañero inteligente de desarrollo y pair programming. A diferencia de las herramientas web tradicionales, Merlin Code se ejecuta como una aplicación nativa ligera con bajo consumo de memoria y acceso seguro y controlado al sistema de archivos local.

La aplicación opera bajo un modelo de **espacio de trabajo confinado (sandboxing)**: únicamente tiene acceso de lectura y escritura a la carpeta que el usuario selecciona explícitamente a través del diálogo nativo del sistema operativo, garantizando la máxima seguridad y privacidad de tu código fuente.

---

## Características Principales

- **Espacio de Trabajo Confinado (Sandboxing Estricto):**
  - Selección de carpeta mediante el diálogo nativo del sistema operativo (`runtime.OpenDirectoryDialog`).
  - Verificación rigurosa de permisos de escritura reales antes de activar un proyecto.
  - Protección activa contra ataques de *Path Traversal* (evita lecturas o escrituras que utilicen secuencias como `../` para salir de la carpeta autorizada).
- **Explorador de Archivos Jerárquico Integrado:**
  - Visualización en árbol del directorio del proyecto en un panel lateral interactivo.
  - Exclusión automática de directorios pesados o binarios (`.git`, `node_modules`, `dist`, `.vscode`, `build`, etc.).
  - Botón de acceso directo para abrir la carpeta del proyecto en el explorador de archivos nativo de tu sistema operativo (Windows Explorer, macOS Finder o Linux File Manager).
- **Diseño Moderno, Modular y Persistente:**
  - Ventana personalizada sin bordes (*frameless*) con controles nativos (minimizar, maximizar/restaurar, cerrar).
  - Memorización persistente del tamaño de la ventana, modo maximizado y visibilidad de los paneles laterales en `%APPDATA%/merlincode/window.json`.
  - Paneles laterales y cajón inferior colapsables y redimensionables con límites ergonómicos de altura y anchura.
- **Entorno de Chat y Pair Programming:**
  - Interfaz conversacional limpia con tema oscuro *petrol/slate*.
  - Vista de artefactos generados, historial de sesiones y configuración de modelos.
  - Mensajes de bienvenida contextuales que te guían a seleccionar un proyecto antes de interactuar con el asistente.

---

## Ejecutar aplicaciones desde el chat

Selecciona un proyecto y pide al agente que ejecute la aplicación. El agente inspecciona
los archivos y prepara una tarjeta con el comando, la carpeta y un botón **Play**.
Solo ese clic inicia la aplicación; recibir una propuesta o recuperar el historial no
ejecuta procesos.

- **JavaScript/TypeScript:** el agente lee `package.json` para analizar los scripts,
  las dependencias y `packageManager`. El backend verifica el manifiesto y los lockfiles
  de npm, pnpm, Yarn y Bun, incluidos los de un workspace superior. Prioriza `dev`, `start`,
  `serve` y `preview`, y también reconoce nombres personalizados como `site:dev` según
  su contenido. Admite React/Vue/Svelte con Vite, React Scripts, Angular CLI,
  Vue CLI, Next, Astro, Nuxt y Webpack, y entradas
  de servidores Node/Bun. Rechaza comandos compuestos, hooks pre/post y argumentos
  fuera de los perfiles admitidos. Una configuración ambigua requiere corregir o
  declarar el gestor; no se elige otro silenciosamente.
- **Python:** ejecuta archivos `.py` seleccionados explícitamente (incluidos scripts
  cortos como `hola_mundo.py`) y reconoce servidores FastAPI, Flask, Django y
  `http.server`, con uv, Poetry, Pipenv o un intérprete local. No requiere convertir
  un script en servidor ni agregar un bloque `__main__`. El agente puede indicar
  una entrada real dentro de una subcarpeta. Los entornos virtuales deben cumplir la política de
  rutas sin enlaces del workspace.
- **Go:** detecta entradas `package main` con `func main`, con o sin `go.mod`.
  Los archivos independientes (por ejemplo `server.go`) se ejecutan con `go run server.go`;
  los módulos se ejecutan como paquetes para incluir sus archivos auxiliares.
- **HTML estático:** utiliza el servidor integrado, enlazado a localhost y un puerto
  disponible, sin depender de Python ni descargar herramientas.

Al pulsar Play se abre la sección **Ejecuciones** del panel derecho, debajo de
Artefactos, con altura ajustable y plegado independiente. Cada aplicación
ofrece Stop, relanzamiento, limpieza de logs (borrador) y una consola desplegable.
La papelera detiene el proceso y elimina su entrada del panel.
Si una ejecución falla, **Pedir solución a Merlin** envía al chat del proyecto el
comando, la entrada, el código de salida y los logs recientes para preparar una corrección con aprobación.
Editar el código y relanzarlo actualiza la ejecución sin duplicarla, siempre que conserve el mismo
comando y perfil; los cambios de lanzador requieren revisar una nueva propuesta. La salida se
actualiza por lotes y conserva los últimos 64 KiB por ejecución en memoria. Se pueden
mantener hasta ocho aplicaciones simultáneas; cerrar el panel no las detiene.

Las ejecuciones siguen activas al cambiar de proyecto. El indicador junto a
Configuración muestra el total activo y la cantidad por proyecto; al elegir uno,
se activa su carpeta y se abre su panel de Ejecuciones. Todas se detienen al cerrar Merlin Code. En Windows
10 o posterior se utiliza un Job Object por ejecución, sin heredar su handle, para
terminar también los procesos hijos incluso si Merlin finaliza inesperadamente.
En macOS/Linux se gestionan grupos de procesos para el cierre normal; esto no ofrece
la misma garantía ante una terminación forzada de Merlin.

Los programas se resuelven mediante el `PATH` del equipo. En Windows también se
respeta `PATHEXT` y se admiten ejecutables y lanzadores `.cmd`/`.bat`, sin depender
de la estructura interna de instalación de npm, pnpm u otros runtimes. Estos
lanzadores conservan la captura de salida y el control de sus procesos hijos.

El gestor no acepta comandos libres ni instala dependencias. Comprueba nuevamente
la propuesta antes de iniciar y requiere una nueva si cambió su configuración.
**Los runtimes externos ejecutan código del proyecto con los permisos del usuario;
la validación de comandos y carpetas no constituye un aislamiento del sistema operativo.**
Las operaciones de archivos de Merlin y el servidor HTML integrado conservan la
validación del workspace y los handles `os.Root`.

## Guía de Instalación desde Cero

Esta guía está redactada paso a paso para personas que **nunca han utilizado Wails** y que **aún no tienen instalado Golang** en su computadora.

### Paso 1: Instalar Golang (Go)

Go es el lenguaje de programación compilado de alto rendimiento que impulsa el backend de Merlin Code.

1. **Descarga el instalador:**
   - Ve al sitio oficial: [https://go.dev/dl/](https://go.dev/dl/)
   - Descarga la versión estable recomendada para tu sistema operativo (por ejemplo, el instalador `.msi` para Windows x64).
2. **Ejecuta el instalador:**
   - Sigue las instrucciones del instalador (deja las rutas por defecto, por ejemplo `C:\Program Files\Go`).
3. **Verifica la instalación:**
   - Abre una terminal nueva (PowerShell, CMD o Terminal de macOS/Linux) y escribe:
     ```bash
     go version
     ```
   - Deberías ver una salida similar a: `go version go1.21.x ...`

---

### Paso 2: Instalar Node.js y npm

Node.js es necesario para compilar el frontend basado en React, TypeScript y Vite.

1. **Descarga el instalador:**
   - Ingresa a [https://nodejs.org/](https://nodejs.org/) y descarga la versión **LTS** (Long Term Support).
2. **Instala Node.js:**
   - Ejecuta el instalador y acepta la configuración por defecto.
3. **Verifica la instalación:**
   - En una terminal nueva, ejecuta:
     ```bash
     node -v
     npm -v
     ```

---

### Paso 3: Requisitos del Sistema Operativo

- **Windows:**
  - **Microsoft WebView2 Runtime:** Viene instalado por defecto en Windows 10 y Windows 11. Si estás en una versión desactualizada o Windows Server, puedes descargarlo de [Microsoft WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/).
  - **Compilador C/C++ (Opcional pero recomendado):** Wails en Windows no requiere CGo para la mayoría de operaciones, pero si se requiere compilar dependencias CGo, se puede instalar [MinGW-w64](https://www.mingw-w64.org/) o las Build Tools de Visual Studio.
- **macOS:**
  - Instala Xcode Command Line Tools ejecutando: `xcode-select --install`
- **Linux:**
  - Instala las dependencias de WebKit2GTK y gcc según tu distribución (ej. `sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev build-essential` en Ubuntu/Debian).

---

### Paso 4: Instalar Wails CLI v2

[Wails](https://wails.io/) es el framework que conecta el backend de Go con la interfaz web de React en un ejecutable de escritorio nativo.

1. Abre tu terminal y ejecuta el siguiente comando:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
   ```

2. **Asegúrate de que la ruta de binarios de Go esté en tu variable de entorno PATH:**
   - En **Windows**, la ruta suele ser: `%USERPROFILE%\go\bin` (por ejemplo `C:\Users\<TuUsuario>\go\bin`).
   - En **macOS / Linux**, la ruta suele ser: `~/go/bin`.

3. **Verifica tu entorno con el diagnóstico oficial de Wails:**
   ```bash
   wails doctor
   ```
   *Wails analizará tu sistema y verificará que Go, Node.js, npm y el motor de renderizado estén listos con marcas verdes de verificación.*

---

## Ejecución y Desarrollo

Una vez que tengas Go, Node.js y Wails instalados:

### 1. Clonar el repositorio
```bash
git clone https://github.com/JoseGaldamez/merlincode.git
cd merlincode
```

### 2. Instalar las dependencias del frontend
```bash
cd frontend
npm ci
cd ..
```

### 3. Iniciar la aplicación en modo desarrollo
Desde la raíz del proyecto, ejecuta:
```bash
wails dev
```

¿Qué hace `wails dev`?
- Inicia el servidor de desarrollo de Vite con **Hot Reload instantáneo** para el frontend.
- Compila el código de Go en segundo plano.
- Genera automáticamente los bindings y tipos TypeScript en `frontend/wailsjs/` cuando modificas métodos en `app.go`.
- Abre la ventana de escritorio de Merlin Code. Si modificas cualquier archivo del frontend o del backend, la aplicación se actualizará automáticamente.

### 4. Levantar el proyecto desde VS Code (Run and Debug)

El repositorio incluye configuración lista en `.vscode/launch.json` y `.vscode/tasks.json` para iniciar `wails dev` directamente desde la pestaña **Run and Debug** (`Ctrl+Shift+D`), seleccionando **"Wails: Run (wails dev)"** y presionando `F5`.

Ambos archivos usan la variable `${env:USERPROFILE}` en la ruta del PATH:

```jsonc
"Path": "${env:ProgramFiles}\\Go\\bin;${env:USERPROFILE}\\go\\bin;${env:Path}"
```

`${env:ProgramFiles}` y `${env:USERPROFILE}` son variables de entorno de Windows que VS Code resuelve automáticamente al ejecutar la tarea: la primera apunta a la carpeta de instalación de programas del sistema, y la segunda a la carpeta personal del usuario que la ejecuta. Se usan en vez de escribir una ruta absoluta (unidad de disco y carpetas específicas) para que la configuración funcione igual para cualquier persona que clone el repositorio, sin exponer ni depender de la ruta local de quien la creó. Esto es necesario porque Go se instala por defecto dentro de `%ProgramFiles%\Go\bin`, y el binario `wails.exe` (instalado con `go install`) vive dentro de `%USERPROFILE%\go\bin`; ambas carpetas deben estar en el PATH para que los comandos `go` y `wails` se reconozcan dentro de la terminal que VS Code abre al ejecutar la configuración.

---

## Claves de API de Proveedores de IA

Merlin Code se conecta a cuatro proveedores de IA (Anthropic, OpenAI, Google y DeepSeek) usando **tu propia clave de API**, configurable desde la pantalla de Configuración de la aplicación. Es importante entender qué es una clave de API y cómo funciona antes de usarla:

- **¿Qué es una API Key?** Es una credencial secreta, única para tu cuenta, que identifica y autoriza tus peticiones ante el proveedor de IA. Funciona como una contraseña: cualquiera que la obtenga puede usarla para generar consumo a tu nombre, así que nunca debe compartirse ni subirse a un repositorio público.
- **Las API Keys son de pago.** Cada proveedor cobra por el uso real del modelo (tokens de entrada y salida), normalmente mediante saldo prepago o facturación mensual asociada a tu cuenta en la plataforma del proveedor. Merlin Code **no cobra nada por su uso ni actúa como intermediario de facturación**: el consumo se factura directamente entre tú y el proveedor (Anthropic, OpenAI, Google o DeepSeek), según sus propias tarifas por modelo.
- **Dónde obtener una clave:**
  - Anthropic: [console.anthropic.com](https://console.anthropic.com/)
  - OpenAI: [platform.openai.com](https://platform.openai.com/)
  - Google AI (Gemini): [aistudio.google.com](https://aistudio.google.com/)
  - DeepSeek: [platform.deepseek.com](https://platform.deepseek.com/)
- **Cómo se guarda:** al validar una clave desde la app, Merlin Code la verifica en vivo contra el proveedor y, solo si es válida, la guarda en el **llavero nativo y cifrado del sistema operativo** (Credential Manager en Windows, Keychain en macOS, Secret Service/libsecret en Linux) en vez de en texto plano. El archivo `%APPDATA%/merlincode/ai_providers.json` solo almacena ID del proveedor, estado y fecha de verificación y modelos disponibles; nunca claves, claves enmascaradas, saldos ni costos. La clave tampoco se envía a servidores de Merlin Code ni a terceros distintos del proveedor correspondiente.
- **Migración de versiones anteriores:** los archivos heredados que contenían claves se migran al llavero únicamente después de verificar la escritura y se reescriben de forma atómica. Esta compatibilidad se mantendrá hasta el **31 de marzo de 2027**. Si usaste Google AI con una versión anterior y recibiste errores de red, rota la clave desde Google AI Studio: versiones antiguas pudieron incluirla en URLs.
- **Admin Key (opcional):** algunos proveedores (por ejemplo Anthropic) requieren una clave adicional de administrador, a nivel de organización, únicamente para poder consultar el consumo y costo real de la cuenta. Es completamente opcional: sin ella, la app sigue funcionando con la clave normal y muestra una estimación local del consumo en lugar del dato exacto de facturación.
- **Probar Conexión:** la sección de prueba de mensaje en Configuración consume tokens reales de tu cuenta (usando tu clave estándar, nunca la Admin Key), por lo que también genera un costo mínimo asociado a tu plan con el proveedor.

---

## Compilación para Producción

Para generar el ejecutable binario redistribuible e independiente (listo para distribuir sin necesidad de tener Go ni Node instalados en la máquina final):

```bash
wails build
```

El binario optimizado y empaquetado se generará en la carpeta:
- **Windows:** `build/bin/merlincode.exe`
- **macOS:** `build/bin/merlincode.app`
- **Linux:** `build/bin/merlincode`

---

## Arquitectura del Proyecto

El backend es un **monolito modular**, organizado por responsabilidad. Los servicios
mantienen su estado y sus reglas; `internal/app` conecta esos servicios con Wails.
Las interfaces se definen donde se consumen y se utilizan en los límites que necesitan
sustitución, como el proveedor de chat y la persistencia del agente.

```text
merlincode/
├── main.go                         # Arranque de Wails, ventana y assets embebidos
├── internal/
│   ├── app/                        # Fachada pública, ciclo de vida y adaptador de eventos Wails
│   │   ├── app.go                  # Composición de servicios y cierre de recursos
│   │   ├── agent.go                # Puente del agente a eventos y aprobaciones de la UI
│   │   └── chat.go, execution.go…   # Métodos públicos agrupados por funcionalidad
│   ├── agent/                      # Coordinación de turnos independiente de Wails
│   │   ├── service.go              # Contratos, configuración y bucle del agente
│   │   ├── approvals.go            # Aprobaciones pendientes, respuesta y cancelación
│   │   ├── tool_execution.go        # Ejecución de herramientas y eventos de resultados
│   │   ├── tools.go                # Registro y adaptadores de herramientas del workspace
│   │   ├── run_tool.go             # Propuestas de ejecución; no inicia procesos
│   │   └── completion.go           # Persistencia y eventos de fin, error o cancelación
│   ├── ai/                         # Proveedores y gestión segura de sus credenciales
│   │   ├── service.go              # Estado compartido e inicialización
│   │   ├── metadata.go             # Persistencia privada y migración de metadatos
│   │   ├── credentials.go          # Operaciones con llavero y estados sin secretos
│   │   ├── provider_operations.go   # Prueba de conexión y consulta de consumo
│   │   ├── chat.go                 # Validación y streaming con límites
│   │   ├── catalog.go, configs/     # Modelos y configuración declarativa
│   │   ├── providers/              # Adaptadores Anthropic, OpenAI, Google y DeepSeek
│   │   └── transport/              # HTTP, límites y sanitización de transporte
│   ├── runner/                     # Detección y supervisión de aplicaciones
│   │   ├── detect.go               # Validación común, huellas y selección de perfiles
│   │   ├── detect_javascript.go    # Scripts y gestores npm/pnpm/yarn/bun
│   │   ├── detect_python.go        # Entradas, frameworks y gestores Python
│   │   ├── detect_go.go            # Aplicaciones Go con o sin módulo
│   │   ├── service.go              # Ciclo de vida, salida y eventos de ejecuciones
│   │   ├── runtime.go              # Resolución de programas y entorno
│   │   └── process_*.go, launch_windows.go # Control de procesos según el SO
│   ├── workspace/                  # Proyecto activo, archivos y confinamiento con os.Root
│   ├── session/                    # Sesiones y mensajes en SQLite
│   ├── window/                     # Persistencia de ventana y paneles
│   ├── domain/                     # Modelos compartidos y errores sin dependencias de servicios
│   └── platform/                   # Adaptadores del sistema: config, keyring y desktop
└── frontend/
    ├── wailsjs/                    # Bindings generados; no se editan manualmente
    └── src/features/               # chat, executions, artifacts, projects, sessions, layout, settings
```

Los archivos `*_test.go` permanecen junto al paquete que verifican. Las pruebas del
agente utilizan proveedores, persistencia y callbacks inyectados; no necesitan un
runtime Wails. Las pruebas de `app` cubren el puente nativo y la integración entre servicios.

### Flujo de datos y límites entre módulos

```text
React ── bindings Wails ──► app
                            ├──► agent ──► ai ──► providers ──► transport
                            │      ├──► workspace (herramientas confinadas)
                            │      └──► session   (persistencia de respuestas)
                            ├──► runner ──► workspace (raíces validadas)
                            ├──► workspace
                            ├──► session
                            └──► window

agent ── callbacks ──► app ── eventos Wails ──► React
```

- `app` conserva el ciclo de vida nativo, la reserva y cancelación de streams y los
  métodos públicos usados por los bindings. Delega la coordinación del agente a `agent.Service`.
- `agent` consume el contrato de chat, el almacén de mensajes y callbacks para emitir
  eventos, notificar escrituras y presentar errores. No importa Wails ni implementaciones de proveedores.
- `ai` conoce los proveedores y el llavero; no conoce el workspace ni ejecuta herramientas.
  Sus archivos se separan por responsabilidad dentro del mismo servicio y conservan los mismos mutexes.
- `runner` mantiene sus perfiles por ecosistema y comparte la lectura validada y la
  huella de los archivos consultados. No acepta comandos arbitrarios del modelo.
- `workspace` centraliza el confinamiento de rutas y los handles `os.Root`. Las
  herramientas y el runner reutilizan esas garantías.
- Preparar una propuesta no inicia procesos. Play sigue siendo una acción explícita
  del usuario. Las escrituras siguen esperando la aprobación de su tarjeta.

Esta distribución permite extender un proveedor, una herramienta o un perfil de
lanzamiento en su módulo correspondiente, sin añadir capas a los servicios pequeños.

---

## Pruebas Automatizadas

El backend incluye una suite de pruebas unitarias para garantizar la confiabilidad del servicio de ventana y la seguridad del sandboxing del espacio de trabajo:

```bash
# Ejecutar todas las pruebas del backend en Go
go test ./...

# Verificar tipado y compilar el frontend
cd frontend && npm run build
```

---

## Creador

<table align="center">
  <tr>
    <td align="center">
      <a href="https://github.com/JoseGaldamez">
        <img src="https://github.com/JoseGaldamez.png" width="120px;" alt="José Galdámez" style="border-radius: 50%;" />
        <br />
        <sub><b>José Galdámez</b></sub>
      </a>
      <br />
      <a href="https://josegaldamez.dev/">josegaldamez.dev</a> •
      <a href="https://github.com/JoseGaldamez">@JoseGaldamez</a>
    </td>
  </tr>
</table>

---

## Contribución

¡Las contribuciones son bienvenidas! Si deseas colaborar con correcciones de errores, mejoras de rendimiento o nuevas funcionalidades, por favor consulta la [Guía de Contribución](CONTRIBUTING.md) antes de enviar un Pull Request.

---

## Términos de Uso y Privacidad

Para conocer los principios de privacidad local-first, términos de uso del software y directrices de seguridad, consulta [Términos de Uso](TERMS.md).

---

## Licencia

Este proyecto está distribuido bajo la licencia **MIT**, la misma licencia con la que nació **OpenCode**. Consulta el archivo [LICENSE](LICENSE) para obtener más información.

## Actualización de credenciales antiguas

Si utilizaste Google con una versión anterior que enviaba la API key en la URL,
revoca esa clave en Google AI Studio y crea otra, especialmente si hubo errores de red
o compartiste registros de diagnóstico. Eliminar la copia local no revoca la clave
en Google. Introduce la nueva clave en Ajustes → Proveedores de IA.

La migración de credenciales JSON heredadas se mantiene hasta el 31 de marzo de 2027.
Solo elimina las claves del JSON después de confirmar su almacenamiento en el llavero.
