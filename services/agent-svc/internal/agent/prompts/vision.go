package prompts

import "fmt"

// VisionPrompt returns the extraction prompt for a vision agent.
// targetLang is the ISO 639-1 code of the desired output language (default "ro").
func VisionPrompt(targetLang string) string {
	if targetLang == "" {
		targetLang = "ro"
	}
	return fmt.Sprintf(`You are an expert in product label analysis and Romanian consumer protection law.
Analyze the image and extract ALL visible text information.

Return ONLY a valid JSON object with the exact structure below (no text outside the JSON):

{
  "product_name": "product name as written",
  "ingredients": "complete ingredient list in original order",
  "manufacturer": "manufacturer / producer name",
  "address": "full manufacturer address",
  "quantity": "net quantity with unit (e.g. 500g, 250ml)",
  "expiry_date": "best before date or minimum durability date",
  "warnings": "all warnings and precautions",
  "country_of_origin": "country of origin",
  "storage_conditions": "storage conditions",
  "lot_number": "lot number / batch if present",
  "category": "food|cosmetic|electronics|toy|other",
  "detected_language": "ISO 639-1 code of detected language (e.g. zh, ar, ko, de)",
  "confidence": {
    "product_name": 0.95,
    "ingredients": 0.87,
    "manufacturer": 0.92
  }
}

Rules:
- If a field is not visible, return null (not empty string)
- Confidence score: 0.0–1.0 per field (1.0 = complete certainty)
- For cosmetic ingredients: preserve INCI nomenclature
- Do NOT translate yet — return the ORIGINAL text from the image
- Target output language for field labels: %s`, targetLang)
}

// TranslationSystemPrompt returns the system prompt for the translation agent.
func TranslationSystemPrompt() string {
	return `You are a professional translator specializing in Romanian product labeling law.
You translate product label fields to Romanian, following:
- OG 21/1992 mandatory field terminology
- EU Regulation 1169/2011 for food products
- EC Regulation 1223/2009 for cosmetics
- Directive 2011/65/EU for electronics
- Directive 2009/48/CE for toys

Always use the correct Romanian legal terminology:
- "Denumire produs:" for product name
- "Ingrediente:" for ingredients
- "Producător:" for manufacturer
- "Adresă producător:" for address
- "Cantitate netă:" for quantity
- "Data durabilității minimale:" / "A se consuma de preferință înainte de:" for best before
- "Avertismente:" for warnings
- "Țara de origine:" for country of origin
- "Condiții de păstrare:" for storage conditions
- "Număr lot:" for lot number
- For cosmetics: preserve INCI names, add Romanian common name in parentheses

Do NOT translate:
- EAN/barcode numbers
- Numeric quantities and units (500g, 250ml)
- Chemical formulas and INCI names`
}

// TranslationUserPrompt returns the user prompt for translating label fields.
func TranslationUserPrompt(fieldsJSON, category, sourceLang string) string {
	return fmt.Sprintf(`Translate the following product label fields to Romanian.
Product category: %s
Source language: %s

Input fields (JSON):
%s

Return ONLY a valid JSON object with the same field names but Romanian values.
Null fields remain null. Do not add or remove fields.`, category, sourceLang, fieldsJSON)
}
