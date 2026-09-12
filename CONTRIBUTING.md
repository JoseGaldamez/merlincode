# Guía de Contribución a Merlin Code 🧙‍♂️

¡Gracias por tu interés en contribuir a **Merlin Code**! Este proyecto es de código abierto y agradece enormemente el tiempo y dedicación de la comunidad para hacer de este asistente una herramienta de desarrollo cada vez más rápida, segura y agradable.

---

## 📋 Tabla de Contenidos

1. [Código de Conducta](#-código-de-conducta)
2. [Tipos de Contribuciones](#-tipos-de-contribuciones)
3. [Entorno de Desarrollo Local](#-entorno-de-desarrollo-local)
4. [Flujo de Trabajo con Git y GitHub](#-flujo-de-trabajo-con-git-y-github)
5. [Estándares y Buenas Prácticas](#-estándares-y-buenas-prácticas)
   - [Backend (Go & Wails)](#backend-go--wails)
   - [Frontend (React, TypeScript & Tailwind)](#frontend-react-typescript--tailwind)
6. [Pruebas Automatizadas](#-pruebas-automatizadas)
7. [Reporte de Errores (Bugs)](#-reporte-de-errores-bugs)
8. [Sugerencia de Nuevas Funcionalidades](#-sugerencia-de-nuevas-funcionalidades)

---

## 🤝 Código de Conducta

Esperamos que todos los participantes y colaboradores traten a los demás con respeto, amabilidad y profesionalismo. No se tolerará ningún tipo de acoso, lenguaje despectivo o comportamiento discriminatorio en issues, pull requests o canales de discusión.

---

## 💡 Tipos de Contribuciones

Las siguientes contribuciones son especialmente valoradas y bienvenidas:

- 🐛 **Corrección de errores (Bug fixes):** Solución de fallos en el explorador de archivos, sincronización de la ventana, o renderizado de mensajes.
- ⚡ **Mejoras de rendimiento:** Optimización en la lectura del árbol de directorios, reducción de re-renders en React y uso de memoria.
- 🛡️ **Seguridad del Sandboxing:** Refuerzo de validaciones contra ataques de Path Traversal o fugas de permisos en el espacio de trabajo.
- 🎨 **Experiencia de Usuario (UI/UX):** Mejoras ergonómicas en la redimensión de paneles, accesibilidad, y temas visuales.
- 📝 **Documentación:** Correcciones ortográficas, mejoras en guías de instalación y traducción a nuevos idiomas.

> [!NOTE]
> Para cambios estructurales grandes o nuevas características centrales de producto, por favor abre primero un **Issue** para debatir la propuesta con el equipo antes de comenzar a programar.

---

## 💻 Entorno de Desarrollo Local

Asegúrate de contar con los requisitos instalados tal como se detalla en el [README.md](README.md):
- **Go 1.21+**
- **Node.js (versión LTS) & npm**
- **Wails CLI v2:** `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

Verifica tu entorno con:
```bash
wails doctor
```

### Iniciar en Modo Desarrollo

```bash
# 1. Instalar dependencias de frontend
cd frontend
npm install
cd ..

# 2. Iniciar Wails en modo de desarrollo con Hot Reload
wails dev
```

Cualquier cambio en archivos `.go` o en la carpeta `frontend/src` recargará automáticamente la aplicación.

---

## 🔄 Flujo de Trabajo con Git y GitHub

1. **Haz un Fork del repositorio:**  
   Haz clic en el botón **Fork** en [https://github.com/JoseGaldamez/merlincode](https://github.com/JoseGaldamez/merlincode).

2. **Clona tu fork localmente:**
   ```bash
   git clone https://github.com/<tu-usuario>/merlincode.git
   cd merlincode
   ```

3. **Crea una rama para tu cambio (Feature Branch):**
   ```bash
   git checkout -b feature/mi-nueva-mejora
   # o para correcciones:
   git checkout -b fix/correccion-arbol-archivos
   ```

4. **Realiza tus cambios y confirma con mensajes convencionales (Conventional Commits):**
   Utilizamos la convención estándar de commits:
   - `feat: añade soporte para colapsar carpetas con doble clic`
   - `fix: previene error al seleccionar ruta con caracteres especiales`
   - `docs: actualiza instrucciones de instalación para Linux`
   - `refactor: modulariza componentes del panel lateral`
   - `test: agrega pruebas unitarias para ValidatePathInProject`

5. **Envía tus cambios a tu repositorio remoto:**
   ```bash
   git push origin feature/mi-nueva-mejora
   ```

6. **Abre un Pull Request:**  
   Ve a GitHub y abre un Pull Request hacia la rama `main` del repositorio oficial. Describe claramente el cambio, qué problema resuelve y cómo verificarlo.

---

## 📐 Estándares y Buenas Prácticas

### Backend (Go & Wails)
- **Arquitectura Limpia:**
  - Los tipos de datos puros y errores viven en `internal/domain`.
  - La lógica de negocio vive en `internal/workspace` y `internal/window`.
  - `app.go` debe permanecer como una fachada delgada (Facade) exponiendo métodos al runtime de Wails.
- **Seguridad en primer lugar:**  
  Cualquier operación que interactúe con el sistema de archivos del usuario **DEBE** pasar por `workspace.ValidatePathInProject` para asegurar que no se acceda fuera de la carpeta autorizada.
- **Concurrencia segura:** Utiliza `sync.RWMutex` cuando複数のgoroutines puedan acceder o mutar el estado persistente.
- **Formateo:** Ejecuta `go fmt ./...` antes de hacer commit.

### Frontend (React, TypeScript & Tailwind)
- **Modularidad basada en características (Feature-Driven):**  
  Coloca los componentes dentro de su respectivo módulo en `src/features/` (`chat/`, `projects/`, `layout/`, etc.).
- **TypeScript Estricto:** Evita el uso de `any`. Define tipos claros en `src/types.ts` o dentro de la característica.
- **Iconos:** Reutiliza o añade nuevos iconos vectoriales dentro de `src/components/Icons.tsx` en formato SVG nativo para mantener el paquete ligero.
- **Estilos:** Emplea utilidades de Tailwind CSS acordes a la paleta establecida en `tailwind.config.js` (`petrol-dark`, `petrol-card`, etc.).

---

## 🧪 Pruebas Automatizadas

Antes de enviar un Pull Request, confirma que todas las pruebas pasen y que el proyecto compile limpiamente:

```bash
# 1. Pruebas unitarias de Go
go test ./...

# 2. Verificación de TypeScript y compilación de producción del frontend
cd frontend
npm run build
cd ..
```

---

## 🐞 Reporte de Errores (Bugs)

Si encuentras un error o comportamiento inesperado:
1. Revisa los [Issues existentes](https://github.com/JoseGaldamez/merlincode/issues) para verificar si ya ha sido reportado.
2. Si no existe, abre un nuevo Issue indicando:
   - Sistema operativo y versión (ej. Windows 11 23H2, Ubuntu 22.04, macOS Sonoma).
   - Pasos detallados para reproducir el fallo.
   - Capturas de pantalla o logs de error si corresponde.
   - Comportamiento esperado vs. comportamiento observado.

---

## 💡 Sugerencia de Nuevas Funcionalidades

¿Tienes una idea para hacer a Merlin Code aún mejor?
1. Abre un Issue con la etiqueta `enhancement`.
2. Explica el caso de uso y por qué beneficiaría a los desarrolladores.
3. Comparte mockups o ejemplos conceptuales si es posible.

¡Gracias por ser parte del crecimiento de **Merlin Code**! 🚀
