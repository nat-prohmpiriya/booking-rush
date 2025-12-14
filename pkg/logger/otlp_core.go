package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

// OTLPCore implements zapcore.Core for sending logs to OTel Collector
type OTLPCore struct {
	zapcore.LevelEnabler
	endpoint      string
	serviceName   string
	client        *http.Client
	buffer        []LogRecord
	bufferMu      sync.Mutex
	batchSize     int
	batchInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// LogRecord represents a log entry in OTLP format
type LogRecord struct {
	Timestamp        int64             `json:"timeUnixNano"`
	SeverityNumber   int32             `json:"severityNumber"`
	SeverityText     string            `json:"severityText"`
	Body             interface{}       `json:"body"`
	Attributes       []KeyValue        `json:"attributes,omitempty"`
	TraceID          string            `json:"traceId,omitempty"`
	SpanID           string            `json:"spanId,omitempty"`
	ObservedTimestamp int64            `json:"observedTimeUnixNano"`
}

// KeyValue represents an attribute
type KeyValue struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// OTLPLogPayload represents the OTLP log export request
type OTLPLogPayload struct {
	ResourceLogs []ResourceLogs `json:"resourceLogs"`
}

// ResourceLogs groups logs by resource
type ResourceLogs struct {
	Resource  Resource    `json:"resource"`
	ScopeLogs []ScopeLogs `json:"scopeLogs"`
}

// Resource represents the resource attributes
type Resource struct {
	Attributes []KeyValue `json:"attributes"`
}

// ScopeLogs groups logs by instrumentation scope
type ScopeLogs struct {
	Scope      Scope       `json:"scope"`
	LogRecords []LogRecord `json:"logRecords"`
}

// Scope represents the instrumentation scope
type Scope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewOTLPCore creates a new OTLP core for sending logs to OTel Collector
func NewOTLPCore(cfg *Config, level zapcore.LevelEnabler) *OTLPCore {
	if cfg == nil || cfg.OTLPEndpoint == "" {
		return nil
	}

	// Construct HTTP endpoint from gRPC endpoint
	// OTel Collector typically exposes HTTP on :4318
	httpEndpoint := fmt.Sprintf("http://%s/v1/logs", cfg.OTLPEndpoint)
	// If endpoint already contains port 4317 (gRPC), change to 4318 (HTTP)
	if cfg.OTLPEndpoint == "localhost:4317" || cfg.OTLPEndpoint == "otel-collector:4317" {
		httpEndpoint = fmt.Sprintf("http://%s/v1/logs",
			cfg.OTLPEndpoint[:len(cfg.OTLPEndpoint)-4]+"4318")
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	batchInterval := cfg.BatchInterval
	if batchInterval <= 0 {
		batchInterval = 1 * time.Second
	}

	timeout := cfg.OTLPTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	core := &OTLPCore{
		LevelEnabler:  level,
		endpoint:      httpEndpoint,
		serviceName:   cfg.ServiceName,
		client:        &http.Client{Timeout: timeout},
		buffer:        make([]LogRecord, 0, batchSize),
		batchSize:     batchSize,
		batchInterval: batchInterval,
		stopChan:      make(chan struct{}),
	}

	// Start background flush goroutine
	core.wg.Add(1)
	go core.flushLoop()

	return core
}

// With adds structured context to the Core
func (c *OTLPCore) With(fields []zapcore.Field) zapcore.Core {
	// Return same core - fields will be added at log time
	return c
}

// Check determines whether the supplied Entry should be logged
func (c *OTLPCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write serializes the Entry and any Fields to the OTLP buffer
func (c *OTLPCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	record := LogRecord{
		Timestamp:         ent.Time.UnixNano(),
		ObservedTimestamp: time.Now().UnixNano(),
		SeverityNumber:    zapLevelToOTLP(ent.Level),
		SeverityText:      ent.Level.String(),
		Body:              map[string]string{"stringValue": ent.Message},
	}

	// Convert fields to attributes
	attrs := make([]KeyValue, 0, len(fields)+2)

	// Add caller info
	if ent.Caller.Defined {
		attrs = append(attrs, KeyValue{
			Key:   "caller",
			Value: map[string]string{"stringValue": ent.Caller.String()},
		})
	}

	// Add logger name
	if ent.LoggerName != "" {
		attrs = append(attrs, KeyValue{
			Key:   "logger",
			Value: map[string]string{"stringValue": ent.LoggerName},
		})
	}

	// Process fields
	for _, f := range fields {
		kv := fieldToKeyValue(f)
		if kv.Key != "" {
			// Check for trace_id and span_id
			if f.Key == "trace_id" {
				record.TraceID = f.String
				continue
			}
			if f.Key == "span_id" {
				record.SpanID = f.String
				continue
			}
			attrs = append(attrs, kv)
		}
	}

	record.Attributes = attrs

	// Add to buffer
	c.bufferMu.Lock()
	c.buffer = append(c.buffer, record)
	shouldFlush := len(c.buffer) >= c.batchSize
	c.bufferMu.Unlock()

	if shouldFlush {
		go c.flush()
	}

	return nil
}

// Sync flushes buffered logs
func (c *OTLPCore) Sync() error {
	c.flush()
	return nil
}

// Close stops the background flush loop
func (c *OTLPCore) Close() error {
	close(c.stopChan)
	c.wg.Wait()
	c.flush()
	return nil
}

// flushLoop periodically flushes the buffer
func (c *OTLPCore) flushLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.stopChan:
			return
		}
	}
}

// flush sends buffered logs to OTel Collector
func (c *OTLPCore) flush() {
	c.bufferMu.Lock()
	if len(c.buffer) == 0 {
		c.bufferMu.Unlock()
		return
	}

	// Copy buffer and reset
	records := make([]LogRecord, len(c.buffer))
	copy(records, c.buffer)
	c.buffer = c.buffer[:0]
	c.bufferMu.Unlock()

	// Build OTLP payload
	payload := OTLPLogPayload{
		ResourceLogs: []ResourceLogs{
			{
				Resource: Resource{
					Attributes: []KeyValue{
						{Key: "service.name", Value: map[string]string{"stringValue": c.serviceName}},
						{Key: "service.namespace", Value: map[string]string{"stringValue": "booking-rush"}},
					},
				},
				ScopeLogs: []ScopeLogs{
					{
						Scope: Scope{
							Name:    "go.uber.org/zap",
							Version: "1.27.0",
						},
						LogRecords: records,
					},
				},
			},
		},
	}

	// Send to OTel Collector
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("logger: failed to marshal OTLP payload: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		fmt.Printf("logger: failed to create OTLP request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		// Silently fail - don't block the application
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Log error but don't block
		fmt.Printf("logger: OTLP export failed with status %d\n", resp.StatusCode)
	}
}

// zapLevelToOTLP converts zap log level to OTLP severity number
func zapLevelToOTLP(level zapcore.Level) int32 {
	switch level {
	case zapcore.DebugLevel:
		return 5 // DEBUG
	case zapcore.InfoLevel:
		return 9 // INFO
	case zapcore.WarnLevel:
		return 13 // WARN
	case zapcore.ErrorLevel:
		return 17 // ERROR
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		return 21 // FATAL
	case zapcore.FatalLevel:
		return 21 // FATAL
	default:
		return 0 // UNSPECIFIED
	}
}

// fieldToKeyValue converts a zap field to OTLP KeyValue
func fieldToKeyValue(f zapcore.Field) KeyValue {
	switch f.Type {
	case zapcore.StringType:
		return KeyValue{Key: f.Key, Value: map[string]string{"stringValue": f.String}}
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return KeyValue{Key: f.Key, Value: map[string]int64{"intValue": f.Integer}}
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return KeyValue{Key: f.Key, Value: map[string]uint64{"intValue": uint64(f.Integer)}}
	case zapcore.Float64Type, zapcore.Float32Type:
		return KeyValue{Key: f.Key, Value: map[string]float64{"doubleValue": float64(f.Integer)}}
	case zapcore.BoolType:
		return KeyValue{Key: f.Key, Value: map[string]bool{"boolValue": f.Integer == 1}}
	case zapcore.DurationType:
		return KeyValue{Key: f.Key, Value: map[string]string{"stringValue": time.Duration(f.Integer).String()}}
	case zapcore.TimeType:
		if f.Interface != nil {
			return KeyValue{Key: f.Key, Value: map[string]string{"stringValue": time.Unix(0, f.Integer).In(f.Interface.(*time.Location)).Format(time.RFC3339Nano)}}
		}
		return KeyValue{Key: f.Key, Value: map[string]string{"stringValue": time.Unix(0, f.Integer).Format(time.RFC3339Nano)}}
	case zapcore.ErrorType:
		if f.Interface != nil {
			return KeyValue{Key: f.Key, Value: map[string]string{"stringValue": f.Interface.(error).Error()}}
		}
		return KeyValue{}
	case zapcore.StringerType:
		if f.Interface != nil {
			return KeyValue{Key: f.Key, Value: map[string]string{"stringValue": f.Interface.(fmt.Stringer).String()}}
		}
		return KeyValue{}
	default:
		// For complex types, try to marshal as JSON
		if f.Interface != nil {
			if data, err := json.Marshal(f.Interface); err == nil {
				return KeyValue{Key: f.Key, Value: map[string]string{"stringValue": string(data)}}
			}
		}
		return KeyValue{}
	}
}
