<div align="center">

  <img src="./frontend/src/assets/images/logo.png" alt="Merlin Code Logo" width="110" height="110" />

  # Merlin Code

  **El asistente de programación en pareja e ingeniería de software asistido por IA para escritorio.**  
  *Construido con Go, Wails v2, React 18, Vite y Tailwind CSS.*

  <p>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-teal.svg?style=for-the-badge" alt="License: MIT" /></a>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" /></a>
    <a href="https://wails.io"><img src="https://img.shields.io/badge/Wails-v2-DF0000?style=for-the-badge&logo=wails&logoColor=white" alt="Wails Version" /></a>
    <a href="https://react.dev"><img src="https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React 18" /></a>
    <a href="https://vitejs.dev"><img src="https://img.shields.io/badge/Vite-3-646CFF?style=for-the-badge&logo=vite&logoColor=white" alt="Vite" /></a>
    <a href="https://tailwindcss.com"><img src="https://img.shields.io/badge/Tailwind-3-38B2AC?style=for-the-badge&logo=tailwind-css&logoColor=white" alt="Tailwind CSS" /></a>
  </p>

  <p>
    <a href="#-descripción-del-proyecto">Descripción</a> •
    <a href="#-características-principales">Características</a> •
    <a href="#-guía-de-instalación-desde-cero">Instalación desde Cero</a> •
    <a href="#-ejecución-y-desarrollo">Ejecución</a> •
    <a href="#-compilación-para-producción">Compilación</a> •
    <a href="#%EF%B8%8F-arquitectura-del-proyecto">Arquitectura</a> •
    <a href="#-creador">Creador</a> •
    <a href="#%EF%B8%8F-licencia">Licencia</a>
  </p>

</div>

---

## 📖 Descripción del Proyecto

**Merlin Code** es una aplicación de escritorio multiplataforma (Windows, macOS y Linux) diseñada para actuar como un compañero inteligente de desarrollo y pair programming. A diferencia de las herramientas web tradicionales, Merlin Code se ejecuta como una aplicación nativa ligera con bajo consumo de memoria y acceso seguro y controlado al sistema de archivos local.

La aplicación opera bajo un modelo de **espacio de trabajo confinado (sandboxing)**: únicamente tiene acceso de lectura y escritura a la carpeta que el usuario selecciona explícitamente a través del diálogo nativo del sistema operativo, garantizando la máxima seguridad y privacidad de tu código fuente.

---

## ✨ Características Principales

- 🛡️ **Espacio de Trabajo Confinado (Sandboxing Estricto):**
  - Selección de carpeta mediante el diálogo nativo del sistema operativo (`runtime.OpenDirectoryDialog`).
  - Verificación rigurosa de permisos de escritura reales antes de activar un proyecto.
  - Protección activa contra ataques de *Path Traversal* (evita lecturas o escrituras que utilicen secuencias como `../` para salir de la carpeta autorizada).
- 🌲 **Explorador de Archivos Jerárquico Integrado:**
  - Visualización en árbol del directorio del proyecto en un panel lateral interactivo.
  - Exclusión automática de directorios pesados o binarios (`.git`, `node_modules`, `dist`, `.vscode`, `build`, etc.).
  - Botón de acceso directo para abrir la carpeta del proyecto en el explorador de archivos nativo de tu sistema operativo (Windows Explorer, macOS Finder o Linux File Manager).
- 📐 **Diseño Moderno, Modular y Persistente:**
  - Ventana personalizada sin bordes (*frameless*) con controles nativos (minimizar, maximizar/restaurar, cerrar).
  - Memorización persistente del tamaño de la ventana, modo maximizado y visibilidad de los paneles laterales en `%APPDATA%/merlincode/window.json`.
  - Paneles laterales y cajón inferior colapsables y redimensionables con límites ergonómicos de altura y anchura.
- 💬 **Entorno de Chat y Pair Programming:**
  - Interfaz conversacional limpia con tema oscuro *petrol/slate*.
  - Vista de artefactos generados, historial de sesiones y configuración de modelos.
  - Mensajes de bienvenida contextuales que te guían a seleccionar un proyecto antes de interactuar con el asistente.

---

## 🚀 Guía de Instalación desde Cero

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
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
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

## 💻 Ejecución y Desarrollo

Una vez que tengas Go, Node.js y Wails instalados:

### 1. Clonar el repositorio
```bash
git clone https://github.com/JoseGaldamez/merlincode.git
cd merlincode
```

### 2. Instalar las dependencias del frontend
```bash
cd frontend
npm install
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

---

## 📦 Compilación para Producción

Para generar el ejecutable binario redistribuible e independiente (listo para distribuir sin necesidad de tener Go ni Node instalados en la máquina final):

```bash
wails build
```

El binario optimizado y empaquetado se generará en la carpeta:
- **Windows:** `build/bin/merlincode.exe`
- **macOS:** `build/bin/merlincode.app`
- **Linux:** `build/bin/merlincode`

---

## 🏗️ Arquitectura del Proyecto

El proyecto está organizado siguiendo principios de **Clean Architecture**, alta cohesión y separación de responsabilidades:

```
merlincode/
├── app.go                      # Fachada (Facade) que expone los métodos de Go a Wails y React
├── main.go                     # Punto de entrada del binario Wails y configuración de ventana
├── go.mod                      # Declaración de módulos y dependencias de Go
├── wails.json                  # Configuración del proyecto Wails
├── build/                      # Recursos de compilación, iconos y binarios generados
│   ├── appicon.png             # Icono principal de la aplicación
│   └── bin/                    # Directorio de salida del ejecutable de producción
├── internal/                   # Paquetes privados del backend en Go
│   ├── domain/                 # Entidades puras y errores de dominio
│   │   ├── errors.go           # Errores fuertemente tipados (acceso denegado, permisos, etc.)
│   │   ├── project.go          # Modelos ProjectInfo, FileItem y FileNode
│   │   └── window.go           # Modelos WindowState y PanelsState
│   ├── window/                 # Servicio de gestión y persistencia de ventana y paneles
│   │   ├── service.go          # Lectura/escritura concurrente de %APPDATA%/merlincode/window.json
│   │   └── service_test.go     # Pruebas unitarias del servicio de ventana
│   └── workspace/              # Servicio del espacio de trabajo y sistema de archivos
│       ├── security.go         # Validación de permisos y protección contra Path Traversal
│       ├── explorer.go         # Lanzador nativo multiplataforma del explorador del sistema
│       ├── service.go          # Gestión de proyecto activo, árbol jerárquico y lectura/escritura
│       └── service_test.go     # Pruebas unitarias de seguridad y manipulación de archivos
└── frontend/                   # Aplicación cliente en React 18 + TypeScript + Vite
    ├── index.html              # Punto de entrada HTML
    ├── package.json            # Dependencias de npm y scripts de Vite
    ├── tailwind.config.js      # Configuración y paleta de colores de Tailwind CSS
    ├── vite.config.ts          # Configuración del empaquetador Vite
    └── src/
        ├── App.tsx             # Componente raíz de la interfaz
        ├── main.tsx            # Inicialización de React en el DOM
        ├── types.ts            # Definición de tipos e interfaces TypeScript
        ├── components/         # Componentes atómicos transversales e iconos SVG
        │   └── Icons.tsx       # Catálogo centralizado de iconos vectoriales
        └── features/           # Arquitectura basada en características (Feature-Driven)
            ├── chat/           # ChatArea, MessageList, MessageItem, ChatInput, EmptyChatState
            ├── layout/         # Header, WindowControls, ResizableSidebar
            ├── projects/       # ProjectFileTree, ProjectSelector
            ├── artifacts/      # Visor y gestión de artefactos generados
            ├── sessions/       # Listado e historial de sesiones de conversación
            └── settings/       # Modal de configuración de modelos y preferencias
```

### Flujo de Datos y Seguridad (Sandboxing)

```
[ React 18 / Frontend ]
         │  (Wails IPC Bindings)
         ▼
[ app.go (Facade) ]
         │
         ├──► [ internal/window.Service ]    ──► Persiste ventana en %APPDATA%
         │
         └──► [ internal/workspace.Service ]
                     │
                     ├──► [ security.go ]    ──► Valida contención estricta (Anti-Path Traversal)
                     ├──► [ service.go ]     ──► Construye árbol de archivos (FileTree)
                     └──► [ explorer.go ]    ──► Invoca explorer.exe / open / xdg-open
```

---

## 🧪 Pruebas Automatizadas

El backend incluye una suite de pruebas unitarias para garantizar la confiabilidad del servicio de ventana y la seguridad del sandboxing del espacio de trabajo:

```bash
# Ejecutar todas las pruebas del backend en Go
go test ./...

# Verificar tipado y compilar el frontend
cd frontend && npm run build
```

---

## 👨‍💻 Creador

<table align="center">
  <tr>
    <td align="center">
      <a href="https://github.com/JoseGaldamez">
        <img src="https://github.com/JoseGaldamez.png" width="120px;" alt="José Galdámez" style="border-radius: 50%;" />
        <br />
        <sub><b>José Galdámez</b></sub>
      </a>
      <br />
      <a href="https://josegaldamez.dev/">🌐 josegaldamez.dev</a> •
      <a href="https://github.com/JoseGaldamez">🐙 @JoseGaldamez</a>
    </td>
  </tr>
</table>

---

## 🤝 Contribución

¡Las contribuciones son bienvenidas! Si deseas colaborar con correcciones de errores, mejoras de rendimiento o nuevas funcionalidades, por favor consulta la [Guía de Contribución](CONTRIBUTING.md) antes de enviar un Pull Request.

---

## 📄 Términos de Uso y Privacidad

Para conocer los principios de privacidad local-first, términos de uso del software y directrices de seguridad, consulta [Términos de Uso](TERMS.md).

---

## ⚖️ Licencia

Este proyecto está distribuido bajo la licencia **MIT**, la misma licencia con la que nació **OpenCode**. Consulta el archivo [LICENSE](LICENSE) para obtener más información.
