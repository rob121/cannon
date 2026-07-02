package extension

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	defaultRequestPath  = "/request"
	defaultPagePath     = "/page"
	defaultBlockPath    = "/block"
	defaultEndpointPath = "/endpoint"
	defaultDataPath     = "/data"
	defaultAdminPath    = "/admin"
	defaultHelpPath     = "/help"
	defaultHookPath     = "/hooks"
	defaultTemplatePath = "/templates"
)

// Server is a Cannon extension HTTP server over a Unix socket.
type Server struct {
	info   Info
	siteID string

	requestPath  string
	pagePath     string
	blockPath    string
	endpointPath string
	dataPath     string
	adminPath    string
	hookPath     string
	templatePath string

	blocks            map[string]blockEntry
	blockListProvider BlockListProvider

	pages            map[string]pageEntry
	pageListProvider PageListProvider

	endpoints            map[string]endpointEntry
	endpointListProvider EndpointListProvider

	dataHandlers map[string]Handler
	dataFallback Handler

	requestHandler Handler
	adminHandler   Handler

	help *helpSource

	templates *Templates

	install InstallFunc

	config ConfigurationProvider

	hookHandlers map[string]HookHandler

	custom map[string]http.HandlerFunc
}

// New creates an extension server with built-in /health, /meta, /capabilities, and /install.
func New(info Info) *Server {
	return &Server{
		info:         info,
		requestPath:  defaultRequestPath,
		pagePath:     defaultPagePath,
		blockPath:    defaultBlockPath,
		adminPath:    defaultAdminPath,
		templatePath: defaultTemplatePath,
		custom:       make(map[string]http.HandlerFunc),
		blocks:       make(map[string]blockEntry),
		pages:        make(map[string]pageEntry),
		endpoints:    make(map[string]endpointEntry),
		dataHandlers: make(map[string]Handler),
		hookHandlers: make(map[string]HookHandler),
	}
}

// Info returns extension metadata registered with New.
func (s *Server) Info() Info {
	return s.info
}

// HandleRequest registers a request middleware capability handler.
func (s *Server) HandleRequest(path string, fn Handler) {
	if path == "" {
		path = defaultRequestPath
	}
	s.requestPath = normalizeCapabilityPath(path)
	s.requestHandler = fn
}

// RegisterPage registers a page definition and renderer.
// GET {pagePath} lists pages; POST {pagePath}/{id} renders one page.
func (s *Server) RegisterPage(def PageDefinition, fn PageHandler) {
	if def.ID == "" {
		panic("extension: page id is required")
	}
	if s.pagePath == "" {
		s.pagePath = defaultPagePath
	}
	s.pagePath = normalizeCapabilityPath(s.pagePath)
	s.pages[def.ID] = pageEntry{def: def, fn: fn}
}

// OnPageList customizes the page definitions returned from GET /page.
func (s *Server) OnPageList(fn PageListProvider) {
	s.pageListProvider = fn
}

// HandlePage registers a single default page at path for simple extensions.
// Prefer RegisterPage when exposing multiple page types or admin metadata fields.
func (s *Server) HandlePage(path string, fn Handler) {
	if path == "" {
		path = defaultPagePath
	}
	s.pagePath = normalizeCapabilityPath(path)
	s.RegisterPage(PageDefinition{ID: "default", Title: "Default"}, func(item string, req WireRequest) WireResponse {
		return fn(req)
	})
}

// RegisterEndpoint registers an endpoint definition and handler.
// GET {endpointPath} lists endpoints; POST {endpointPath}/{id} handles a request.
func (s *Server) RegisterEndpoint(def EndpointDefinition, fn EndpointHandler) {
	if def.ID == "" {
		panic("extension: endpoint id is required")
	}
	if s.endpointPath == "" {
		s.endpointPath = defaultEndpointPath
	}
	s.endpointPath = normalizeCapabilityPath(s.endpointPath)
	s.endpoints[def.ID] = endpointEntry{def: def, fn: fn}
}

// OnEndpointList customizes the endpoint definitions returned from GET /endpoint.
func (s *Server) OnEndpointList(fn EndpointListProvider) {
	s.endpointListProvider = fn
}

// HandleEndpoint registers a single default endpoint at path for simple extensions.
func (s *Server) HandleEndpoint(path string, fn Handler) {
	if path == "" {
		path = defaultEndpointPath
	}
	s.endpointPath = normalizeCapabilityPath(path)
	s.RegisterEndpoint(EndpointDefinition{ID: "default", Title: "Default"}, func(item string, req WireRequest) WireResponse {
		return fn(req)
	})
}

// HandleData registers a path-based data handler served at POST /data/{relativePath}.
// Cannon exposes it publicly at /ext/{route_hash}/{relativePath} without an admin route.
func (s *Server) HandleData(relativePath string, fn Handler) {
	if s.dataPath == "" {
		s.dataPath = defaultDataPath
	}
	s.dataPath = normalizeCapabilityPath(s.dataPath)
	rel := strings.Trim(strings.TrimSpace(relativePath), "/")
	if rel == "" {
		panic("extension: data route path is required")
	}
	s.dataHandlers[rel] = fn
}

// OnData registers a fallback data handler when no HandleData route matches.
func (s *Server) OnData(fn Handler) {
	if s.dataPath == "" {
		s.dataPath = defaultDataPath
	}
	s.dataPath = normalizeCapabilityPath(s.dataPath)
	s.dataFallback = fn
}

// RegisterBlock registers a block definition and renderer.
// GET {blockPath} lists blocks; POST {blockPath}/{id} renders one block.
func (s *Server) RegisterBlock(def BlockDefinition, fn BlockHandler) {
	if def.ID == "" {
		panic("extension: block id is required")
	}
	if s.blockPath == "" {
		s.blockPath = defaultBlockPath
	}
	s.blockPath = normalizeCapabilityPath(s.blockPath)
	s.blocks[def.ID] = blockEntry{def: def, fn: fn}
}

// OnBlockList customizes the block definitions returned from GET /block.
// Use it when block admin metadata depends on extension state, such as database rows.
func (s *Server) OnBlockList(fn BlockListProvider) {
	s.blockListProvider = fn
}

// HandleBlock registers a single default block at path for simple extensions.
// Prefer RegisterBlock when exposing multiple block types or space mappings.
func (s *Server) HandleBlock(path string, fn Handler) {
	if path == "" {
		path = defaultBlockPath
	}
	s.blockPath = normalizeCapabilityPath(path)
	s.RegisterBlock(BlockDefinition{ID: "default", Title: "Default"}, func(item string, req WireRequest) WireResponse {
		return fn(req)
	})
}

// HandleAdmin registers an admin UI capability handler.
func (s *Server) HandleAdmin(path string, fn Handler) {
	if path == "" {
		path = defaultAdminPath
	}
	s.adminPath = normalizeCapabilityPath(path)
	s.adminHandler = fn
}

// OnHook registers a handler for a Cannon event hook.
// Cannon discovers subscribed hooks via GET /hooks and dispatches with POST /hooks.
func (s *Server) OnHook(event string, fn HookHandler) {
	event = strings.TrimSpace(event)
	if event == "" {
		panic("extension: hook event name is required")
	}
	if s.hookPath == "" {
		s.hookPath = defaultHookPath
	}
	s.hookPath = normalizeCapabilityPath(s.hookPath)
	s.hookHandlers[event] = fn
}

// EmbedHelp serves markdown help articles from an embedded filesystem.
// dir is the folder inside fsys that contains *.md files (for example "help").
// base is the HTTP path prefix (default "/help").
func (s *Server) EmbedHelp(fsys fs.FS, dir string, base ...string) {
	path := defaultHelpPath
	if len(base) > 0 && base[0] != "" {
		path = base[0]
	}
	s.help = newHelpSource(fsys, dir, path)
}

// EmbedTemplates registers HTML templates embedded in the extension binary.
// Local paths such as "contact/form.html" can be overridden from the site
// template directory at "extension/contact/form.html".
func (s *Server) EmbedTemplates(fsys fs.FS, root string) {
	s.templates = NewTemplates(fsys, root)
}

// Templates returns embedded templates registered with EmbedTemplates.
func (s *Server) Templates() *Templates {
	return s.templates
}

// OnInstall registers a one-time setup handler for POST /install.
func (s *Server) OnInstall(fn InstallFunc) {
	s.install = fn
}

// OnConfiguration registers JSON Forms settings served at GET/POST /configuration.
func (s *Server) OnConfiguration(p ConfigurationProvider) {
	s.config = p
}

// Handle registers a custom HTTP route.
func (s *Server) Handle(path string, fn http.HandlerFunc) {
	s.custom[path] = fn
}

// Run parses os.Args and listens on the Unix socket Cannon provides.
func (s *Server) Run() error {
	return s.RunFromArgs(os.Args)
}

// RunFromArgs parses --site= and --socket= from args and starts the server.
func (s *Server) RunFromArgs(args []string) error {
	flags, err := ParseFlags(args)
	if err != nil {
		return err
	}
	return s.Listen(flags.SiteID, flags.SocketPath)
}

// Listen starts the HTTP server on a Unix socket for the given site.
func (s *Server) Listen(siteID, socketPath string) error {
	s.siteID = siteID
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("remove existing socket: %w", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	defer listener.Close()

	mux := s.mux()
	log.Printf("%s extension for site %q listening on %s", s.info.Name, siteID, socketPath)
	return http.Serve(listener, mux)
}

// Handler returns the HTTP handler (useful in tests).
func (s *Server) Handler() http.Handler {
	return s.mux()
}

func (s *Server) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/meta", s.handleMeta)
	mux.HandleFunc("/capabilities", s.handleCapabilities)
	mux.HandleFunc("/install", s.handleInstall)
	if s.config != nil {
		mux.HandleFunc(configurationPath, s.handleConfiguration)
	}

	if s.requestHandler != nil {
		mux.HandleFunc(s.requestPath, s.serveWire(s.requestHandler))
	}
	s.registerPageRoutes(mux)
	s.registerBlockRoutes(mux)
	s.registerEndpointRoutes(mux)
	s.registerDataRoutes(mux)
	s.registerHookRoutes(mux)
	if s.adminHandler != nil {
		mux.HandleFunc(s.adminPath, s.serveWire(s.adminHandler))
		mux.HandleFunc(s.adminPath+"/", s.serveWire(s.adminHandler))
	}
	if s.help != nil {
		s.help.register(mux)
	}
	if s.templates != nil {
		mux.HandleFunc(s.templatePath, s.handleTemplates)
		mux.HandleFunc(s.templatePath+"/", s.handleTemplateSource)
	}
	for path, fn := range s.custom {
		mux.HandleFunc(path, fn)
	}
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"site_id": s.siteID,
	})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, metaResponse{
		Name:          s.info.Name,
		Title:         s.info.Title,
		Description:   s.info.Description,
		Version:       s.info.Version,
		UpdateURLBase: s.info.UpdateURLBase,
		RouteHash:     RouteHash(s.info.Name, s.siteID),
	})
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	caps := map[string]string{}
	if s.requestHandler != nil {
		caps["request"] = s.requestPath
	}
	if len(s.pages) > 0 {
		caps["page"] = s.pageBase()
	}
	if len(s.blocks) > 0 {
		caps["block"] = s.blockBase()
	}
	if len(s.endpoints) > 0 {
		caps["endpoint"] = s.endpointBase()
	}
	if len(s.dataHandlers) > 0 || s.dataFallback != nil {
		caps["data"] = s.dataBase()
	}
	if s.adminHandler != nil {
		caps["admin"] = s.adminPath
	}
	if s.help != nil {
		caps["help"] = s.help.path()
	}
	if s.config != nil {
		caps["configuration"] = configurationPath
	}
	if len(s.hookHandlers) > 0 {
		caps["hooks"] = s.hookPath
	}
	if s.templates != nil {
		caps["templates"] = s.templatePath
	}
	writeJSON(w, http.StatusOK, capabilitiesResponse{
		Capabilities: caps,
		Defaults: capabilityDefaults{
			Admin: adminDefaults{
				MenuName: s.info.AdminMenuName,
			},
		},
	})
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	templates, err := s.templates.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, TemplateListResponse{Templates: templates})
}

func (s *Server) handleTemplateSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, strings.TrimRight(s.templatePath, "/")+"/")
	content, _, err := s.templates.ReadEmbedded(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	override, err := TemplateOverridePath(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, TemplateSourceResponse{
		Path:         normalizeTemplatePath(name),
		OverridePath: override,
		Content:      content,
	})
}

func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		WriteWireResponse(w, Error(http.StatusBadRequest, err.Error()))
		return
	}
	if s.install == nil {
		WriteWireResponse(w, OK())
		return
	}
	if err := s.install(req); err != nil {
		WriteWireResponse(w, Error(http.StatusInternalServerError, err.Error()))
		return
	}
	WriteWireResponse(w, OK())
}

func (s *Server) serveWire(fn Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := DecodeWireRequest(r, s.siteID)
		if err != nil {
			WriteWireResponse(w, Error(http.StatusBadRequest, err.Error()))
			return
		}
		WriteWireResponse(w, fn(req))
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		fmt.Fprintf(os.Stderr, "write response: %v\n", err)
	}
}
