package agent

import "context"

// ProductCategory represents the type of product for category-specific processing.
type ProductCategory string

const (
	CategoryFood        ProductCategory = "food"
	CategoryCosmetic    ProductCategory = "cosmetic"
	CategoryElectronics ProductCategory = "electronics"
	CategoryToy         ProductCategory = "toy"
	CategoryOther       ProductCategory = "other"
)

// ─── Vision ───────────────────────────────────────────────────────────────────

// VisionRequest is the input to a vision agent.
type VisionRequest struct {
	LabelID        string
	ImageS3Key     string
	WorkspaceID    string
	TargetLanguage string // default "ro"
}

// VisionResult is the structured output from a vision agent.
type VisionResult struct {
	ProductName     *string            `json:"product_name"`
	Ingredients     *string            `json:"ingredients"`
	Manufacturer    *string            `json:"manufacturer"`
	Address         *string            `json:"address"`
	Quantity        *string            `json:"quantity"`
	ExpiryDate      *string            `json:"expiry_date"`
	Warnings        *string            `json:"warnings"`
	CountryOfOrigin *string            `json:"country_of_origin"`
	StorageConditions *string          `json:"storage_conditions"`
	LotNumber       *string            `json:"lot_number"`
	Category        ProductCategory    `json:"category"`
	DetectedLang    string             `json:"detected_language"`
	Confidence      map[string]float32 `json:"confidence"`
	ProviderUsed    string             `json:"provider_used"`
	TokensUsed      int                `json:"tokens_used"`
	LatencyMS       int64              `json:"latency_ms"`
}

// VisionAgent extracts structured data from a label image.
type VisionAgent interface {
	ExtractFields(ctx context.Context, req VisionRequest) (*VisionResult, error)
	Name() string
	EstimateCost(imageBytes int64) float64 // USD per image
}

// ─── Translation ──────────────────────────────────────────────────────────────

type TranslRequest struct {
	LabelID     string
	WorkspaceID string
	Fields      map[string]*string // field name → raw extracted value
	Category    ProductCategory
	SourceLang  string
}

type TranslResult struct {
	Translated   map[string]*string `json:"translated"`
	ProviderUsed string             `json:"provider_used"`
	TokensUsed   int                `json:"tokens_used"`
	LatencyMS    int64              `json:"latency_ms"`
}

// TranslationAgent translates label fields to the target language.
type TranslationAgent interface {
	Translate(ctx context.Context, req TranslRequest) (*TranslResult, error)
	Name() string
	SupportedLanguages() []string
}

// ─── Validation ───────────────────────────────────────────────────────────────

type ValidRequest struct {
	LabelID     string
	WorkspaceID string
	Fields      map[string]*string
	Category    ProductCategory
}

type MissingField struct {
	Field    string `json:"field"`
	Severity string `json:"severity"` // "blocker" | "warning"
	Message  string `json:"message"`
}

type ComplianceResult struct {
	Score        int            `json:"score"` // 0–100
	Missing      []MissingField `json:"missing"`
	RulesVersion string         `json:"rules_version"`
}

// ValidationAgent checks legal compliance of translated label fields.
type ValidationAgent interface {
	Validate(ctx context.Context, req ValidRequest) (*ComplianceResult, error)
	Name() string
	RulesVersion() string
}
