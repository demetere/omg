package omg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	openfgaSdk "github.com/openfga/go-sdk"
	"github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
)

// Client wraps the OpenFGA SDK client with convenient methods
type Client struct {
	sdk     *client.OpenFgaClient
	storeID string
}

// Config holds OpenFGA client configuration
type Config struct {
	ApiURL        string
	StoreID       string
	AuthMethod    string // "none", "token", or "client_credentials"
	APIToken      string
	ClientID      string
	ClientSecret  string
	TokenIssuer   string
	TokenAudience string
}

// NewClient creates a new OpenFGA client from configuration
func NewClient(cfg Config) (*Client, error) {
	if cfg.ApiURL == "" {
		return nil, fmt.Errorf("OPENFGA_API_URL is required")
	}
	if cfg.StoreID == "" {
		return nil, fmt.Errorf("OPENFGA_STORE_ID is required")
	}

	configuration := &client.ClientConfiguration{
		ApiUrl:  cfg.ApiURL,
		StoreId: cfg.StoreID,
	}

	// Configure authentication
	switch cfg.AuthMethod {
	case "token":
		if cfg.APIToken == "" {
			return nil, fmt.Errorf("OPENFGA_API_TOKEN is required when auth method is 'token'")
		}
		configuration.Credentials = &credentials.Credentials{
			Method: credentials.CredentialsMethodApiToken,
			Config: &credentials.Config{
				ApiToken: cfg.APIToken,
			},
		}
	case "client_credentials":
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, fmt.Errorf("OPENFGA_CLIENT_ID and OPENFGA_CLIENT_SECRET are required for client_credentials auth")
		}
		configuration.Credentials = &credentials.Credentials{
			Method: credentials.CredentialsMethodClientCredentials,
			Config: &credentials.Config{
				ClientCredentialsClientId:       cfg.ClientID,
				ClientCredentialsClientSecret:   cfg.ClientSecret,
				ClientCredentialsApiTokenIssuer: cfg.TokenIssuer,
				ClientCredentialsApiAudience:    cfg.TokenAudience,
			},
		}
	case "none", "":
		// No authentication
	default:
		return nil, fmt.Errorf("unknown auth method: %s", cfg.AuthMethod)
	}

	sdkClient, err := client.NewSdkClient(configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenFGA client: %w", err)
	}

	return &Client{
		sdk:     sdkClient,
		storeID: cfg.StoreID,
	}, nil
}

// Tuple represents an OpenFGA relationship tuple
type Tuple struct {
	User     string
	Relation string
	Object   string
}

// ReadTuplesRequest defines parameters for reading tuples
type ReadTuplesRequest struct {
	User     string
	Relation string
	Object   string
}

// WriteTuple writes a single tuple
func (c *Client) WriteTuple(ctx context.Context, tuple Tuple) error {
	body := client.ClientWriteRequest{
		Writes: []openfgaSdk.TupleKey{
			{
				User:     tuple.User,
				Relation: tuple.Relation,
				Object:   tuple.Object,
			},
		},
	}

	_, err := c.sdk.Write(ctx).Body(body).Execute()
	return err
}

// WriteTuples writes multiple tuples in a single request
func (c *Client) WriteTuples(ctx context.Context, tuples []Tuple) error {
	if len(tuples) == 0 {
		return nil
	}

	keys := make([]openfgaSdk.TupleKey, len(tuples))
	for i, tuple := range tuples {
		keys[i] = openfgaSdk.TupleKey{
			User:     tuple.User,
			Relation: tuple.Relation,
			Object:   tuple.Object,
		}
	}

	body := client.ClientWriteRequest{
		Writes: keys,
	}

	_, err := c.sdk.Write(ctx).Body(body).Execute()
	return err
}

// DeleteTuple deletes a single tuple
func (c *Client) DeleteTuple(ctx context.Context, tuple Tuple) error {
	body := client.ClientWriteRequest{
		Deletes: []openfgaSdk.TupleKeyWithoutCondition{
			{
				User:     tuple.User,
				Relation: tuple.Relation,
				Object:   tuple.Object,
			},
		},
	}

	_, err := c.sdk.Write(ctx).Body(body).Execute()
	return err
}

// DeleteTuples deletes multiple tuples in a single request
func (c *Client) DeleteTuples(ctx context.Context, tuples []Tuple) error {
	if len(tuples) == 0 {
		return nil
	}

	keys := make([]openfgaSdk.TupleKeyWithoutCondition, len(tuples))
	for i, tuple := range tuples {
		keys[i] = openfgaSdk.TupleKeyWithoutCondition{
			User:     tuple.User,
			Relation: tuple.Relation,
			Object:   tuple.Object,
		}
	}

	body := client.ClientWriteRequest{
		Deletes: keys,
	}

	_, err := c.sdk.Write(ctx).Body(body).Execute()
	return err
}

// ReadAllTuples reads all tuples matching the request parameters
// Use empty strings to match all values for that parameter
// Note: OpenFGA API requires at least an object type prefix when filtering
func (c *Client) ReadAllTuples(ctx context.Context, req ReadTuplesRequest) ([]Tuple, error) {
	var allTuples []Tuple
	continuationToken := ""

	for {
		body := client.ClientReadRequest{}

		// OpenFGA requires tuple_key to have at least object type if any field is set
		// If only user/relation is set without object, we need to provide empty object prefix
		hasFilter := req.User != "" || req.Relation != "" || req.Object != ""

		if hasFilter {
			// OpenFGA API requires object type when filtering by user or relation
			// If we don't have an object filter, or if object is just a type prefix ending in ":"
			// we need to filter client-side
			objectIsTypeOnly := req.Object != "" && req.Object[len(req.Object)-1] == ':'

			if req.Object == "" && (req.User != "" || req.Relation != "") {
				return c.readAndFilter(ctx, req)
			}

			if objectIsTypeOnly {
				// Filter by object type prefix client-side
				return c.readAndFilterByType(ctx, req)
			}

			// Set filters
			if req.User != "" {
				body.User = openfgaSdk.PtrString(req.User)
			}
			if req.Relation != "" {
				body.Relation = openfgaSdk.PtrString(req.Relation)
			}
			if req.Object != "" {
				body.Object = openfgaSdk.PtrString(req.Object)
			}
		}

		options := client.ClientReadOptions{}
		if continuationToken != "" {
			options.ContinuationToken = openfgaSdk.PtrString(continuationToken)
		}

		response, err := c.sdk.Read(ctx).Body(body).Options(options).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to read tuples: %w", err)
		}

		// Convert SDK tuples to our Tuple type
		for _, t := range response.GetTuples() {
			key := t.GetKey()
			allTuples = append(allTuples, Tuple{
				User:     key.GetUser(),
				Relation: key.GetRelation(),
				Object:   key.GetObject(),
			})
		}

		// Check if there are more pages
		continuationToken = response.GetContinuationToken()
		if continuationToken == "" {
			break
		}
	}

	return allTuples, nil
}

// GetCurrentModel retrieves the current authorization model as DSL string
func (c *Client) GetCurrentModel(ctx context.Context) (string, error) {
	response, err := c.sdk.ReadLatestAuthorizationModel(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("failed to read authorization model: %w", err)
	}

	model := response.GetAuthorizationModel()

	// Convert model to DSL format
	dsl := formatModelAsDSL(model)
	return dsl, nil
}

// formatModelAsDSL converts an authorization model to DSL format
func formatModelAsDSL(model openfgaSdk.AuthorizationModel) string {
	var dsl strings.Builder

	// Add schema version if present
	if schemaVersion := model.GetSchemaVersion(); schemaVersion != "" {
		dsl.WriteString(fmt.Sprintf("model\n  schema %s\n\n", schemaVersion))
	}

	for _, typeDef := range model.GetTypeDefinitions() {
		dsl.WriteString(fmt.Sprintf("type %s\n", typeDef.GetType()))

		relations := typeDef.GetRelations()
		if len(relations) > 0 {
			dsl.WriteString("  relations\n")

			// Get metadata for type restrictions
			var relationsMetadata map[string]openfgaSdk.RelationMetadata
			if metadata := typeDef.Metadata; metadata != nil {
				relationsMetadata = metadata.GetRelations()
			}

			for relName, userset := range relations {
				var typeRestrictions []openfgaSdk.RelationReference
				if relMeta, exists := relationsMetadata[relName]; exists {
					typeRestrictions = relMeta.GetDirectlyRelatedUserTypes()
				}
				dsl.WriteString(fmt.Sprintf("    define %s: %s\n", relName, formatUsersetWithMetadata(userset, typeRestrictions)))
			}
		}
		dsl.WriteString("\n")
	}

	return dsl.String()
}

// formatUserset converts a Userset to DSL format (without metadata)
func formatUserset(userset openfgaSdk.Userset) string {
	return formatUsersetWithMetadata(userset, nil)
}

// formatUsersetWithMetadata converts a Userset to DSL format with type restriction metadata
func formatUsersetWithMetadata(userset openfgaSdk.Userset, typeRestrictions []openfgaSdk.RelationReference) string {
	// Direct assignment (e.g., [user, group#member])
	if this := userset.This; this != nil {
		if len(typeRestrictions) > 0 {
			// Format type restrictions
			var types []string
			for _, tr := range typeRestrictions {
				if tr.Relation != nil && *tr.Relation != "" {
					types = append(types, fmt.Sprintf("%s#%s", tr.Type, *tr.Relation))
				} else {
					types = append(types, tr.Type)
				}
			}
			return "[" + strings.Join(types, ", ") + "]"
		}
		// Fallback if no metadata
		return "[user]"
	}

	// Computed userset (e.g., owner)
	if computedUserset := userset.ComputedUserset; computedUserset != nil {
		if rel := computedUserset.Relation; rel != nil {
			return *rel
		}
	}

	// Tuple to userset (e.g., parent->owner or owner from parent)
	if tupleToUserset := userset.TupleToUserset; tupleToUserset != nil {
		tupleset := ""
		if tupleToUserset.Tupleset.Relation != nil {
			tupleset = *tupleToUserset.Tupleset.Relation
		}

		computedUserset := ""
		if tupleToUserset.ComputedUserset.Relation != nil {
			computedUserset = *tupleToUserset.ComputedUserset.Relation
		}

		// Use "from" syntax: owner from parent
		return fmt.Sprintf("%s from %s", computedUserset, tupleset)
	}

	// Union (e.g., [user] or owner)
	if union := userset.Union; union != nil {
		var parts []string
		for _, child := range union.GetChild() {
			parts = append(parts, formatUserset(child))
		}
		return strings.Join(parts, " or ")
	}

	// Intersection (e.g., [user] and approved)
	if intersection := userset.Intersection; intersection != nil {
		var parts []string
		for _, child := range intersection.GetChild() {
			parts = append(parts, formatUserset(child))
		}
		return strings.Join(parts, " and ")
	}

	// Difference (e.g., [user] but not blocked)
	if difference := userset.Difference; difference != nil {
		base := formatUserset(difference.Base)
		subtract := formatUserset(difference.Subtract)
		return fmt.Sprintf("%s but not %s", base, subtract)
	}

	return "[unknown]"
}

// WriteAuthorizationModel writes a new authorization model
// Note: This requires the model in the correct format
func (c *Client) WriteAuthorizationModel(ctx context.Context, model openfgaSdk.AuthorizationModel) error {
	body := client.ClientWriteAuthorizationModelRequest{
		TypeDefinitions: model.TypeDefinitions,
		SchemaVersion:   model.SchemaVersion,
	}

	_, err := c.sdk.WriteAuthorizationModel(ctx).Body(body).Execute()
	return err
}

// GetCurrentAuthorizationModel retrieves the current authorization model from OpenFGA
func (c *Client) GetCurrentAuthorizationModel(ctx context.Context) (openfgaSdk.AuthorizationModel, error) {
	response, err := c.sdk.ReadLatestAuthorizationModel(ctx).Execute()
	if err != nil {
		return openfgaSdk.AuthorizationModel{}, fmt.Errorf("failed to read authorization model: %w", err)
	}

	return response.GetAuthorizationModel(), nil
}

// GetStoreID returns the store ID
func (c *Client) GetStoreID() string {
	return c.storeID
}

// GetSDKClient returns the underlying SDK client (for testing)
func (c *Client) GetSDKClient() *client.OpenFgaClient {
	return c.sdk
}

// readAndFilter reads all tuples and filters client-side
// Used when OpenFGA API constraints don't allow server-side filtering
func (c *Client) readAndFilter(ctx context.Context, req ReadTuplesRequest) ([]Tuple, error) {
	var allTuples []Tuple
	continuationToken := ""

	// Read all tuples without any filters (direct API call)
	for {
		body := client.ClientReadRequest{}

		options := client.ClientReadOptions{}
		if continuationToken != "" {
			options.ContinuationToken = openfgaSdk.PtrString(continuationToken)
		}

		response, err := c.sdk.Read(ctx).Body(body).Options(options).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to read tuples: %w", err)
		}

		// Convert SDK tuples to our Tuple type
		for _, t := range response.GetTuples() {
			key := t.GetKey()
			allTuples = append(allTuples, Tuple{
				User:     key.GetUser(),
				Relation: key.GetRelation(),
				Object:   key.GetObject(),
			})
		}

		// Check if there are more pages
		continuationToken = response.GetContinuationToken()
		if continuationToken == "" {
			break
		}
	}

	// Filter client-side
	var filtered []Tuple
	for _, tuple := range allTuples {
		match := true

		if req.User != "" && tuple.User != req.User {
			match = false
		}
		if req.Relation != "" && tuple.Relation != req.Relation {
			match = false
		}
		if req.Object != "" && tuple.Object != req.Object {
			match = false
		}

		if match {
			filtered = append(filtered, tuple)
		}
	}

	return filtered, nil
}

// readAndFilterByType reads all tuples and filters by object type prefix
func (c *Client) readAndFilterByType(ctx context.Context, req ReadTuplesRequest) ([]Tuple, error) {
	// Read all tuples
	allTuples, err := c.readAndFilter(ctx, ReadTuplesRequest{})
	if err != nil {
		return nil, err
	}

	// Filter by object type prefix (e.g., "document:")
	var filtered []Tuple
	for _, tuple := range allTuples {
		// Check if object starts with the type prefix
		if strings.HasPrefix(tuple.Object, req.Object) {
			// Also filter by user/relation if specified
			match := true
			if req.User != "" && tuple.User != req.User {
				match = false
			}
			if req.Relation != "" && tuple.Relation != req.Relation {
				match = false
			}
			if match {
				filtered = append(filtered, tuple)
			}
		}
	}

	return filtered, nil
}

// CreateStore creates a new OpenFGA store
// Returns the store ID
func CreateStore(apiURL, storeName string) (string, error) {
	reqBody := map[string]string{"name": storeName}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(apiURL+"/stores", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create store: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create store: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

// Store represents an OpenFGA store
type Store struct {
	ID   string
	Name string
}

// ListStores lists all stores in the OpenFGA instance
func ListStores(apiURL string) ([]Store, error) {
	resp, err := http.Get(apiURL + "/stores")
	if err != nil {
		return nil, fmt.Errorf("failed to list stores: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list stores: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Stores []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"stores"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our Store type
	stores := make([]Store, len(result.Stores))
	for i, s := range result.Stores {
		stores[i] = Store{ID: s.ID, Name: s.Name}
	}

	return stores, nil
}

// StoreExists checks if a store with the given ID exists
func StoreExists(apiURL, storeID string) (bool, error) {
	resp, err := http.Get(fmt.Sprintf("%s/stores/%s", apiURL, storeID))
	if err != nil {
		return false, fmt.Errorf("failed to check store: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("unexpected status checking store: %d, body: %s", resp.StatusCode, string(body))
}
