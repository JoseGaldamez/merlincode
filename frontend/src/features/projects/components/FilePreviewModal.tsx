import { lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import { ReadProjectFilePreview } from '../../../../wailsjs/go/app/App';
import { IconClose, IconMinus, IconPlus } from '../../../components/Icons';
import { renderSafeMarkdown } from '../../chat/safeMarkdown';
import { decodeFilePreview, fileNameFromPath } from '../filePreview';
import type { FilePreview } from '../filePreview';
import { FileIcon } from './FileIcon';
import './filePreview.css';

const PdfFilePreview = lazy(() => import('./PdfFilePreview'));
const TextFilePreview = lazy(() => import('./TextFilePreview'));
const loadingContent = <p className="file-preview-status" role="status">Cargando vista previa…</p>;
const MIN_ZOOM = 25;
const MAX_ZOOM = 400;

interface FilePreviewModalProps {
  project: string;
  path: string;
  onClose: () => void;
}

export function FilePreviewModal({ project, path, onClose }: FilePreviewModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const [preview, setPreview] = useState<FilePreview>();
  const [error, setError] = useState('');
  const [zoom, setZoom] = useState(100);
  const [imageSize, setImageSize] = useState<{ width: number; height: number }>();
  const name = fileNameFromPath(path);
  const showZoom = preview?.kind === 'image' && imageSize && !error;
  const markdown = useMemo(() => preview?.kind === 'markdown'
    ? renderSafeMarkdown(preview.text, { mode: 'document' })
    : '', [preview]);

  useEffect(() => {
    const dialog = dialogRef.current!;
    dialog.showModal();
    closeRef.current?.focus();
    return () => { if (dialog.open) dialog.close(); };
  }, []);

  useEffect(() => {
    let disposed = false;
    ReadProjectFilePreview(project, path)
      .then((encoded) => {
        if (!disposed) setPreview(decodeFilePreview(path, encoded));
      })
      .catch((reason: unknown) => {
        if (disposed) return;
        setError(String(reason).includes('4 MiB')
          ? 'El archivo supera el límite de 4 MiB.'
          : 'No se pudo abrir el archivo. Comprueba que exista en el proyecto activo.');
      });
    return () => { disposed = true; };
  }, [path, project]);

  const changeZoom = (step: number) => {
    setZoom((current) => Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, current + step)));
  };

  return (
    <dialog
      ref={dialogRef}
      className="file-preview"
      aria-labelledby="file-preview-title"
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <header className="file-preview-header">
        <div className="flex items-center gap-3 min-w-0" title={path}>
          <FileIcon name={name} />
          <div className="grid min-w-0 gap-0.5 font-mono">
            <h2 id="file-preview-title" className="truncate text-sm font-semibold text-content-headline">{name}</h2>
            {path !== name && <p className="truncate text-[11px] text-content-muted">{path}</p>}
          </div>
        </div>
        <div className="flex items-center gap-3 shrink-0">
          {showZoom && (
            <div className="file-preview-zoom" role="group" aria-label="Zoom de la imagen">
              <button type="button" aria-label="Alejar" title="Alejar" disabled={zoom === MIN_ZOOM} onClick={() => changeZoom(-25)}>
                <IconMinus size={15} />
              </button>
              <button type="button" aria-label={zoom + ' %, ajustar a la ventana'} title="Ajustar a la ventana" onClick={() => setZoom(100)}>
                {zoom} %
              </button>
              <button type="button" aria-label="Acercar" title="Acercar" disabled={zoom === MAX_ZOOM} onClick={() => changeZoom(25)}>
                <IconPlus size={15} />
              </button>
            </div>
          )}
          <button
            ref={closeRef}
            type="button"
            className="file-preview-close"
            onClick={onClose}
            aria-label="Cerrar vista previa"
            title="Cerrar (Esc)"
          >
            <IconClose size={17} />
          </button>
        </div>
      </header>
      <div ref={bodyRef} className="file-preview-body" tabIndex={0} aria-label="Contenido del archivo">
        {error ? <p className="file-preview-status" role="alert">{error}</p>
          : !preview ? loadingContent
          : preview.kind === 'unsupported' ? (
            <p className="file-preview-status" role="status">Este formato de archivo no tiene vista previa.</p>
          ) : preview.kind === 'image' ? (
            <div className="file-preview-image">
              <img
                src={preview.src}
                alt={name}
                draggable={false}
                onLoad={({ currentTarget }) => setImageSize({ width: currentTarget.naturalWidth, height: currentTarget.naturalHeight })}
                onError={() => setError('No se pudo mostrar la imagen.')}
                style={imageSize ? {
                  '--image-width': imageSize.width + 'px',
                  '--image-ratio': imageSize.width / imageSize.height,
                  '--image-zoom': zoom / 100,
                } as CSSProperties : { visibility: 'hidden' }}
              />
            </div>
          ) : preview.kind === 'markdown' ? (
            markdown.trim()
              ? <article className="markdown-body file-preview-markdown" dangerouslySetInnerHTML={{ __html: markdown }} />
              : <p className="file-preview-status" role="status">El archivo está vacío.</p>
          ) : (
            <Suspense fallback={loadingContent}>
              {preview.kind === 'pdf'
                ? <PdfFilePreview data={preview.data} scrollRoot={bodyRef} />
                : <TextFilePreview name={name} text={preview.text} />}
            </Suspense>
          )}
      </div>
    </dialog>
  );
}
