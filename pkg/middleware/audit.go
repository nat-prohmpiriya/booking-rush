package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionCreate  AuditAction = "create"
	AuditActionUpdate  AuditAction = "update"
	AuditActionDelete  AuditAction = "delete"
	AuditActionLogin   AuditAction = "login"
	AuditActionLogout  AuditAction = "logout"
	AuditActionReserve AuditAction = "reserve"
	AuditActionConfirm AuditAction = "confirm"
	AuditActionCancel  AuditAction = "cancel"
	AuditActionRefund  AuditAction = "refund"
	AuditActionView    AuditAction = "view"
)

// Context keys for audit data
const (
	ContextKeyAuditResourceType = "audit_resource_type"
	ContextKeyAuditResourceID   = "audit_resource_id"
	ContextKeyAuditOldValues    = "audit_old_values"
	ContextKeyAuditNewValues    = "audit_new_values"
	ContextKeyAuditMetadata     = "audit_metadata"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID           string                 `json:"id"`
	TenantID     *string                `json:"tenant_id,omitempty"`
	UserID       *string                `json:"user_id,omitempty"`
	UserEmail    string                 `json:"user_email,omitempty"`
	UserRole     string                 `json:"user_role,omitempty"`
	Action       AuditAction            `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	OldValues    map[string]interface{} `json:"old_values,omitempty"`
	NewValues    map[string]interface{} `json:"new_values,omitempty"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AuditConfig holds configuration for the audit middleware
type AuditConfig struct {
	// DB is the PostgreSQL connection pool for storing audit logs
	DB *pgxpool.Pool
	// BufferSize is the size of the async audit buffer (default: 1000)
	BufferSize int
	// FlushInterval is how often to flush the buffer (default: 5 seconds)
	FlushInterval time.Duration
	// BatchSize is the maximum number of entries to insert in one batch (default: 100)
	BatchSize int
	// SkipPaths is a list of paths to skip auditing
	SkipPaths []string
	// SkipMethods is a list of HTTP methods to skip (default: GET, HEAD, OPTIONS)
	SkipMethods []string
	// ActionMapper maps HTTP method + path pattern to audit action
	ActionMapper func(method, path string) AuditAction
	// ResourceExtractor extracts resource type and ID from path
	ResourceExtractor func(path string) (resourceType string, resourceID string)
	// EnableRequestBody enables capturing request body (default: false for security)
	EnableRequestBody bool
	// EnableResponseBody enables capturing response body (default: false)
	EnableResponseBody bool
	// MaxBodySize limits the size of captured body (default: 10KB)
	MaxBodySize int
	// SensitiveFields are field names that should be masked
	SensitiveFields []string
}

// DefaultAuditConfig returns default configuration
func DefaultAuditConfig(db *pgxpool.Pool) *AuditConfig {
	return &AuditConfig{
		DB:                db,
		BufferSize:        1000,
		FlushInterval:     5 * time.Second,
		BatchSize:         100,
		SkipPaths:         []string{"/health", "/ready", "/metrics"},
		SkipMethods:       []string{"GET", "HEAD", "OPTIONS"},
		ActionMapper:      defaultActionMapper,
		ResourceExtractor: defaultResourceExtractor,
		EnableRequestBody: false,
		EnableResponseBody: false,
		MaxBodySize:       10 * 1024, // 10KB
		SensitiveFields:   []string{"password", "token", "secret", "api_key", "credit_card"},
	}
}

// AuditLogger handles async audit logging
type AuditLogger struct {
	config    *AuditConfig
	buffer    chan *AuditEntry
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once

	// For testing: collect entries instead of writing to DB
	testMode    bool
	testEntries []*AuditEntry
	testMu      sync.Mutex
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *AuditConfig) *AuditLogger {
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	al := &AuditLogger{
		config: config,
		buffer: make(chan *AuditEntry, config.BufferSize),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start background worker
	al.wg.Add(1)
	go al.worker()

	return al
}

// Log adds an audit entry to the buffer (non-blocking)
func (al *AuditLogger) Log(entry *AuditEntry) {
	select {
	case al.buffer <- entry:
		// Entry added to buffer
	default:
		// Buffer full, drop entry (or could implement overflow handling)
	}
}

// Close gracefully shuts down the audit logger
func (al *AuditLogger) Close() error {
	al.closeOnce.Do(func() {
		al.cancel()
		close(al.buffer)
		al.wg.Wait()
	})
	return nil
}

// SetTestMode enables test mode which collects entries instead of writing to DB
func (al *AuditLogger) SetTestMode(enabled bool) {
	al.testMu.Lock()
	defer al.testMu.Unlock()
	al.testMode = enabled
	if enabled {
		al.testEntries = make([]*AuditEntry, 0)
	}
}

// GetTestEntries returns collected test entries (only in test mode)
func (al *AuditLogger) GetTestEntries() []*AuditEntry {
	al.testMu.Lock()
	defer al.testMu.Unlock()
	result := make([]*AuditEntry, len(al.testEntries))
	copy(result, al.testEntries)
	return result
}

// ClearTestEntries clears collected test entries
func (al *AuditLogger) ClearTestEntries() {
	al.testMu.Lock()
	defer al.testMu.Unlock()
	al.testEntries = make([]*AuditEntry, 0)
}

// worker processes audit entries in the background
func (al *AuditLogger) worker() {
	defer al.wg.Done()

	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]*AuditEntry, 0, al.config.BatchSize)

	for {
		select {
		case entry, ok := <-al.buffer:
			if !ok {
				// Channel closed, flush remaining entries
				if len(batch) > 0 {
					al.flush(batch)
				}
				return
			}
			batch = append(batch, entry)
			if len(batch) >= al.config.BatchSize {
				al.flush(batch)
				batch = make([]*AuditEntry, 0, al.config.BatchSize)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				al.flush(batch)
				batch = make([]*AuditEntry, 0, al.config.BatchSize)
			}
		case <-al.ctx.Done():
			// Flush remaining entries before exit
			if len(batch) > 0 {
				al.flush(batch)
			}
			return
		}
	}
}

// flush writes a batch of entries to the database
func (al *AuditLogger) flush(entries []*AuditEntry) {
	if len(entries) == 0 {
		return
	}

	// In test mode, just collect entries
	al.testMu.Lock()
	if al.testMode {
		al.testEntries = append(al.testEntries, entries...)
		al.testMu.Unlock()
		return
	}
	al.testMu.Unlock()

	if al.config.DB == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use batch insert for efficiency
	query := `
		INSERT INTO audit_logs (
			id, tenant_id, user_id, user_email, user_role,
			action, resource_type, resource_id,
			ip_address, user_agent, request_id, trace_id,
			old_values, new_values, changes, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16, $17
		)
	`

	batch := &pgxBatch{}
	for _, entry := range entries {
		oldValuesJSON, _ := json.Marshal(entry.OldValues)
		newValuesJSON, _ := json.Marshal(entry.NewValues)
		changesJSON, _ := json.Marshal(entry.Changes)
		metadataJSON, _ := json.Marshal(entry.Metadata)

		// Handle empty maps
		if string(oldValuesJSON) == "null" {
			oldValuesJSON = nil
		}
		if string(newValuesJSON) == "null" {
			newValuesJSON = nil
		}
		if string(changesJSON) == "null" {
			changesJSON = nil
		}
		if string(metadataJSON) == "null" {
			metadataJSON = []byte("{}")
		}

		batch.Queue(query,
			entry.ID, entry.TenantID, entry.UserID, entry.UserEmail, entry.UserRole,
			string(entry.Action), entry.ResourceType, entry.ResourceID,
			entry.IPAddress, entry.UserAgent, entry.RequestID, entry.TraceID,
			oldValuesJSON, newValuesJSON, changesJSON, metadataJSON, entry.CreatedAt,
		)
	}

	// Execute batch
	for _, item := range batch.items {
		_, err := al.config.DB.Exec(ctx, item.query, item.args...)
		if err != nil {
			// Log error but don't fail - audit logs should not block the application
			continue
		}
	}
}

// pgxBatch is a simple batch helper
type pgxBatch struct {
	items []batchItem
}

type batchItem struct {
	query string
	args  []interface{}
}

func (b *pgxBatch) Queue(query string, args ...interface{}) {
	b.items = append(b.items, batchItem{query: query, args: args})
}

// AuditMiddleware creates a new audit logging middleware
func AuditMiddleware(logger *AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := logger.config

		// Check if path should be skipped
		for _, path := range config.SkipPaths {
			if matchPath(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// Check if method should be skipped
		for _, method := range config.SkipMethods {
			if c.Request.Method == method {
				c.Next()
				return
			}
		}

		// Capture request body if enabled
		var requestBody map[string]interface{}
		if config.EnableRequestBody && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, int64(config.MaxBodySize)))
			if err == nil && len(bodyBytes) > 0 {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				_ = json.Unmarshal(bodyBytes, &requestBody)
				// Mask sensitive fields
				requestBody = maskSensitiveFields(requestBody, config.SensitiveFields)
			}
		}

		// Create response writer wrapper for capturing response
		var responseWriter *auditResponseWriter
		if config.EnableResponseBody {
			responseWriter = &auditResponseWriter{
				ResponseWriter: c.Writer,
				body:           bytes.NewBuffer(nil),
				maxSize:        config.MaxBodySize,
			}
			c.Writer = responseWriter
		}

		// Store start time
		startTime := time.Now()

		// Process request
		c.Next()

		// Skip if request handler indicated skip
		if skip, exists := c.Get("audit_skip"); exists && skip.(bool) {
			return
		}

		// Create audit entry
		entry := &AuditEntry{
			ID:        uuid.New().String(),
			CreatedAt: startTime,
		}

		// Extract user info from context (set by JWT middleware)
		if userID, ok := GetUserID(c); ok && userID != "" {
			entry.UserID = &userID
		}
		if email, ok := GetEmail(c); ok {
			entry.UserEmail = email
		}
		if role, ok := GetRole(c); ok {
			entry.UserRole = role
		}
		if tenantID, ok := GetTenantID(c); ok && tenantID != "" {
			entry.TenantID = &tenantID
		}

		// Extract action
		if config.ActionMapper != nil {
			entry.Action = config.ActionMapper(c.Request.Method, c.Request.URL.Path)
		}

		// Extract resource info
		if config.ResourceExtractor != nil {
			resourceType, resourceID := config.ResourceExtractor(c.Request.URL.Path)
			entry.ResourceType = resourceType
			if resourceID != "" {
				entry.ResourceID = &resourceID
			}
		}

		// Override with context values if set by handlers
		if rt, exists := c.Get(ContextKeyAuditResourceType); exists {
			entry.ResourceType = rt.(string)
		}
		if rid, exists := c.Get(ContextKeyAuditResourceID); exists {
			if s, ok := rid.(string); ok && s != "" {
				entry.ResourceID = &s
			}
		}
		if oldVals, exists := c.Get(ContextKeyAuditOldValues); exists {
			entry.OldValues = oldVals.(map[string]interface{})
		}
		if newVals, exists := c.Get(ContextKeyAuditNewValues); exists {
			entry.NewValues = newVals.(map[string]interface{})
		}
		if meta, exists := c.Get(ContextKeyAuditMetadata); exists {
			entry.Metadata = meta.(map[string]interface{})
		}

		// Compute changes if both old and new values exist
		if entry.OldValues != nil && entry.NewValues != nil {
			entry.Changes = computeChanges(entry.OldValues, entry.NewValues)
		}

		// Add request body as new values if enabled and not already set
		if config.EnableRequestBody && requestBody != nil && entry.NewValues == nil {
			entry.NewValues = requestBody
		}

		// Add response body handling if enabled
		if config.EnableResponseBody && responseWriter != nil {
			var responseBody map[string]interface{}
			if json.Unmarshal(responseWriter.body.Bytes(), &responseBody) == nil {
				if entry.Metadata == nil {
					entry.Metadata = make(map[string]interface{})
				}
				entry.Metadata["response_status"] = responseWriter.status
			}
		}

		// Extract request context
		entry.IPAddress = getClientIP(c)
		entry.UserAgent = c.GetHeader("User-Agent")
		entry.RequestID = c.GetHeader("X-Request-ID")
		entry.TraceID = c.GetHeader("X-Trace-ID")

		// If no request ID, generate one
		if entry.RequestID == "" {
			if reqID, exists := c.Get("request_id"); exists {
				entry.RequestID = reqID.(string)
			}
		}

		// Log asynchronously
		logger.Log(entry)
	}
}

// auditResponseWriter captures response body
type auditResponseWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	status  int
	maxSize int
	written int
}

func (w *auditResponseWriter) Write(b []byte) (int, error) {
	if w.written < w.maxSize {
		remaining := w.maxSize - w.written
		if len(b) <= remaining {
			w.body.Write(b)
			w.written += len(b)
		} else {
			w.body.Write(b[:remaining])
			w.written = w.maxSize
		}
	}
	return w.ResponseWriter.Write(b)
}

func (w *auditResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// defaultActionMapper maps HTTP method to audit action
func defaultActionMapper(method, path string) AuditAction {
	// Check for specific path patterns first
	pathLower := strings.ToLower(path)

	if strings.Contains(pathLower, "/login") {
		return AuditActionLogin
	}
	if strings.Contains(pathLower, "/logout") {
		return AuditActionLogout
	}
	if strings.Contains(pathLower, "/reserve") {
		return AuditActionReserve
	}
	if strings.Contains(pathLower, "/confirm") {
		return AuditActionConfirm
	}
	if strings.Contains(pathLower, "/cancel") {
		return AuditActionCancel
	}
	if strings.Contains(pathLower, "/refund") {
		return AuditActionRefund
	}

	// Default mapping by HTTP method
	switch method {
	case http.MethodPost:
		return AuditActionCreate
	case http.MethodPut, http.MethodPatch:
		return AuditActionUpdate
	case http.MethodDelete:
		return AuditActionDelete
	default:
		return AuditActionView
	}
}

// defaultResourceExtractor extracts resource type and ID from path
// Example: /api/v1/bookings/123 -> ("booking", "123")
func defaultResourceExtractor(path string) (resourceType string, resourceID string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Skip api version prefix
	startIdx := 0
	for i, part := range parts {
		if part == "api" || strings.HasPrefix(part, "v") {
			continue
		}
		startIdx = i
		break
	}

	if startIdx >= len(parts) {
		return "unknown", ""
	}

	// Get resource type (remove trailing 's' for plural)
	resourceType = parts[startIdx]
	if strings.HasSuffix(resourceType, "s") {
		resourceType = resourceType[:len(resourceType)-1]
	}

	// Get resource ID if present
	if startIdx+1 < len(parts) {
		resourceID = parts[startIdx+1]
		// Validate it looks like an ID (UUID or numeric)
		if !isValidID(resourceID) {
			resourceID = ""
		}
	}

	return resourceType, resourceID
}

// isValidID checks if a string looks like a valid ID
func isValidID(s string) bool {
	// Check if UUID format
	if _, err := uuid.Parse(s); err == nil {
		return true
	}
	// Check if numeric
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// getClientIP extracts the client IP address
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// maskSensitiveFields masks sensitive data in a map
func maskSensitiveFields(data map[string]interface{}, sensitiveFields []string) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range data {
		lowKey := strings.ToLower(k)
		masked := false
		for _, sf := range sensitiveFields {
			if strings.Contains(lowKey, strings.ToLower(sf)) {
				result[k] = "[REDACTED]"
				masked = true
				break
			}
		}
		if !masked {
			// Recursively mask nested maps
			if nested, ok := v.(map[string]interface{}); ok {
				result[k] = maskSensitiveFields(nested, sensitiveFields)
			} else {
				result[k] = v
			}
		}
	}
	return result
}

// computeChanges computes the differences between old and new values
func computeChanges(oldVals, newVals map[string]interface{}) map[string]interface{} {
	changes := make(map[string]interface{})

	// Find changed and new fields
	for k, newV := range newVals {
		if oldV, exists := oldVals[k]; exists {
			// Compare values (simple comparison)
			if !jsonEqual(oldV, newV) {
				changes[k] = map[string]interface{}{
					"old": oldV,
					"new": newV,
				}
			}
		} else {
			// New field
			changes[k] = map[string]interface{}{
				"old": nil,
				"new": newV,
			}
		}
	}

	// Find deleted fields
	for k, oldV := range oldVals {
		if _, exists := newVals[k]; !exists {
			changes[k] = map[string]interface{}{
				"old": oldV,
				"new": nil,
			}
		}
	}

	return changes
}

// jsonEqual compares two values for JSON equality
func jsonEqual(a, b interface{}) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// Helper functions for handlers to set audit context

// SetAuditResourceType sets the resource type for audit logging
func SetAuditResourceType(c *gin.Context, resourceType string) {
	c.Set(ContextKeyAuditResourceType, resourceType)
}

// SetAuditResourceID sets the resource ID for audit logging
func SetAuditResourceID(c *gin.Context, resourceID string) {
	c.Set(ContextKeyAuditResourceID, resourceID)
}

// SetAuditOldValues sets the old values for audit logging (before update/delete)
func SetAuditOldValues(c *gin.Context, oldValues map[string]interface{}) {
	c.Set(ContextKeyAuditOldValues, oldValues)
}

// SetAuditNewValues sets the new values for audit logging (after create/update)
func SetAuditNewValues(c *gin.Context, newValues map[string]interface{}) {
	c.Set(ContextKeyAuditNewValues, newValues)
}

// SetAuditMetadata sets additional metadata for audit logging
func SetAuditMetadata(c *gin.Context, metadata map[string]interface{}) {
	c.Set(ContextKeyAuditMetadata, metadata)
}

// SkipAudit marks the current request to skip audit logging
func SkipAudit(c *gin.Context) {
	c.Set("audit_skip", true)
}
