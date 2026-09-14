package domain

import "errors"

var (
	ErrFileTooLarge       = errors.New("el archivo excede el límite de 4 MiB")
	ErrChatTooLarge       = errors.New("la conversación excede el límite de 256 KiB o 200 mensajes")
	ErrInvalidChat        = errors.New("la solicitud de chat no es válida")
	ErrChatBusy           = errors.New("ya hay una generación activa; espera a que finalice o detenla")
	ErrChatOutputTooLarge = errors.New("la respuesta excede el límite de tamaño permitido")

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

	// ErrInvalidModel indica que el modelo solicitado no pertenece a los modelos permitidos para el proveedor
	ErrInvalidModel = errors.New("el modelo especificado no pertenece a la lista de modelos permitidos para este proveedor")

	// ErrMessageTooLarge indica que el mensaje de prueba excede el límite permitido de 4 KiB
	ErrMessageTooLarge = errors.New("el mensaje de prueba excede el límite máximo permitido de 4 KiB")

	// ErrRateLimited indica que se debe esperar antes de realizar otra petición de prueba
	ErrRateLimited = errors.New("por favor espera unos segundos antes de enviar otra solicitud de prueba")

	// ErrConcurrentRequestBlocked indica que ya hay una solicitud de prueba activa para el proveedor
	ErrConcurrentRequestBlocked = errors.New("ya hay una solicitud de prueba en curso para este proveedor")

	// ErrCredentialStoreUnavailable clasifica fallos del llavero sin exponer detalles del sistema operativo.
	ErrCredentialStoreUnavailable = errors.New("no se pudo acceder al almacén seguro de credenciales del sistema")

	// ErrProviderMetadataUnavailable clasifica fallos de permisos o persistencia de metadata.
	ErrProviderMetadataUnavailable = errors.New("no se pudo acceder a la configuración segura de proveedores")

	// ErrCredentialMigrationFailed indica que una configuración heredada no pudo migrarse sin riesgo.
	ErrCredentialMigrationFailed = errors.New("no se pudo migrar la configuración heredada de credenciales de forma segura")

	// ErrAIProviderOperationFailed evita exponer errores internos inesperados a la interfaz.
	ErrAIProviderOperationFailed = errors.New("no se pudo completar la operación con el proveedor de inteligencia artificial")
)
