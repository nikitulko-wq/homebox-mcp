// Package tools defines the MCP tools exposed by the Homebox server.
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/litvinovns/homebox-mcp/internal/homebox"
)

// Register adds all Homebox tools to the given MCP server.
func Register(server *mcp.Server, hb *homebox.Client) {
	api := &API{hb: hb}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_items",
		Description: "Search Homebox items by text query, optionally filtered by tags and/or locations. Supports pagination. Returns compact item summaries with IDs.",
	}, api.SearchItems)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_item",
		Description: "Get full details of a single Homebox item by its UUID: description, location, tags, purchase/warranty/sold info, custom fields and attachments.",
	}, api.GetItem)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_item_by_asset_id",
		Description: "Look up a Homebox item by its asset ID label (format \"000-123\", as printed on generated asset tags).",
	}, api.GetItemByAssetID)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_locations",
		Description: "List all storage locations in Homebox (IDs, names, descriptions). Use these IDs for parentId/locationId parameters of other tools.",
	}, api.ListLocations)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tags",
		Description: "List all tags (formerly labels) in Homebox with their IDs. Use these IDs for tagIds parameters of other tools.",
	}, api.ListTags)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_entity_types",
		Description: "List all entity types in Homebox (e.g. the default \"Item\" type). Needed only when creating items of a non-default type.",
	}, api.ListEntityTypes)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stats",
		Description: "Get Homebox group statistics: total items, locations, tags, total item value, items under warranty.",
	}, api.GetStats)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_item",
		Description: "Create a new item in Homebox. Only name is required; parentId places the item in a location (get IDs via list_locations), tagIds via list_tags.",
	}, api.CreateItem)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_item",
		Description: "Update fields of an existing Homebox item. Only provided fields are changed; all others keep their current values. Dates use YYYY-MM-DD format, prices are plain numbers, assetId has the form \"000-123\".",
	}, api.UpdateItem)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_item",
		Description: "Permanently delete an item from Homebox by its UUID. This cannot be undone.",
	}, api.DeleteItem)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_attachment",
		Description: "Attach a photo or document from a local file path to a Homebox item. type: photo|attachment|manual|warranty|receipt (default: auto-detect). Use primary=true to make a photo the item's main image.",
	}, api.AddAttachment)
}

// API holds the Homebox client shared by all tool handlers.
type API struct {
	hb *homebox.Client
}

// ---------- shared compact output types ----------

type ItemBrief struct {
	ID            string   `json:"id" jsonschema:"item UUID"`
	AssetID       string   `json:"assetId,omitempty"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Quantity      float64  `json:"quantity"`
	Location      string   `json:"location,omitempty" jsonschema:"location name"`
	Tags          []string `json:"tags,omitempty" jsonschema:"tag names"`
	PurchasePrice float64  `json:"purchasePrice,omitempty"`
	Insured       bool     `json:"insured,omitempty"`
	Archived      bool     `json:"archived,omitempty"`
	HasImage      bool     `json:"hasImage,omitempty"`
}

func brief(e homebox.EntitySummary) ItemBrief {
	b := ItemBrief{
		ID:            e.ID,
		AssetID:       e.AssetID,
		Name:          e.Name,
		Description:   e.Description,
		Quantity:      e.Quantity,
		PurchasePrice: e.PurchasePrice,
		Insured:       e.Insured,
		Archived:      e.Archived,
		HasImage:      e.ImageID != "",
	}
	if e.Location != nil {
		b.Location = e.Location.Name
	} else if e.Parent != nil {
		// In the entity model an item's location is its parent.
		b.Location = e.Parent.Name
	}
	for _, t := range e.Tags {
		b.Tags = append(b.Tags, t.Name)
	}
	return b
}

// ---------- read tools ----------

type SearchItemsIn struct {
	Q               string   `json:"q,omitempty" jsonschema:"text query (name/description/notes); omit to list everything"`
	Page            *int     `json:"page,omitempty" jsonschema:"1-based page number"`
	PageSize        *int     `json:"pageSize,omitempty" jsonschema:"items per page (default 25)"`
	TagIDs          []string `json:"tagIds,omitempty" jsonschema:"filter by tag UUIDs (from list_tags)"`
	LocationIDs     []string `json:"locationIds,omitempty" jsonschema:"filter by location UUIDs (from list_locations)"`
	IncludeArchived *bool    `json:"includeArchived,omitempty" jsonschema:"include archived items"`
	OnlyWithPhoto   *bool    `json:"onlyWithPhoto,omitempty" jsonschema:"only items that have a photo"`
}

type SearchItemsOut struct {
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Total    int         `json:"total"`
	Items    []ItemBrief `json:"items"`
}

func (a *API) SearchItems(ctx context.Context, _ *mcp.CallToolRequest, in SearchItemsIn) (*mcp.CallToolResult, SearchItemsOut, error) {
	p := homebox.SearchParams{
		Q:           in.Q,
		TagIDs:      in.TagIDs,
		LocationIDs: in.LocationIDs,
	}
	if in.Page != nil {
		p.Page = *in.Page
	}
	if in.PageSize != nil {
		p.PageSize = *in.PageSize
	}
	if in.IncludeArchived != nil {
		p.IncludeArchived = *in.IncludeArchived
	}
	if in.OnlyWithPhoto != nil {
		p.OnlyWithPhoto = *in.OnlyWithPhoto
	}
	res, err := a.hb.SearchEntities(ctx, p)
	if err != nil {
		return nil, SearchItemsOut{}, err
	}
	out := SearchItemsOut{Page: res.Page, PageSize: res.PageSize, Total: res.Total}
	for _, e := range res.Items {
		out.Items = append(out.Items, brief(e))
	}
	if out.Items == nil {
		out.Items = []ItemBrief{}
	}
	return nil, out, nil
}

type GetItemIn struct {
	ID string `json:"id" jsonschema:"item UUID"`
}

type GetItemOut struct {
	Item *homebox.EntityOut `json:"item"`
}

func (a *API) GetItem(ctx context.Context, _ *mcp.CallToolRequest, in GetItemIn) (*mcp.CallToolResult, GetItemOut, error) {
	e, err := a.hb.GetEntity(ctx, in.ID)
	if err != nil {
		return nil, GetItemOut{}, err
	}
	return nil, GetItemOut{Item: e}, nil
}

type GetItemByAssetIDIn struct {
	AssetID string `json:"assetId" jsonschema:"asset ID label, e.g. 000-123"`
}

type GetItemByAssetIDOut struct {
	Found bool        `json:"found"`
	Items []ItemBrief `json:"items"`
}

func (a *API) GetItemByAssetID(ctx context.Context, _ *mcp.CallToolRequest, in GetItemByAssetIDIn) (*mcp.CallToolResult, GetItemByAssetIDOut, error) {
	id := strings.TrimSpace(in.AssetID)
	if id == "" {
		return nil, GetItemByAssetIDOut{}, fmt.Errorf("assetId must not be empty")
	}
	res, err := a.hb.SearchEntities(ctx, homebox.SearchParams{Q: "#" + id})
	if err != nil {
		return nil, GetItemByAssetIDOut{}, err
	}
	out := GetItemByAssetIDOut{Found: len(res.Items) > 0, Items: []ItemBrief{}}
	for _, e := range res.Items {
		out.Items = append(out.Items, brief(e))
	}
	return nil, out, nil
}

type NoParams struct{}

type LocationBrief struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ListLocationsOut struct {
	Total     int             `json:"total"`
	Locations []LocationBrief `json:"locations"`
}

func (a *API) ListLocations(ctx context.Context, _ *mcp.CallToolRequest, _ NoParams) (*mcp.CallToolResult, ListLocationsOut, error) {
	res, err := a.hb.ListLocations(ctx)
	if err != nil {
		return nil, ListLocationsOut{}, err
	}
	out := ListLocationsOut{Total: len(res.Items), Locations: []LocationBrief{}}
	for _, e := range res.Items {
		out.Locations = append(out.Locations, LocationBrief{ID: e.ID, Name: e.Name, Description: e.Description})
	}
	return nil, out, nil
}

type TagBrief struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

type ListTagsOut struct {
	Total int        `json:"total"`
	Tags  []TagBrief `json:"tags"`
}

func (a *API) ListTags(ctx context.Context, _ *mcp.CallToolRequest, _ NoParams) (*mcp.CallToolResult, ListTagsOut, error) {
	tags, err := a.hb.ListTags(ctx)
	if err != nil {
		return nil, ListTagsOut{}, err
	}
	out := ListTagsOut{Total: len(tags), Tags: []TagBrief{}}
	for _, t := range tags {
		out.Tags = append(out.Tags, TagBrief{ID: t.ID, Name: t.Name, Description: t.Description, Color: t.Color})
	}
	return nil, out, nil
}

type EntityTypeBrief struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsLocation bool   `json:"isLocation"`
	Icon       string `json:"icon,omitempty"`
}

type ListEntityTypesOut struct {
	Types []EntityTypeBrief `json:"entityTypes"`
}

func (a *API) ListEntityTypes(ctx context.Context, _ *mcp.CallToolRequest, _ NoParams) (*mcp.CallToolResult, ListEntityTypesOut, error) {
	types, err := a.hb.ListEntityTypes(ctx)
	if err != nil {
		return nil, ListEntityTypesOut{}, err
	}
	out := ListEntityTypesOut{Types: []EntityTypeBrief{}}
	for _, t := range types {
		out.Types = append(out.Types, EntityTypeBrief{ID: t.ID, Name: t.Name, IsLocation: t.IsLocation, Icon: t.Icon})
	}
	return nil, out, nil
}

type GetStatsOut struct {
	Stats *homebox.StatisticsOut `json:"statistics"`
}

func (a *API) GetStats(ctx context.Context, _ *mcp.CallToolRequest, _ NoParams) (*mcp.CallToolResult, GetStatsOut, error) {
	s, err := a.hb.GetStats(ctx)
	if err != nil {
		return nil, GetStatsOut{}, err
	}
	return nil, GetStatsOut{Stats: s}, nil
}

// ---------- write tools ----------

type CreateItemIn struct {
	Name         string   `json:"name" jsonschema:"item name (required)"`
	Description  string   `json:"description,omitempty" jsonschema:"short description"`
	ParentID     string   `json:"parentId,omitempty" jsonschema:"UUID of the location (or parent item) to place this item in; get via list_locations"`
	TagIDs       []string `json:"tagIds,omitempty" jsonschema:"tag UUIDs to attach; get via list_tags"`
	Quantity     *float64 `json:"quantity,omitempty" jsonschema:"quantity (default 1, fractional allowed)"`
	EntityTypeID string   `json:"entityTypeId,omitempty" jsonschema:"entity type UUID; defaults to the built-in Item type"`
	ModelNumber  string   `json:"modelNumber,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
}

type CreateItemOut struct {
	Created bool             `json:"created"`
	Item    *homebox.EntityOut `json:"item,omitempty"`
}

func (a *API) CreateItem(ctx context.Context, _ *mcp.CallToolRequest, in CreateItemIn) (*mcp.CallToolResult, CreateItemOut, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, CreateItemOut{}, fmt.Errorf("name must not be empty")
	}
	typeID := in.EntityTypeID
	if typeID == "" {
		t, err := a.hb.DefaultItemType(ctx)
		if err != nil {
			return nil, CreateItemOut{}, err
		}
		typeID = t.ID
	}
	create := homebox.EntityCreate{
		Name:         strings.TrimSpace(in.Name),
		EntityTypeID: typeID,
		Description:  in.Description,
		TagIDs:       in.TagIDs,
		ModelNumber:  in.ModelNumber,
		Manufacturer: in.Manufacturer,
	}
	if in.ParentID != "" {
		pid := in.ParentID
		create.ParentID = &pid
	}
	if in.Quantity != nil {
		create.Quantity = *in.Quantity
	}
	e, err := a.hb.CreateEntity(ctx, create)
	if err != nil {
		return nil, CreateItemOut{}, err
	}
	return nil, CreateItemOut{Created: true, Item: e}, nil
}

// UpdateItemIn uses pointer fields so "not provided" is distinguishable from
// a zero value; only provided fields are changed.
type UpdateItemIn struct {
	ID               string   `json:"id" jsonschema:"UUID of the item to update (required)"`
	Name             *string  `json:"name,omitempty"`
	Description      *string  `json:"description,omitempty"`
	Quantity         *float64 `json:"quantity,omitempty"`
	ParentID         *string  `json:"parentId,omitempty" jsonschema:"new location (or parent item) UUID; empty string removes the parent"`
	TagIDs           []string `json:"tagIds,omitempty" jsonschema:"replaces the whole tag list"`
	Insured          *bool    `json:"insured,omitempty"`
	Archived         *bool    `json:"archived,omitempty"`
	AssetID          *string  `json:"assetId,omitempty" jsonschema:"asset ID label, e.g. 000-123; empty string unassigns"`
	SerialNumber     *string  `json:"serialNumber,omitempty"`
	ModelNumber      *string  `json:"modelNumber,omitempty"`
	Manufacturer     *string  `json:"manufacturer,omitempty"`
	LifetimeWarranty *bool    `json:"lifetimeWarranty,omitempty"`
	WarrantyExpires  *string  `json:"warrantyExpires,omitempty" jsonschema:"date YYYY-MM-DD; empty string clears"`
	WarrantyDetails  *string  `json:"warrantyDetails,omitempty"`
	PurchaseDate     *string  `json:"purchaseDate,omitempty" jsonschema:"date YYYY-MM-DD; empty string clears"`
	PurchaseFrom     *string  `json:"purchaseFrom,omitempty"`
	PurchasePrice    *float64 `json:"purchasePrice,omitempty"`
	SoldDate         *string  `json:"soldDate,omitempty" jsonschema:"date YYYY-MM-DD; empty string clears"`
	SoldTo           *string  `json:"soldTo,omitempty"`
	SoldPrice        *float64 `json:"soldPrice,omitempty"`
	SoldNotes        *string  `json:"soldNotes,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
}

type UpdateItemOut struct {
	Updated bool               `json:"updated"`
	Item    *homebox.EntityOut `json:"item,omitempty"`
}

func (a *API) UpdateItem(ctx context.Context, _ *mcp.CallToolRequest, in UpdateItemIn) (*mcp.CallToolResult, UpdateItemOut, error) {
	if in.ID == "" {
		return nil, UpdateItemOut{}, fmt.Errorf("id must not be empty")
	}
	current, err := a.hb.GetEntity(ctx, in.ID)
	if err != nil {
		return nil, UpdateItemOut{}, fmt.Errorf("fetching item before update: %w", err)
	}

	u := buildUpdate(current)
	if in.Name != nil {
		u.Name = *in.Name
	}
	if in.Description != nil {
		u.Description = *in.Description
	}
	if in.Quantity != nil {
		u.Quantity = *in.Quantity
	}
	if in.ParentID != nil {
		if *in.ParentID == "" {
			u.ParentID = nil
		} else {
			pid := *in.ParentID
			u.ParentID = &pid
		}
	}
	if in.TagIDs != nil {
		u.TagIDs = in.TagIDs
	}
	if in.Insured != nil {
		u.Insured = *in.Insured
	}
	if in.Archived != nil {
		u.Archived = *in.Archived
	}
	if in.AssetID != nil {
		u.AssetID = *in.AssetID
	}
	if in.SerialNumber != nil {
		u.SerialNumber = *in.SerialNumber
	}
	if in.ModelNumber != nil {
		u.ModelNumber = *in.ModelNumber
	}
	if in.Manufacturer != nil {
		u.Manufacturer = *in.Manufacturer
	}
	if in.LifetimeWarranty != nil {
		u.LifetimeWarranty = *in.LifetimeWarranty
	}
	if in.WarrantyExpires != nil {
		u.WarrantyExpires = *in.WarrantyExpires
	}
	if in.WarrantyDetails != nil {
		u.WarrantyDetails = *in.WarrantyDetails
	}
	if in.PurchaseDate != nil {
		u.PurchaseDate = *in.PurchaseDate
	}
	if in.PurchaseFrom != nil {
		u.PurchaseFrom = *in.PurchaseFrom
	}
	if in.PurchasePrice != nil {
		u.PurchasePrice = *in.PurchasePrice
	}
	if in.SoldDate != nil {
		u.SoldDate = *in.SoldDate
	}
	if in.SoldTo != nil {
		u.SoldTo = *in.SoldTo
	}
	if in.SoldPrice != nil {
		u.SoldPrice = *in.SoldPrice
	}
	if in.SoldNotes != nil {
		u.SoldNotes = *in.SoldNotes
	}
	if in.Notes != nil {
		u.Notes = *in.Notes
	}

	e, err := a.hb.UpdateEntity(ctx, u)
	if err != nil {
		return nil, UpdateItemOut{}, err
	}
	return nil, UpdateItemOut{Updated: true, Item: e}, nil
}

// buildUpdate converts a fetched EntityOut into a full EntityUpdate payload.
func buildUpdate(e *homebox.EntityOut) homebox.EntityUpdate {
	u := homebox.EntityUpdate{
		ID:                       e.ID,
		Name:                     e.Name,
		Description:              e.Description,
		Quantity:                 e.Quantity,
		Insured:                  e.Insured,
		Archived:                 e.Archived,
		SyncChildEntityLocations: e.SyncChildEntityLocations,
		AssetID:                  e.AssetID,
		SerialNumber:             e.SerialNumber,
		ModelNumber:              e.ModelNumber,
		Manufacturer:             e.Manufacturer,
		LifetimeWarranty:         e.LifetimeWarranty,
		WarrantyExpires:          e.WarrantyExpires,
		WarrantyDetails:          e.WarrantyDetails,
		PurchaseDate:             e.PurchaseDate,
		PurchaseFrom:             e.PurchaseFrom,
		PurchasePrice:            e.PurchasePrice,
		SoldDate:                 e.SoldDate,
		SoldTo:                   e.SoldTo,
		SoldPrice:                e.SoldPrice,
		SoldNotes:                e.SoldNotes,
		Notes:                    e.Notes,
		Fields:                   e.Fields,
		TagIDs:                   []string{},
	}
	if e.EntityType != nil {
		u.EntityTypeID = e.EntityType.ID
	}
	if e.Location != nil {
		lid := e.Location.ID
		u.LocationID = &lid
	}
	if e.Parent != nil {
		pid := e.Parent.ID
		u.ParentID = &pid
	} else if e.Location != nil {
		// In the entity model an item's location is represented via parentId.
		lid := e.Location.ID
		u.ParentID = &lid
	}
	for _, t := range e.Tags {
		u.TagIDs = append(u.TagIDs, t.ID)
	}
	return u
}

type DeleteItemIn struct {
	ID string `json:"id" jsonschema:"UUID of the item to delete"`
}

type DeleteItemOut struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

func (a *API) DeleteItem(ctx context.Context, _ *mcp.CallToolRequest, in DeleteItemIn) (*mcp.CallToolResult, DeleteItemOut, error) {
	if in.ID == "" {
		return nil, DeleteItemOut{}, fmt.Errorf("id must not be empty")
	}
	if err := a.hb.DeleteEntity(ctx, in.ID); err != nil {
		return nil, DeleteItemOut{}, err
	}
	return nil, DeleteItemOut{Deleted: true, ID: in.ID, Message: "item deleted"}, nil
}

type AddAttachmentIn struct {
	ItemID   string `json:"itemId" jsonschema:"UUID of the item"`
	FilePath string `json:"filePath" jsonschema:"absolute or relative path to the file on the machine where this MCP server runs"`
	Type     string `json:"type,omitempty" jsonschema:"photo|attachment|manual|warranty|receipt; default: auto-detect"`
	Primary  *bool  `json:"primary,omitempty" jsonschema:"true makes a photo the item's main image"`
}

type AttachmentBrief struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Primary bool   `json:"primary"`
}

type AddAttachmentOut struct {
	Attached    bool              `json:"attached"`
	ItemID      string            `json:"itemId"`
	FileName    string            `json:"fileName"`
	Attachments []AttachmentBrief `json:"itemAttachments" jsonschema:"all attachments of the item after upload"`
}

func (a *API) AddAttachment(ctx context.Context, _ *mcp.CallToolRequest, in AddAttachmentIn) (*mcp.CallToolResult, AddAttachmentOut, error) {
	if in.ItemID == "" || strings.TrimSpace(in.FilePath) == "" {
		return nil, AddAttachmentOut{}, fmt.Errorf("itemId and filePath are required")
	}
	switch in.Type {
	case "", "photo", "attachment", "manual", "warranty", "receipt":
	default:
		return nil, AddAttachmentOut{}, fmt.Errorf("invalid type %q: must be one of photo|attachment|manual|warranty|receipt", in.Type)
	}

	f, err := os.Open(in.FilePath)
	if err != nil {
		return nil, AddAttachmentOut{}, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, AddAttachmentOut{}, fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return nil, AddAttachmentOut{}, fmt.Errorf("%s is a directory, not a file", in.FilePath)
	}
	if info.Size() == 0 {
		return nil, AddAttachmentOut{}, fmt.Errorf("%s is empty", in.FilePath)
	}

	primary := in.Primary != nil && *in.Primary
	e, err := a.hb.AddAttachment(ctx, in.ItemID, f, filepath.Base(in.FilePath), in.Type, primary)
	if err != nil {
		return nil, AddAttachmentOut{}, err
	}
	out := AddAttachmentOut{
		Attached:    true,
		ItemID:      in.ItemID,
		FileName:    filepath.Base(in.FilePath),
		Attachments: []AttachmentBrief{},
	}
	for _, att := range e.Attachments {
		out.Attachments = append(out.Attachments, AttachmentBrief{ID: att.ID, Title: att.Title, Type: att.Type, Primary: att.Primary})
	}
	return nil, out, nil
}
