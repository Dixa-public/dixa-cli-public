package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Dixa-public/dixa-cli-public/internal/spec"
)

type Options struct {
	BaseURL    string
	APIKey     string
	Debug      bool
	HTTPClient *http.Client
	ErrWriter  io.Writer
}

type Client struct {
	baseURL    string
	apiKey     string
	debug      bool
	httpClient *http.Client
	errWriter  io.Writer
}

type RequestError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *RequestError) Error() string {
	message := strings.TrimSpace(e.Body)
	if message == "" {
		return fmt.Sprintf("%s %s failed with HTTP %d", e.Method, e.URL, e.StatusCode)
	}
	return fmt.Sprintf("%s %s failed with HTTP %d: %s", e.Method, e.URL, e.StatusCode, message)
}

func New(opts Options) *Client {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		baseURL:    strings.TrimRight(opts.BaseURL, "/"),
		apiKey:     strings.TrimSpace(opts.APIKey),
		debug:      opts.Debug,
		httpClient: httpClient,
		errWriter:  opts.ErrWriter,
	}
}

func (c *Client) ExecuteOperation(ctx context.Context, op spec.Operation, params map[string]any) (any, error) {
	switch op.Mode {
	case "http":
		return c.executeHTTP(ctx, op, params)
	case "analytics_prepare_metric_query":
		return c.prepareAnalyticsMetricQuery(ctx, params)
	case "analytics_prepare_record_query":
		return c.prepareAnalyticsRecordQuery(ctx, params)
	case "analytics_fetch_aggregated_data":
		return c.fetchAggregatedData(ctx, params)
	case "analytics_fetch_unaggregated_data":
		return c.fetchUnaggregatedData(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation mode %q for %s", op.Mode, op.ID)
	}
}

func (c *Client) executeHTTP(ctx context.Context, op spec.Operation, params map[string]any) (any, error) {
	if err := validateOperationParams(op, params); err != nil {
		return nil, err
	}

	path := op.PathTemplate
	query := url.Values{}
	body := map[string]any{}
	bodyParams := make([]spec.Parameter, 0, len(op.Parameters))

	for _, param := range op.Parameters {
		value, ok := params[param.Name]
		if !ok {
			continue
		}

		switch param.Location {
		case "path":
			path = strings.ReplaceAll(path, "{"+param.Name+"}", url.PathEscape(fmt.Sprint(value)))
		case "query":
			addQueryValue(query, param.APIName, value)
		case "body":
			bodyParams = append(bodyParams, param)
			body[param.APIName] = value
		}
	}

	var payload any
	if shouldSendRawJSONBody(op, bodyParams) {
		payload = params[bodyParams[0].Name]
	} else if len(body) > 0 {
		payload = body
	}

	return c.do(ctx, op.HTTPMethod, path, query, payload, op.SuccessMessage)
}

func (c *Client) prepareAnalyticsMetricQuery(ctx context.Context, params map[string]any) (any, error) {
	metricID := stringValue(params["metric_id"])
	pageKey := stringValue(params["page_key"])
	pageLimit, hasPageLimit := intValue(params["page_limit"])

	query := url.Values{}
	if pageKey != "" {
		query.Set("pageKey", pageKey)
	}
	if hasPageLimit {
		query.Set("pageLimit", fmt.Sprintf("%d", pageLimit))
	}

	if metricID == "" {
		return c.do(ctx, http.MethodGet, "/analytics/metrics", query, nil, "")
	}

	detailsRaw, err := c.do(ctx, http.MethodGet, "/analytics/metrics/"+url.PathEscape(metricID), nil, nil, "")
	if err != nil {
		return nil, err
	}

	details := mapValue(detailsRaw)
	data := nestedMap(details, "data")
	filtersRaw := sliceValue(data["filters"])
	availableFilters := make([]map[string]any, 0, len(filtersRaw))
	for _, item := range filtersRaw {
		filterInfo := mapValue(item)
		attribute := stringValue(filterInfo["filterAttribute"])
		if attribute == "" {
			continue
		}
		valuesRaw, err := c.do(ctx, http.MethodGet, "/analytics/filter/"+url.PathEscape(attribute), nil, nil, "")
		values := []any{}
		if err == nil {
			values = sliceValue(nestedMap(mapValue(valuesRaw), "data")["items"])
			if len(values) == 0 {
				values = sliceValue(mapValue(valuesRaw)["data"])
			}
		}
		availableFilters = append(availableFilters, map[string]any{
			"attribute":   attribute,
			"description": stringValue(filterInfo["description"]),
			"values":      values,
		})
	}

	return map[string]any{
		"metric_id":              metricID,
		"description":            stringValue(data["description"]),
		"available_filters":      availableFilters,
		"available_aggregations": sliceValue(data["aggregations"]),
		"related_record_ids":     sliceValue(data["relatedRecordIds"]),
	}, nil
}

func (c *Client) prepareAnalyticsRecordQuery(ctx context.Context, params map[string]any) (any, error) {
	recordID := stringValue(params["record_id"])
	pageKey := stringValue(params["page_key"])
	pageLimit, hasPageLimit := intValue(params["page_limit"])

	query := url.Values{}
	if pageKey != "" {
		query.Set("pageKey", pageKey)
	}
	if hasPageLimit {
		query.Set("pageLimit", fmt.Sprintf("%d", pageLimit))
	}

	if recordID == "" {
		return c.do(ctx, http.MethodGet, "/analytics/records", query, nil, "")
	}

	detailsRaw, err := c.do(ctx, http.MethodGet, "/analytics/records/"+url.PathEscape(recordID), nil, nil, "")
	if err != nil {
		return nil, err
	}

	details := mapValue(detailsRaw)
	data := nestedMap(details, "data")
	filtersRaw := sliceValue(data["filters"])
	availableFilters := make([]map[string]any, 0, len(filtersRaw))
	for _, item := range filtersRaw {
		filterInfo := mapValue(item)
		attribute := stringValue(filterInfo["filterAttribute"])
		if attribute == "" {
			continue
		}
		valuesRaw, err := c.do(ctx, http.MethodGet, "/analytics/filter/"+url.PathEscape(attribute), nil, nil, "")
		values := []any{}
		if err == nil {
			values = sliceValue(mapValue(valuesRaw)["data"])
		}
		availableFilters = append(availableFilters, map[string]any{
			"attribute":   attribute,
			"description": stringValue(filterInfo["description"]),
			"values":      values,
		})
	}

	return map[string]any{
		"record_id":          recordID,
		"description":        stringValue(data["description"]),
		"available_filters":  availableFilters,
		"fields_metadata":    sliceValue(data["fieldsMetadata"]),
		"related_metric_ids": sliceValue(data["relatedMetricIds"]),
	}, nil
}

func (c *Client) fetchAggregatedData(ctx context.Context, params map[string]any) (any, error) {
	metricID := stringValue(params["metric_id"])
	timezone := stringValue(params["timezone"])
	if metricID == "" || timezone == "" {
		return nil, fmt.Errorf("metric_id and timezone are required")
	}

	request, err := buildAnalyticsRequest(metricID, timezone, params, true)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, "/analytics/metrics", nil, request, "")
}

func (c *Client) fetchUnaggregatedData(ctx context.Context, params map[string]any) (any, error) {
	recordID := stringValue(params["record_id"])
	timezone := stringValue(params["timezone"])
	if recordID == "" || timezone == "" {
		return nil, fmt.Errorf("record_id and timezone are required")
	}

	if pageLimit, ok := intValue(params["page_limit"]); ok {
		if pageLimit < 1 {
			return nil, fmt.Errorf("page_limit must be at least 1, but got %d", pageLimit)
		}
		if pageLimit > 300 {
			return nil, fmt.Errorf("page_limit must be at most 300, but got %d", pageLimit)
		}
	}

	request, err := buildAnalyticsRequest(recordID, timezone, params, false)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if pageKey := stringValue(params["page_key"]); pageKey != "" {
		query.Set("pageKey", pageKey)
	}
	if pageLimit, ok := intValue(params["page_limit"]); ok {
		query.Set("pageLimit", fmt.Sprintf("%d", pageLimit))
	}

	return c.do(ctx, http.MethodPost, "/analytics/records", query, request, "")
}

func buildAnalyticsRequest(id, timezone string, params map[string]any, includeAggregations bool) (map[string]any, error) {
	request := map[string]any{
		"id":       id,
		"timezone": timezone,
	}

	periodFilter, hasPeriod := params["period_filter"]
	csidFilter, hasCsid := params["csid_filter"]
	filters, hasFilters := cleanAnalyticsFilters(params["filters"])

	if !hasPeriod && !hasCsid && !hasFilters {
		return nil, fmt.Errorf("at least one of period_filter, csid_filter, or filters must be provided")
	}
	if hasPeriod && !hasFilters {
		return nil, fmt.Errorf("when using period_filter, filters must contain at least one value")
	}

	switch {
	case hasPeriod:
		request["periodFilter"] = periodFilter
		request["filters"] = filters
	case hasCsid:
		request["csidFilter"] = csidFilter
		if hasFilters {
			request["filters"] = filters
		}
	case hasFilters:
		request["filters"] = filters
	}

	if includeAggregations {
		if aggregations, ok := params["aggregations"]; ok {
			request["aggregations"] = aggregations
		}
	}

	return request, nil
}

func cleanAnalyticsFilters(raw any) ([]any, bool) {
	filters := sliceValue(raw)
	if len(filters) == 0 {
		return nil, false
	}

	cleaned := make([]any, 0, len(filters))
	for _, item := range filters {
		filter := mapValue(item)
		if len(filter) == 0 {
			continue
		}
		values := sliceValue(filter["values"])
		if stringValue(filter["attribute"]) == "" || len(values) == 0 {
			continue
		}
		filter["values"] = values
		cleaned = append(cleaned, filter)
	}
	return cleaned, len(cleaned) > 0
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, payload any, successMessage string) (any, error) {
	var bodyReader io.Reader
	var requestBody []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request payload: %w", err)
		}
		requestBody = encoded
		bodyReader = bytes.NewReader(encoded)
	}

	requestURL := c.baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build HTTP request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.apiKey)
	}

	c.debugRequest(req, requestBody)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform HTTP request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read HTTP response: %w", err)
	}
	c.debugResponse(res, responseBody)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &RequestError{
			Method:     method,
			URL:        requestURL,
			StatusCode: res.StatusCode,
			Body:       errorBody(responseBody),
		}
	}

	if len(responseBody) == 0 || res.StatusCode == http.StatusNoContent {
		result := map[string]any{
			"success": true,
			"status":  res.StatusCode,
		}
		if successMessage != "" {
			result["message"] = successMessage
		}
		return result, nil
	}

	var parsed any
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return map[string]any{
			"raw":    string(responseBody),
			"status": res.StatusCode,
		}, nil
	}
	return parsed, nil
}

func (c *Client) debugRequest(req *http.Request, body []byte) {
	if !c.debug || c.errWriter == nil {
		return
	}
	headers := redactHeaders(req.Header)
	fmt.Fprintf(c.errWriter, "[debug] request %s %s\n", req.Method, req.URL.String())
	if len(headers) > 0 {
		fmt.Fprintf(c.errWriter, "[debug] headers %s\n", headers)
	}
	if len(body) > 0 {
		fmt.Fprintf(c.errWriter, "[debug] body %s\n", string(body))
	}
}

func (c *Client) debugResponse(res *http.Response, body []byte) {
	if !c.debug || c.errWriter == nil {
		return
	}
	fmt.Fprintf(c.errWriter, "[debug] response %d %s\n", res.StatusCode, http.StatusText(res.StatusCode))
	if len(body) > 0 {
		fmt.Fprintf(c.errWriter, "[debug] response-body %s\n", string(body))
	}
}

func redactHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.Join(headers.Values(key), ",")
		if strings.EqualFold(key, "Authorization") {
			value = redactSecret(value)
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, " ")
}

func redactSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 6 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:3] + strings.Repeat("*", len(secret)-6) + secret[len(secret)-3:]
}

func addQueryValue(values url.Values, key string, raw any) {
	switch value := raw.(type) {
	case []string:
		for _, item := range value {
			values.Add(key, item)
		}
	case []int:
		for _, item := range value {
			values.Add(key, fmt.Sprintf("%d", item))
		}
	default:
		values.Set(key, fmt.Sprint(raw))
	}
}

func mapValue(raw any) map[string]any {
	value, _ := raw.(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func nestedMap(raw map[string]any, key string) map[string]any {
	return mapValue(raw[key])
}

func sliceValue(raw any) []any {
	switch value := raw.(type) {
	case []any:
		return value
	case []map[string]any:
		out := make([]any, 0, len(value))
		for _, item := range value {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func stringValue(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(raw)
	}
}

func intValue(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func errorBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err == nil {
		encoded, _ := json.Marshal(parsed)
		return string(encoded)
	}
	return string(body)
}

func shouldSendRawJSONBody(op spec.Operation, bodyParams []spec.Parameter) bool {
	if len(bodyParams) != 1 {
		return false
	}

	param := bodyParams[0]
	if param.Type != "json" {
		return false
	}

	switch op.ID {
	case "custom_attributes.update_conversation_custom_attributes", "custom_attributes.update_end_user_custom_attributes":
		return true
	default:
		return false
	}
}

func validateOperationParams(op spec.Operation, params map[string]any) error {
	switch op.ID {
	case "conversations.search_conversations":
		return validateConversationSearch(params)
	default:
		return nil
	}
}

func validateConversationSearch(params map[string]any) error {
	if pageLimit, ok := intValue(params["page_limit"]); ok && pageLimit > 50 {
		return fmt.Errorf("page_limit must be less than or equal to 50, but got %d", pageLimit)
	}

	queryRaw, hasQuery := params["query"]
	filtersRaw, hasFilters := params["filters"]
	if !hasQuery && !hasFilters {
		return fmt.Errorf("conversations search requires at least one of --query or --filters")
	}

	if hasQuery {
		query, ok := queryRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("conversations search --query must be a JSON object like '{\"value\":\"refund\"}'")
		}
		if strings.TrimSpace(stringValue(query["value"])) == "" {
			return fmt.Errorf("conversations search --query must include a non-empty \"value\" field")
		}
	}

	if hasFilters {
		filters, ok := filtersRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("conversations search --filters must be a JSON object with \"strategy\" and \"conditions\", not a JSON array")
		}
		strategy := strings.TrimSpace(stringValue(filters["strategy"]))
		if strategy == "" {
			return fmt.Errorf("conversations search --filters must include a \"strategy\" field with value \"All\" or \"Any\"")
		}
		if strategy != "All" && strategy != "Any" {
			return fmt.Errorf("conversations search --filters.strategy must be \"All\" or \"Any\"")
		}
		conditions := sliceValue(filters["conditions"])
		if len(conditions) == 0 {
			return fmt.Errorf("conversations search --filters must include a non-empty \"conditions\" array")
		}
	}

	return nil
}
