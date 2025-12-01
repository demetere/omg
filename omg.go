// Package omg provides a model-first migration tool for OpenFGA authorization models.
//
// This package re-exports the core types and functions from the internal pkg directory,
// allowing users to import simply as:
//
//	import "github.com/demetere/omg"
//
// Example usage:
//
//	client, err := omg.NewClient(omg.Config{
//	    ApiURL:  "http://localhost:8080",
//	    StoreID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
package omg

import omgpkg "github.com/demetere/omg/pkg"

// Client types and functions
type (
	// Client wraps the OpenFGA SDK client with convenient methods
	Client = omgpkg.Client

	// Config holds OpenFGA client configuration
	Config = omgpkg.Config

	// Tuple represents an OpenFGA relationship tuple
	Tuple = omgpkg.Tuple

	// ReadTuplesRequest configures tuple read operations
	ReadTuplesRequest = omgpkg.ReadTuplesRequest

	// Store represents an OpenFGA store
	Store = omgpkg.Store
)

// NewClient creates a new OpenFGA client from configuration
var NewClient = omgpkg.NewClient

// Store operations
var (
	CreateStore = omgpkg.CreateStore
	ListStores  = omgpkg.ListStores
	StoreExists = omgpkg.StoreExists
)

// Migration types and functions
type (
	// Migration represents a database migration
	Migration = omgpkg.Migration

	// MigrationInfo contains metadata about a migration
	MigrationInfo = omgpkg.MigrationInfo

	// Tracker tracks applied migrations
	Tracker = omgpkg.Tracker

	// TransformFunc is a function that transforms tuples during migration
	TransformFunc = omgpkg.TransformFunc
)

// NewTracker creates a new migration tracker
var NewTracker = omgpkg.NewTracker

// Migration registry functions
var (
	Register = omgpkg.Register
	GetAll   = omgpkg.GetAll
	Reset    = omgpkg.Reset
)

// Model parsing and state management
type (
	// ModelState represents the state of an authorization model
	ModelState = omgpkg.ModelState

	// TypeState represents the state of a type definition
	TypeState = omgpkg.TypeState

	// ModelChange represents a detected change in the model
	ModelChange = omgpkg.ModelChange

	// ChangeType represents the type of model change
	ChangeType = omgpkg.ChangeType

	// ConfidenceLevel represents confidence in rename detection
	ConfidenceLevel = omgpkg.ConfidenceLevel
)

// ChangeType constants
const (
	ChangeTypeAddType         = omgpkg.ChangeTypeAddType
	ChangeTypeRemoveType      = omgpkg.ChangeTypeRemoveType
	ChangeTypeRenameType      = omgpkg.ChangeTypeRenameType
	ChangeTypeAddRelation     = omgpkg.ChangeTypeAddRelation
	ChangeTypeRemoveRelation  = omgpkg.ChangeTypeRemoveRelation
	ChangeTypeRenameRelation  = omgpkg.ChangeTypeRenameRelation
	ChangeTypeUpdateRelation  = omgpkg.ChangeTypeUpdateRelation
)

// ConfidenceLevel constants
const (
	ConfidenceHigh   = omgpkg.ConfidenceHigh
	ConfidenceMedium = omgpkg.ConfidenceMedium
	ConfidenceLow    = omgpkg.ConfidenceLow
	ConfidenceNone   = omgpkg.ConfidenceNone
)

// Model operations
var (
	ParseDSLToModel                     = omgpkg.ParseDSLToModel
	LoadCurrentModel                    = omgpkg.LoadCurrentModel
	LoadCurrentModelFromPath            = omgpkg.LoadCurrentModelFromPath
	GetCurrentModel                     = omgpkg.GetCurrentModel
	LoadModelStateFromOpenFGA           = omgpkg.LoadModelStateFromOpenFGA
	BuildModelState                     = omgpkg.BuildModelState
	BuildModelStateFromAuthorizationModel = omgpkg.BuildModelStateFromAuthorizationModel
	CompareModels                       = omgpkg.CompareModels
	DetectChanges                       = omgpkg.DetectChanges
	DetectPotentialRenames              = omgpkg.DetectPotentialRenames
	ApplyModelFromDSL                   = omgpkg.ApplyModelFromDSL
	ApplyModelFromFile                  = omgpkg.ApplyModelFromFile
)

// Migration generation
var (
	GenerateMigrationFromChanges = omgpkg.GenerateMigrationFromChanges
)

// Helper functions for migrations
var (
	// Tuple operations
	ReadAllTuples          = omgpkg.ReadAllTuples
	WriteTuplesBatch       = omgpkg.WriteTuplesBatch
	DeleteTuplesBatch      = omgpkg.DeleteTuplesBatch
	CountTuples            = omgpkg.CountTuples
	BackupTuples           = omgpkg.BackupTuples
	RestoreTuples          = omgpkg.RestoreTuples

	// Type operations
	AddTypeToModel         = omgpkg.AddTypeToModel
	RemoveTypeFromModel    = omgpkg.RemoveTypeFromModel
	RenameType             = omgpkg.RenameType

	// Relation operations
	AddRelationToType      = omgpkg.AddRelationToType
	RemoveRelationFromType = omgpkg.RemoveRelationFromType
	UpdateRelationDefinition = omgpkg.UpdateRelationDefinition
	RenameRelation         = omgpkg.RenameRelation
	CopyRelation           = omgpkg.CopyRelation
	DeleteRelation         = omgpkg.DeleteRelation

	// Advanced operations
	MigrateRelationWithTransform = omgpkg.MigrateRelationWithTransform
)
