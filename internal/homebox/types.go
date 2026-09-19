package homebox

import "time"

// TokenResponse is the payload returned by POST /users/login.
// Note: Token already includes the "Bearer " prefix.
type TokenResponse struct {
	Token           string    `json:"token"`
	AttachmentToken string    `json:"attachmentToken"`
	ExpiresAt       time.Time `json:"expiresAt"`
}

// StatusResponse is the payload returned by the unauthenticated GET /status endpoint.
type StatusResponse struct {
	Health bool `json:"health"`
	Versions struct {
		Version string `json:"version"`
		Build   string `json:"build"`
	} `json:"versions"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Build    struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	} `json:"build"`
}

// EntityRef is a minimal entity reference (used for parents, locations).
type EntityRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// TagSummary is a compact tag representation embedded in entities.
type TagSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

// TagOut is the full tag payload returned by /tags endpoints.
type TagOut struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Color       string       `json:"color,omitempty"`
	Icon        string       `json:"icon,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Parent      *TagOut      `json:"parent,omitempty"`
	Children    []TagOut     `json:"children,omitempty"`
}

// EntityTypeOut describes an entity type (e.g. the default "Item" and "Location" types).
type EntityTypeOut struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsLocation bool   `json:"isLocation"`
	Icon       string `json:"icon,omitempty"`
}

// CustomField is a user-defined field on an entity.
type CustomField struct {
	ID           string  `json:"id,omitempty"`
	Type         string  `json:"type"` // text | number | boolean
	Name         string  `json:"name"`
	TextValue    string  `json:"textValue"`
	NumberValue  float64 `json:"numberValue"`
	BooleanValue bool    `json:"booleanValue"`
}

// Attachment describes a file attached to an entity.
type Attachment struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Type      string    `json:"type"` // photo | attachment | manual | warranty | receipt
	Primary   bool      `json:"primary"`
	Path      string    `json:"path"`
	Title     string    `json:"title"`
}

// EntitySummary is the compact entity representation returned in list/search results.
type EntitySummary struct {
	ID            string     `json:"id"`
	AssetID       string     `json:"assetId"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Quantity      float64    `json:"quantity"`
	Insured       bool       `json:"insured"`
	Archived      bool       `json:"archived"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	PurchasePrice float64    `json:"purchasePrice"`
	SoldDate      string     `json:"soldDate,omitempty"`
	Location      *EntityRef `json:"location,omitempty"`
	Tags          []TagSummary `json:"tags,omitempty"`
	Parent        *EntityRef `json:"parent,omitempty"`
	ImageID       string     `json:"imageId,omitempty"`
	ThumbnailID   string     `json:"thumbnailId,omitempty"`
}

// EntityOut is the full entity payload.
type EntityOut struct {
	EntitySummary
	EntityType             *EntityTypeOut `json:"entityType,omitempty"`
	ModelNumber            string         `json:"modelNumber,omitempty"`
	Manufacturer           string         `json:"manufacturer,omitempty"`
	SerialNumber           string         `json:"serialNumber,omitempty"`
	LifetimeWarranty       bool           `json:"lifetimeWarranty"`
	WarrantyExpires        string         `json:"warrantyExpires,omitempty"`
	WarrantyDetails        string         `json:"warrantyDetails,omitempty"`
	PurchaseDate           string         `json:"purchaseDate,omitempty"`
	PurchaseFrom           string         `json:"purchaseFrom,omitempty"`
	SoldTo                 string         `json:"soldTo,omitempty"`
	SoldPrice              float64        `json:"soldPrice"`
	SoldNotes              string         `json:"soldNotes,omitempty"`
	Notes                  string         `json:"notes,omitempty"`
	SyncChildEntityLocations bool         `json:"syncChildEntityLocations"`
	Attachments            []Attachment   `json:"attachments,omitempty"`
	Fields                 []CustomField  `json:"fields,omitempty"`
}

// EntityListResult is the paginated list/search response.
type EntityListResult struct {
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	Total      int             `json:"total"`
	Items      []EntitySummary `json:"items"`
	TotalPrice float64         `json:"totalPrice"`
}

// EntityCreate is the payload for POST /entities.
type EntityCreate struct {
	Name         string   `json:"name"`
	EntityTypeID string   `json:"entityTypeId"`
	ParentID     *string  `json:"parentId,omitempty"`
	Description  string   `json:"description,omitempty"`
	Quantity     float64  `json:"quantity"`
	TagIDs       []string `json:"tagIds"`
	ModelNumber  string   `json:"modelNumber,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
}

// EntityUpdate is the full-replacement payload for PUT /entities/{id}.
type EntityUpdate struct {
	ID                       string        `json:"id"`
	Name                     string        `json:"name"`
	Description              string        `json:"description"`
	Quantity                 float64       `json:"quantity"`
	Insured                  bool          `json:"insured"`
	Archived                 bool          `json:"archived"`
	SyncChildEntityLocations bool          `json:"syncChildEntityLocations"`
	EntityTypeID             string        `json:"entityTypeId"`
	LocationID               *string       `json:"locationId,omitempty"`
	ParentID                 *string       `json:"parentId,omitempty"`
	TagIDs                   []string      `json:"tagIds"`
	AssetID                  string        `json:"assetId"`
	SerialNumber             string        `json:"serialNumber"`
	ModelNumber              string        `json:"modelNumber"`
	Manufacturer             string        `json:"manufacturer"`
	LifetimeWarranty         bool          `json:"lifetimeWarranty"`
	WarrantyExpires          string        `json:"warrantyExpires"`
	WarrantyDetails          string        `json:"warrantyDetails"`
	PurchaseDate             string        `json:"purchaseDate"`
	PurchaseFrom             string        `json:"purchaseFrom"`
	PurchasePrice            float64       `json:"purchasePrice"`
	SoldDate                 string        `json:"soldDate"`
	SoldTo                   string        `json:"soldTo"`
	SoldPrice                float64       `json:"soldPrice"`
	SoldNotes                string        `json:"soldNotes"`
	Notes                    string        `json:"notes"`
	Fields                   []CustomField `json:"fields"`
}

// StatisticsOut is the payload returned by GET /groups/statistics.
type StatisticsOut struct {
	TotalUsers        int     `json:"totalUsers"`
	TotalItems        int     `json:"totalItems"`
	TotalLocations    int     `json:"totalLocations"`
	TotalTags         int     `json:"totalTags"`
	TotalItemPrice    float64 `json:"totalItemPrice"`
	TotalWithWarranty int     `json:"totalWithWarranty"`
}
