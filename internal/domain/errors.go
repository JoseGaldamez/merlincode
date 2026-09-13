package domain

import "errors"

var (
	// ErrNoActiveProject indica que no hay ninguna carpeta o proyecto activo seleccionado
	ErrNoActiveProject = errors.New("no hay ningún proyecto activo seleccionado")

	// ErrAccessDenied indica un intento de acceder o escribir fuera del directorio del proyecto
	ErrAccessDenied = errors.New("acceso denegado: solo se permite acceder a archivos dentro de la carpeta del proyecto")

	// ErrFolderNotExist indica que la ruta especificada no existe en el sistema de archivos
	ErrFolderNotExist = errors.New("la carpeta no existe")

	// ErrNotADirectory indica que la ruta especificada no corresponde a un directorio
	ErrNotADirectory = errors.New("la ruta especificada no es una carpeta")

	// ErrWritePermissionDenied indica que no se tienen permisos de escritura en la carpeta
	ErrWritePermissionDenied = errors.New("permiso de escritura denegado en la carpeta")

	// ErrRuntimeNotInitialized indica que el contexto del runtime de Wails no está disponible
	ErrRuntimeNotInitialized = errors.New("runtime context not initialized")

	// ErrNoTargetPath indica que no se especificó ruta ni existe proyecto activo para la acción
	ErrNoTargetPath = errors.New("no hay ningún proyecto activo ni ruta especificada")

	// ErrUnknownAIProvider indica que el proveedor de IA solicitado no está registrado
	ErrUnknownAIProvider = errors.New("proveedor de inteligencia artificial no reconocido")

	// ErrAdminKeyNotSupported indica que el proveedor no requiere ni admite una Admin API Key
	ErrAdminKeyNotSupported = errors.New("este proveedor no admite una clave de administrador para consultar uso real")
)
