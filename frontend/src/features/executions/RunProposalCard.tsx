import React, { useState } from 'react';
import { IconChevronRight, IconFileCode, IconFolder, IconPlay } from '../../components/Icons';
import { useExecutionActions } from './ExecutionProvider';
import type { RunProposal } from './types';

export function RunProposalCard({ proposal }: { proposal: RunProposal }) {
  const { start, project } = useExecutionActions();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const wrongProject = project !== proposal.project;
  return (
    <section className="run-proposal" aria-label={`Ejecutar ${proposal.title}`}>
      <div className="run-proposal-heading">
        <span className="run-proposal-icon" aria-hidden="true"><IconFileCode size={18} /></span>
        <div className="run-proposal-description">
          <strong>{proposal.title}</strong>
          <p className="run-reason">{proposal.reason}</p>
          <div className="run-directory">
            <IconFolder size={13} />
            <span>Carpeta: <span className={proposal.directory === '.' ? undefined : 'run-directory-path'}>{proposal.directory === '.' ? 'Raíz del proyecto' : proposal.directory}</span></span>
          </div>
        </div>
      </div>
      <div className="run-command-row">
        <div className="run-command-content">
          <IconChevronRight size={15} className="run-command-prompt" />
          <code className="run-command" aria-label="Comando a ejecutar">{proposal.command}</code>
        </div>
        <button type="button" className="run-play" disabled={pending || wrongProject}
          onClick={async (event) => {
            const source = event.currentTarget.getBoundingClientRect();
            setPending(true); setError('');
            try { await start(proposal, source); } catch (err) { setError(String(err instanceof Error ? err.message : err)); }
            finally { setPending(false); }
          }} aria-label={`Play: ${proposal.command}`}>
          <IconPlay size={14} />{pending ? 'Iniciando…' : 'Ejecutar'}
        </button>
      </div>
      <div className="run-proposal-footer">
        <p className="run-guidance" role="status">{wrongProject ? 'Activa el proyecto de esta propuesta para ejecutarla.' : pending ? 'Enviando la ejecución al panel…' : 'La salida se mostrará en el panel Ejecuciones.'}</p>
        <p className="run-permissions">El código del proyecto se ejecuta con los permisos de tu usuario.</p>
      </div>
      {error && <p className="run-error" role="alert">{error}</p>}
    </section>
  );
}
