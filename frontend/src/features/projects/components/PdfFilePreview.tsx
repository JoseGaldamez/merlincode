import { useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import { AnnotationMode, getDocument, GlobalWorkerOptions } from 'pdfjs-dist';
import type { PDFDocumentProxy, PDFPageProxy, RenderTask } from 'pdfjs-dist';

const assets = import.meta.env.BASE_URL + 'pdfjs/';
GlobalWorkerOptions.workerSrc = assets + 'pdf.worker.min.mjs';

type ScrollRoot = RefObject<HTMLDivElement>;

function PdfPage({ pdf, number, scrollRoot }: { pdf: PDFDocumentProxy; number: number; scrollRoot: ScrollRoot }) {
  const pageRef = useRef<HTMLElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [visible, setVisible] = useState(false);
  const [ratio, setRatio] = useState(0.77);
  const [error, setError] = useState(false);

  useEffect(() => {
    const element = pageRef.current!;
    const observer = new IntersectionObserver(([entry]) => setVisible(entry.isIntersecting), {
      root: scrollRoot.current,
      rootMargin: '700px 0px',
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, [scrollRoot]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!visible || !canvas || error) return;

    let disposed = false;
    let page: PDFPageProxy | undefined;
    let task: RenderTask | undefined;
    pdf.getPage(number).then((nextPage) => {
      if (disposed) return;
      page = nextPage;
      const viewport = page.getViewport({ scale: 1 });
      setRatio(viewport.width / viewport.height);
      // Bound canvas allocation even if the PDF declares an unusually large page.
      const maxSide = Math.min(2048, Math.max(1, canvas.clientWidth) * (window.devicePixelRatio || 1));
      const scaledViewport = page.getViewport({ scale: maxSide / Math.max(viewport.width, viewport.height) });
      canvas.width = Math.max(1, Math.ceil(scaledViewport.width));
      canvas.height = Math.max(1, Math.ceil(scaledViewport.height));
      task = page.render({ canvas, viewport: scaledViewport, annotationMode: AnnotationMode.DISABLE });
      return task.promise;
    }).catch(() => {
      if (!disposed) setError(true);
    });

    return () => {
      disposed = true;
      task?.cancel();
      page?.cleanup();
    };
  }, [error, number, pdf, visible]);

  return (
    <figure ref={pageRef} className="file-preview-pdf-page">
      <div className="file-preview-pdf-sheet" style={{ aspectRatio: ratio }}>
        {error
          ? <p className="file-preview-pdf-error" role="status">No se pudo mostrar esta página.</p>
          : visible && <canvas ref={canvasRef} aria-label={'Página ' + number} role="img" />}
      </div>
      <figcaption>Página {number} de {pdf.numPages}</figcaption>
    </figure>
  );
}

export default function PdfFilePreview({ data, scrollRoot }: { data: Uint8Array<ArrayBuffer>; scrollRoot: ScrollRoot }) {
  const [pdf, setPdf] = useState<PDFDocumentProxy>();
  const [error, setError] = useState('');

  useEffect(() => {
    let disposed = false;
    const task = getDocument({
      data: data.slice(),
      cMapUrl: assets + 'cmaps/',
      standardFontDataUrl: assets + 'standard_fonts/',
      wasmUrl: assets + 'wasm/',
      iccUrl: assets + 'iccs/',
      isEvalSupported: false,
      enableXfa: false,
      maxImageSize: 16_000_000,
    });
    task.promise.then((nextPdf) => {
      if (!disposed) setPdf(nextPdf);
    }).catch((reason: unknown) => {
      if (disposed) return;
      const passwordRequired = reason instanceof Error && reason.name === 'PasswordException';
      setError(passwordRequired ? 'Este PDF está protegido con contraseña.' : 'No se pudo mostrar el PDF.');
    });
    return () => {
      disposed = true;
      void task.destroy().catch(() => {});
    };
  }, [data]);

  if (error) return <p className="file-preview-status" role="alert">{error}</p>;
  if (!pdf) return <p className="file-preview-status" role="status">Cargando PDF…</p>;

  return (
    <div className="file-preview-pdf">
      {Array.from({ length: pdf.numPages }, (_, index) => (
        <PdfPage key={index} pdf={pdf} number={index + 1} scrollRoot={scrollRoot} />
      ))}
    </div>
  );
}
