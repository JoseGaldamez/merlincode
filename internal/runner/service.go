package runner

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"merlincode/internal/domain"
	"merlincode/internal/workspace"
)

const maxLogBytes = 64 * 1024
const maxExecutions = 8

type childProcess interface {
	Stop() error
	Wait() (int, error)
}

type execution struct {
	snapshot  domain.Execution
	logBuffer []byte
	logOffset int
	logSize   int
	stop      func() error
	done      chan struct{}
	dirty     bool
	removing  bool
}

// Service owns launches across project changes until stopped or application shutdown.
// Output writers only touch bounded memory. A separate ticker batches UI events.
type Service struct {
	mu        sync.Mutex
	ws        *workspace.Service
	proposals map[string]recipe
	runs      map[string]*execution
	emit      func(domain.Execution)
	closed    bool
	cancel    context.CancelFunc
	pumpDone  chan struct{}
	revision  uint64
}

func NewService(ws *workspace.Service) *Service {
	return &Service{ws: ws, proposals: map[string]recipe{}, runs: map[string]*execution{}}
}

// StartEvents begins one bounded event pump, independent of chat turns.
func (s *Service) StartEvents(ctx context.Context, emit func(domain.Execution)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil || s.closed {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.emit = emit
	s.pumpDone = make(chan struct{})
	go func() {
		defer close(s.pumpDone)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.Lock()
				var batch []domain.Execution
				for _, run := range s.runs {
					if run.dirty {
						batch = append(batch, snapshotLocked(run))
						run.dirty = false
					}
				}
				s.mu.Unlock()
				for _, snapshot := range batch {
					emit(snapshot)
				}
			}
		}
	}()
}

// Propose inspects an explicitly selected project. It cannot start a process.
func (s *Service) Propose(req domain.RunRequest) ([]domain.RunProposal, error) {
	var result []domain.RunProposal
	err := s.ws.WithExecutionRoot(req.Directory, func(root, projectRoot *os.Root, dir, project string) error {
		recipes, err := detect(root, projectRoot, dir, project, req)
		if err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed {
			return policy("Merlin Code se está cerrando.")
		}
		if len(s.proposals)+len(recipes) > 128 {
			s.proposals = map[string]recipe{}
		}
		for _, recipe := range recipes {
			s.proposals[recipe.ID] = recipe
			result = append(result, recipe.RunProposal)
		}
		return nil
	})
	return result, err
}

// Start accepts only a backend-issued proposal ID and rechecks its evidence.
func (s *Service) Start(id string) (domain.Execution, error) {
	s.mu.Lock()
	recipe, exists := s.proposals[id]
	s.mu.Unlock()
	if !exists {
		return domain.Execution{}, policy("La propuesta ya no está disponible. Solicita una nueva propuesta de ejecución.")
	}
	var snapshot domain.Execution
	err := s.ws.WithExecutionRoot(recipe.Directory, func(root, projectRoot *os.Root, dir, project string) error {
		if project != recipe.Project {
			return policy("Activa el proyecto de esta propuesta antes de ejecutarla.")
		}
		current, err := detect(root, projectRoot, dir, project, recipe.Request)
		if err != nil {
			return err
		}
		valid := false
		for _, candidate := range current {
			if candidate.ID == id {
				valid = true
				break
			}
		}
		if !valid {
			return policy("Cambió la configuración del proyecto. Solicita una nueva propuesta antes de ejecutar.")
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed {
			return policy("Merlin Code se está cerrando.")
		}
		if previous := s.runs[id]; previous != nil && previous.removing {
			return policy("La ejecución se está eliminando. Espera a que termine antes de volver a iniciarla.")
		}
		if previous := s.runs[id]; previous != nil && isActive(previous.snapshot.Status) {
			snapshot = snapshotLocked(previous)
			return nil
		}
		active := 0
		for _, r := range s.runs {
			if isActive(r.snapshot.Status) {
				active++
			}
		}
		if active >= maxExecutions {
			return policy("Se alcanzó el límite de ocho aplicaciones simultáneas.")
		}
		// Keep a bounded number of finished entries as well as a bounded log per entry.
		if len(s.runs) >= 24 {
			for key, r := range s.runs {
				if !isActive(r.snapshot.Status) {
					delete(s.runs, key)
					break
				}
			}
		}
		run := &execution{snapshot: domain.Execution{Proposal: recipe.RunProposal, Status: "starting"}, done: make(chan struct{})}
		s.changedLocked(run)
		s.runs[id] = run
		writer := &logWriter{service: s, run: run}
		if recipe.static {
			ownedRoot, err := root.OpenRoot(".")
			if err != nil {
				return s.failedStart(run, "No se pudo abrir la carpeta del sitio.")
			}
			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				ownedRoot.Close()
				return s.failedStart(run, "No se pudo reservar un puerto local.")
			}
			server := &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: staticHandler(ownedRoot, dir, writer)}
			run.stop = server.Close
			run.snapshot.URL = "http://" + listener.Addr().String()
			go func() {
				err := server.Serve(listener)
				ownedRoot.Close()
				if errors.Is(err, http.ErrServerClosed) {
					err = nil
				}
				s.finished(run, 0, err)
			}()
		} else {
			exe, args, err := resolveExecutable(recipe.executable, recipe.args, root, dir)
			if err != nil {
				return s.failedStart(run, err.Error())
			}
			process, err := startProcess(exe, args, dir, executionEnvironment(), writer)
			if err != nil {
				return s.failedStart(run, "No se pudo iniciar la aplicación. Comprueba el runtime y sus permisos.")
			}
			run.stop = process.Stop
			go func() { code, err := process.Wait(); s.finished(run, code, err) }()
		}
		run.snapshot.Status = "running"
		snapshot = run.snapshot
		return nil
	})
	return snapshot, err
}

func (s *Service) failedStart(run *execution, message string) error {
	run.snapshot.Status, run.snapshot.Error = "error", message
	s.changedLocked(run)
	close(run.done)
	return policy(message)
}

func (s *Service) finished(run *execution, code int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.snapshot.Status == "stopping" {
		run.snapshot.Status = "stopped"
	} else if err != nil || code != 0 {
		run.snapshot.Status, run.snapshot.Error = "error", "La aplicación terminó con un error. Consulta su salida."
	} else {
		run.snapshot.Status = "exited"
	}
	run.snapshot.ExitCode = &code
	s.changedLocked(run)
	close(run.done)
}

func isActive(status string) bool {
	return status == "starting" || status == "running" || status == "stopping"
}

// Stop is idempotent and leaves the entry available for another explicit Play.
func (s *Service) Stop(id string) error {
	s.mu.Lock()
	run := s.runs[id]
	if run == nil || !isActive(run.snapshot.Status) {
		s.mu.Unlock()
		return nil
	}
	run.snapshot.Status = "stopping"
	run.snapshot.Error = ""
	s.changedLocked(run)
	stop := run.stop
	s.mu.Unlock()
	if stop != nil {
		if err := stop(); err != nil {
			s.mu.Lock()
			if run.snapshot.Status == "stopping" {
				run.snapshot.Status = "running"
				run.snapshot.Error = "No se pudo detener el proceso. Vuelve a intentarlo."
				s.changedLocked(run)
			}
			s.mu.Unlock()
			return policy("No se pudo detener el proceso. Vuelve a intentarlo.")
		}
	}
	return nil
}

func (s *Service) ClearLogs(id string) (domain.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.runs[id]
	if run == nil {
		return domain.Execution{}, policy("La ejecución ya no está disponible.")
	}
	run.logOffset, run.logSize = 0, 0
	s.changedLocked(run)
	return snapshotLocked(run), nil
}

// Remove stops the process tree and waits for cleanup before deleting the entry.
// A revisioned removal event prevents queued log updates from bringing it back.
func (s *Service) Remove(ctx context.Context, id string) (domain.Execution, error) {
	s.mu.Lock()
	run := s.runs[id]
	if run != nil {
		run.removing = true
	}
	s.mu.Unlock()
	if run != nil {
		defer func() { s.mu.Lock(); run.removing = false; s.mu.Unlock() }()
		if err := s.Stop(id); err != nil {
			return domain.Execution{}, err
		}
		select {
		case <-run.done:
		case <-ctx.Done():
			return domain.Execution{}, policy("No se pudo confirmar la detención. La ejecución no se eliminó; vuelve a intentarlo.")
		}
	}
	s.mu.Lock()
	s.revision++
	removed := domain.Execution{Proposal: domain.RunProposal{ID: id}, Status: "removed", Revision: s.revision}
	delete(s.runs, id)
	emit := s.emit
	s.mu.Unlock()
	if emit != nil {
		emit(removed)
	}
	return removed, nil
}

// Revisions remain monotonic even when a deleted recipe is explicitly run again.
func (s *Service) changedLocked(run *execution) {
	s.revision++
	run.snapshot.Revision = s.revision
	run.dirty = true
}

func (s *Service) List() []domain.Execution {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]domain.Execution, 0, len(s.runs))
	for _, run := range s.runs {
		result = append(result, snapshotLocked(run))
	}
	return result
}

// StopAll waits for process-tree cleanup, with a deadline rather than an unbounded UI wait.
func (s *Service) StopAll(ctx context.Context) error {
	s.mu.Lock()
	ids := make([]string, 0, len(s.runs))
	var done []<-chan struct{}
	for id, run := range s.runs {
		if isActive(run.snapshot.Status) {
			ids = append(ids, id)
			done = append(done, run.done)
		}
	}
	s.mu.Unlock()
	var stopErr error
	for _, id := range ids {
		stopErr = errors.Join(stopErr, s.Stop(id))
	}
	for _, ch := range done {
		select {
		case <-ch:
		case <-ctx.Done():
			return policy("No se pudo confirmar la detención de todas las aplicaciones.")
		}
	}
	return stopErr
}

func (s *Service) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	if err := s.StopAll(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	cancel, done := s.cancel, s.pumpDone
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type logWriter struct {
	service *Service
	run     *execution
}

func (w *logWriter) Write(data []byte) (int, error) {
	n := len(data)
	w.service.mu.Lock()
	defer w.service.mu.Unlock()
	if len(data) > maxLogBytes {
		data = data[len(data)-maxLogBytes:]
	}
	if w.run.logBuffer == nil {
		w.run.logBuffer = make([]byte, maxLogBytes)
	}
	first := copy(w.run.logBuffer[w.run.logOffset:], data)
	copy(w.run.logBuffer, data[first:])
	w.run.logOffset = (w.run.logOffset + len(data)) % maxLogBytes
	w.run.logSize = min(maxLogBytes, w.run.logSize+len(data))
	w.service.changedLocked(w.run)
	return n, nil
}

// Only snapshots allocate a log string; high-volume writers update a fixed ring.
// This avoids copying the entire retained console for every pipe read.
func snapshotLocked(run *execution) domain.Execution {
	snapshot := run.snapshot
	if run.logSize == 0 {
		snapshot.Logs = ""
		return snapshot
	}
	start := (run.logOffset - run.logSize + maxLogBytes) % maxLogBytes
	data := make([]byte, run.logSize)
	first := copy(data, run.logBuffer[start:min(maxLogBytes, start+run.logSize)])
	copy(data[first:], run.logBuffer[:run.logSize-first])
	snapshot.Logs = strings.ToValidUTF8(string(data), "�")
	if len(snapshot.Logs) > maxLogBytes {
		text := snapshot.Logs[len(snapshot.Logs)-maxLogBytes:]
		for len(text) > 0 && !utf8.RuneStart(text[0]) {
			text = text[1:]
		}
		snapshot.Logs = text
	}
	return snapshot
}

func staticHandler(root *os.Root, directory string, output io.Writer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}
		name, err := url.PathUnescape(r.URL.EscapedPath())
		if err != nil || strings.ContainsAny(name, "\\\x00") {
			http.NotFound(w, r)
			return
		}
		name = strings.TrimPrefix(name, "/")
		if name == "" || strings.HasSuffix(name, "/") {
			name += "index.html"
		}
		for _, part := range strings.Split(name, "/") {
			if strings.HasPrefix(part, ".") {
				http.NotFound(w, r)
				return
			}
		}
		if !filepath.IsLocal(name) {
			http.NotFound(w, r)
			return
		}
		// Serving a static site must not expose project credentials or source/config files.
		ext := strings.ToLower(filepath.Ext(name))
		allowed := map[string]bool{".html": true, ".htm": true, ".css": true, ".js": true, ".mjs": true, ".json": true, ".svg": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true, ".woff": true, ".woff2": true, ".ttf": true, ".mp4": true, ".webm": true, ".mp3": true, ".wav": true, ".wasm": true}
		if !allowed[ext] || strings.HasSuffix(name, "package.json") || strings.HasSuffix(name, "lock.json") {
			http.NotFound(w, r)
			return
		}
		data, err := readProject(root, directory, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, time.Time{}, strings.NewReader(string(data)))
		// Do not record paths, query parameters or headers from requests.
		_, _ = io.WriteString(output, r.Method+" · recurso servido\n")
	})
}
