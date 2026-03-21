package agent

// fieldWeights assigns importance to each extracted field for quality scoring.
// Core fields required by EU Reg. 1169/2011 and general label regulations get
// higher weight; optional/supplementary fields get lower weight.
var fieldWeights = map[string]float32{
	"product_name":      1.5,
	"manufacturer":      1.2,
	"quantity":          1.2,
	"ingredients":       1.3,
	"expiry_date":       1.3,
	"country_of_origin": 1.0,
	"warnings":          1.0,
	"lot_number":        0.8,
	"address":           0.7,
	"storage_conditions": 0.6,
}

// ComputeQualityScore returns a weighted average confidence score (0.0–1.0)
// across all fields present in the confidence map. Fields missing from the
// weight table are counted with weight 1.0. Returns 0 if the map is empty.
func ComputeQualityScore(confidence map[string]float32) float32 {
	if len(confidence) == 0 {
		return 0
	}

	var weightedSum, totalWeight float32
	for field, conf := range confidence {
		w := float32(1.0)
		if wv, ok := fieldWeights[field]; ok {
			w = wv
		}
		weightedSum += conf * w
		totalWeight += w
	}

	if totalWeight == 0 {
		return 0
	}
	score := weightedSum / totalWeight
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// QualityLabel converts a 0–1 quality score to a human-readable label.
func QualityLabel(score float32) string {
	switch {
	case score >= 0.90:
		return "excellent"
	case score >= 0.75:
		return "good"
	case score >= 0.55:
		return "fair"
	default:
		return "poor"
	}
}
