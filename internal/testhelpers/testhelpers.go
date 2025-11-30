package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/demetere/omg/pkg"
	openfgaSdk "github.com/openfga/go-sdk"
	"github.com/openfga/go-sdk/client"
	"github.com/stretchr/testify/require"
	openfgacontainer "github.com/testcontainers/testcontainers-go/modules/openfga"
)

// SetupOpenFGAContainer starts an OpenFGA container and returns it with a configured client
func SetupOpenFGAContainer(t *testing.T, ctx context.Context, modelDSL string) (*openfgacontainer.OpenFGAContainer, *omg.Client) {
	container, err := openfgacontainer.Run(ctx, "openfga/openfga:v1.8.0")
	require.NoError(t, err)

	httpEndpoint, err := container.HttpEndpoint(ctx)
	require.NoError(t, err)

	// Create a store using HTTP API
	storeID, err := createStore(ctx, httpEndpoint, "test-store")
	require.NoError(t, err)

	// Create our client
	client, err := omg.NewClient(omg.Config{
		ApiURL:     httpEndpoint,
		StoreID:    storeID,
		AuthMethod: "none",
	})
	require.NoError(t, err)

	// Write the model if provided
	if modelDSL != "" {
		err = WriteModel(ctx, client, modelDSL)
		require.NoError(t, err)
	}

	return container, client
}

func createStore(ctx context.Context, apiURL, name string) (string, error) {
	reqBody := fmt.Sprintf(`{"name":"%s"}`, name)
	resp, err := http.Post(apiURL+"/stores", "application/json", strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.ID, nil
}

// WriteModel writes an authorization model for testing
// This is a simplified version that parses basic DSL
func WriteModel(ctx context.Context, cl *omg.Client, modelDSL string) error {
	sdkClient := cl.GetSDKClient()

	// Parse model DSL and create type definitions
	// For now, this is hardcoded for common test scenarios
	// In production, you'd use the FGA CLI or a proper DSL parser

	typeDefinitions := parseModelDSL(modelDSL)

	body := client.ClientWriteAuthorizationModelRequest{
		TypeDefinitions: typeDefinitions,
		SchemaVersion:   "1.1",
	}

	_, err := sdkClient.WriteAuthorizationModel(ctx).Body(body).Execute()
	return err
}

// parseModelDSL is a simplified DSL parser for testing
// In production, use the official FGA DSL parser
func parseModelDSL(dsl string) []openfgaSdk.TypeDefinition {
	// This is a very basic parser - for testing only
	// Returns predefined types based on what's in the DSL

	types := []openfgaSdk.TypeDefinition{
		{Type: "user"},
	}

	// Add common test types if they appear in the DSL
	if strings.Contains(dsl, "type document") {
		types = append(types, openfgaSdk.TypeDefinition{
			Type: "document",
			Relations: &map[string]openfgaSdk.Userset{
				"owner":  {This: &map[string]interface{}{}},
				"editor": {This: &map[string]interface{}{}},
				"viewer": {This: &map[string]interface{}{}},
			},
			Metadata: &openfgaSdk.Metadata{
				Relations: &map[string]openfgaSdk.RelationMetadata{
					"owner":  {DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}}},
					"editor": {DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}}},
					"viewer": {DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}}},
				},
			},
		})
	}

	if strings.Contains(dsl, "type folder") {
		types = append(types, openfgaSdk.TypeDefinition{
			Type: "folder",
			Relations: &map[string]openfgaSdk.Userset{
				"owner":  {This: &map[string]interface{}{}},
				"editor": {This: &map[string]interface{}{}},
				"viewer": {This: &map[string]interface{}{}},
			},
			Metadata: &openfgaSdk.Metadata{
				Relations: &map[string]openfgaSdk.RelationMetadata{
					"owner":  {DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}}},
					"editor": {DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}}},
					"viewer": {DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}}},
				},
			},
		})
	}

	if strings.Contains(dsl, "type team") {
		types = append(types, createTeamType())
	}

	if strings.Contains(dsl, "type organization") {
		types = append(types, createOrganizationType())
	}

	if strings.Contains(dsl, "type migration") {
		types = append(types, openfgaSdk.TypeDefinition{
			Type: "system",
		})
		types = append(types, openfgaSdk.TypeDefinition{
			Type: "migration",
			Relations: &map[string]openfgaSdk.Userset{
				"applied": {This: &map[string]interface{}{}},
			},
			Metadata: &openfgaSdk.Metadata{
				Relations: &map[string]openfgaSdk.RelationMetadata{
					"applied": {DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "system"}}},
				},
			},
		})
	}

	return types
}

func createTeamType() openfgaSdk.TypeDefinition {
	relations := &map[string]openfgaSdk.Userset{
		"owner":               {This: &map[string]interface{}{}},
		"admin":               {This: &map[string]interface{}{}},
		"member":              {This: &map[string]interface{}{}},
		"manager":             {This: &map[string]interface{}{}},
		"employee":            {This: &map[string]interface{}{}},
		"can_manage":          {This: &map[string]interface{}{}},
		"can_manage_members":  {This: &map[string]interface{}{}},
		"deprecated":          {This: &map[string]interface{}{}},
	}

	metadata := &openfgaSdk.Metadata{
		Relations: &map[string]openfgaSdk.RelationMetadata{},
	}
	for rel := range *relations {
		(*metadata.Relations)[rel] = openfgaSdk.RelationMetadata{
			DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}},
		}
	}

	return openfgaSdk.TypeDefinition{
		Type:      "team",
		Relations: relations,
		Metadata:  metadata,
	}
}

func createOrganizationType() openfgaSdk.TypeDefinition {
	relations := &map[string]openfgaSdk.Userset{
		"owner":    {This: &map[string]interface{}{}},
		"admin":    {This: &map[string]interface{}{}},
		"member":   {This: &map[string]interface{}{}},
		"employee": {This: &map[string]interface{}{}},
	}

	metadata := &openfgaSdk.Metadata{
		Relations: &map[string]openfgaSdk.RelationMetadata{},
	}
	for rel := range *relations {
		(*metadata.Relations)[rel] = openfgaSdk.RelationMetadata{
			DirectlyRelatedUserTypes: &[]openfgaSdk.RelationReference{{Type: "user"}},
		}
	}

	return openfgaSdk.TypeDefinition{
		Type:      "organization",
		Relations: relations,
		Metadata:  metadata,
	}
}
