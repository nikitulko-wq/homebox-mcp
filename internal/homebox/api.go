package homebox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// SearchParams are the filters for entity search/list requests.
type SearchParams struct {
	Q             string   // free text; "#000-123" searches by asset ID
	Page          int
	PageSize      int
	TagIDs        []string // repeated "tags" query param
	LocationIDs   []string // repeated "parentIds" query param (an item's location is its parent)
	IsLocation    bool     // true → list locations only
	IncludeArchived bool
	OnlyWithPhoto bool
}

func (p SearchParams) query() url.Values {
	q := url.Values{}
	if p.Q != "" {
		q.Set("q", p.Q)
	}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(p.PageSize))
	}
	for _, t := range p.TagIDs {
		q.Add("tags", t)
	}
	for _, l := range p.LocationIDs {
		q.Add("parentIds", l)
	}
	if p.IsLocation {
		q.Set("isLocation", "true")
	}
	if p.IncludeArchived {
		q.Set("includeArchived", "true")
	}
	if p.OnlyWithPhoto {
		q.Set("onlyWithPhoto", "true")
	}
	return q
}

// SearchEntities lists/searches entities via GET /api/v1/entities.
func (c *Client) SearchEntities(ctx context.Context, p SearchParams) (*EntityListResult, error) {
	var out EntityListResult
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/entities", p.query(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListLocations returns all location entities.
func (c *Client) ListLocations(ctx context.Context) (*EntityListResult, error) {
	return c.SearchEntities(ctx, SearchParams{IsLocation: true, PageSize: 1000})
}

// GetEntity fetches a single full entity by ID.
func (c *Client) GetEntity(ctx context.Context, id string) (*EntityOut, error) {
	var out EntityOut
	path := fmt.Sprintf("/api/v1/entities/%s", url.PathEscape(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateEntity creates a new entity via POST /api/v1/entities.
func (c *Client) CreateEntity(ctx context.Context, in EntityCreate) (*EntityOut, error) {
	if in.TagIDs == nil {
		in.TagIDs = []string{}
	}
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	var out EntityOut
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/entities", nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateEntity performs a full-replacement update via PUT /api/v1/entities/{id}.
func (c *Client) UpdateEntity(ctx context.Context, in EntityUpdate) (*EntityOut, error) {
	if in.TagIDs == nil {
		in.TagIDs = []string{}
	}
	if in.Fields == nil {
		in.Fields = []CustomField{}
	}
	var out EntityOut
	path := fmt.Sprintf("/api/v1/entities/%s", url.PathEscape(in.ID))
	if err := c.doJSON(ctx, http.MethodPut, path, nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteEntity deletes an entity by ID.
func (c *Client) DeleteEntity(ctx context.Context, id string) error {
	path := fmt.Sprintf("/api/v1/entities/%s", url.PathEscape(id))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// ListTags returns all tags via GET /api/v1/tags.
func (c *Client) ListTags(ctx context.Context) ([]TagOut, error) {
	var out []TagOut
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/tags", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListEntityTypes returns all entity types via GET /api/v1/entity-types.
func (c *Client) ListEntityTypes(ctx context.Context) ([]EntityTypeOut, error) {
	var out []EntityTypeOut
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/entity-types", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DefaultItemType finds the group's default "Item" entity type (isLocation=false).
func (c *Client) DefaultItemType(ctx context.Context) (*EntityTypeOut, error) {
	types, err := c.ListEntityTypes(ctx)
	if err != nil {
		return nil, err
	}
	for i := range types {
		if !types[i].IsLocation && types[i].Name == "Item" {
			return &types[i], nil
		}
	}
	// Fall back to the first non-location type if "Item" is not present.
	for i := range types {
		if !types[i].IsLocation {
			return &types[i], nil
		}
	}
	return nil, fmt.Errorf("homebox: no item entity type found (create one in Homebox first)")
}

// GetStats returns group statistics.
func (c *Client) GetStats(ctx context.Context) (*StatisticsOut, error) {
	var out StatisticsOut
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/groups/statistics", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddAttachment uploads a file to an entity via multipart
// POST /api/v1/entities/{id}/attachments. attachmentType may be empty
// (auto-detected by Homebox). Returns the updated full entity.
func (c *Client) AddAttachment(ctx context.Context, entityID string, file io.Reader, filename, attachmentType string, primary bool) (*EntityOut, error) {
	fields := map[string]string{"name": filename}
	if attachmentType != "" {
		fields["type"] = attachmentType
	}
	if primary {
		fields["primary"] = "true"
	}
	var out EntityOut
	path := fmt.Sprintf("/api/v1/entities/%s/attachments", url.PathEscape(entityID))
	if err := c.doMultipart(ctx, http.MethodPost, path, file, filename, fields, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
