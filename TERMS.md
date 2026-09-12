# Términos de Uso de Merlin Code 📄

*Última actualización: 12 de septiembre de 2026*

Bienvenido a **Merlin Code**. Al descargar, compilar, instalar o utilizar esta aplicación de escritorio, aceptas quedar vinculado por los presentes Términos de Uso. Si no estás de acuerdo con alguna parte de estos términos, por favor no utilices la aplicación.

---

## 1. Naturaleza del Software y Licencia

Merlin Code es un software de código abierto desarrollado por **José Galdámez** y colaboradores de la comunidad, distribuido bajo la licencia permisiva **MIT License** (la misma licencia de lanzamiento original de proyectos como OpenCode). 

Se te otorga el derecho de utilizar, copiar, modificar, fusionar, publicar, distribuir, sublicenciar y/o vender copias del Software, sujeto a las condiciones expresadas en el archivo [LICENSE](LICENSE).

---

## 2. Privacidad y Filosofía Local-First

1. **Procesamiento Local:**  
   Merlin Code opera como una herramienta orientada a la privacidad y al control del usuario (*Local-First*). Tus archivos, árboles de directorios, configuraciones de interfaz y sesiones se almacenan de manera local en tu propio disco duro.
2. **Espacio de Trabajo Confinado (Sandboxing):**  
   La aplicación sólo interactúa con la carpeta que selecciones voluntariamente a través del selector nativo del sistema operativo. Merlin Code implementa protecciones técnicas contra *Path Traversal* para evitar leer o escribir en ubicaciones fuera del proyecto activo.
3. **Sin Telemetría Invasiva:**  
   Merlin Code no recopila ni comercializa tu código fuente ni tu información personal.

---

## 3. Integración con Modelos de Inteligencia Artificial (LLMs)

1. **Responsabilidad sobre Claves de API (API Keys):**  
   Si utilizas proveedores de modelos en la nube (como OpenAI, Anthropic u otros), tú eres el único responsable de suministrar, proteger y pagar por las credenciales y cuotas de consumo asociadas a dichos servicios.
2. **Tráfico de Red con Proveedores de IA:**  
   Al interactuar con el chat del asistente, los fragmentos de código y consultas que envíes serán procesados por el proveedor de modelo que hayas configurado. Te recomendamos revisar las políticas de privacidad y términos de servicio del proveedor de IA correspondiente.
3. **Revisión del Código Generado:**  
   Los modelos de lenguaje pueden cometer imprecisiones, generar código con vulnerabilidades de seguridad o proponer dependencias desactualizadas. **Es responsabilidad absoluta del desarrollador revisar, auditar y probar todo código sugerido o generado antes de ejecutarlo o desplegarlo en producción.**

---

## 4. Permisos del Sistema de Archivos

Al abrir un directorio en Merlin Code:
- Concedes a la aplicación permiso para inspeccionar la jerarquía de archivos y leer/escribir contenido dentro de dicha carpeta con el fin de realizar tareas de pair programming.
- Merlin Code no asume responsabilidad alguna por sobrescrituras accidentales resultantes de instrucciones emitidas por el usuario o sugerencias de código aplicadas. Se recomienda encarecidamente trabajar siempre con un sistema de control de versiones como Git.

---

## 5. Exclusión de Garantías y Limitación de Responsabilidad

TAL COMO LO ESTABLECE LA LICENCIA MIT:

EL SOFTWARE SE PROPORCIONA "TAL CUAL" ("AS IS"), SIN GARANTÍA DE NINGÚN TIPO, EXPRESA O IMPLÍCITA, INCLUYENDO, PERO SIN LIMITARSE A, GARANTÍAS DE COMERCIABILIDAD, IDONEIDAD PARA UN PROPÓSITO PARTICULAR Y NO INFRACCIÓN. 

EN NINGÚN CASO LOS AUTORES O TITULARES DEL COPYRIGHT SERÁN RESPONSABLES DE NINGUNA RECLAMACIÓN, DAÑOS U OTRAS RESPONSABILIDADES, YA SEA EN UNA ACCIÓN CONTRACTUAL, EXTRACONTRACTUAL O DE OTRO TIPO, QUE SURJA DE, O EN CONEXIÓN CON EL SOFTWARE O EL USO U OTROS TRATOS EN EL SOFTWARE.

---

## 6. Modificaciones a los Términos

Dado el carácter abierto y colaborativo del proyecto, estos términos pueden actualizarse periódicamente en el repositorio público. Las versiones actualizadas entrarán en vigor a partir del momento de su publicación en el repositorio.

---

## 7. Contacto

Para dudas o consultas respecto a estos términos o al proyecto, puedes ponerte en contacto a través de:
- **Sitio Web Oficial:** [https://josegaldamez.dev/](https://josegaldamez.dev/)
- **GitHub:** [https://github.com/JoseGaldamez](https://github.com/JoseGaldamez)
