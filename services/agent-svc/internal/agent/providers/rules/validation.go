package rules

import (
	"context"
	"embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent"
)

//go:embed data/*.yaml
var ruleFS embed.FS

// ruleSet is the schema for each YAML rule file.
type ruleSet struct {
	Version string      `yaml:"version"`
	Rules   []fieldRule `yaml:"rules"`
}

type fieldRule struct {
	Field    string `yaml:"field"`
	Severity string `yaml:"severity"`
	Message  string `yaml:"message"`
}

// rulesAgent implements agent.ValidationAgent using YAML-driven rules.
type rulesAgent struct {
	once    sync.Once
	version string
	common  []fieldRule
	byCategory map[agent.ProductCategory][]fieldRule
}

func NewValidationAgent() agent.ValidationAgent {
	return &rulesAgent{}
}

func (a *rulesAgent) Name() string         { return "rules_engine" }
func (a *rulesAgent) RulesVersion() string { a.load(); return a.version }

func (a *rulesAgent) Validate(_ context.Context, req agent.ValidRequest) (*agent.ComplianceResult, error) {
	a.load()
	rules := append(a.common, a.byCategory[req.Category]...)

	var missing []agent.MissingField
	for _, r := range rules {
		v, ok := req.Fields[r.Field]
		if !ok || v == nil || *v == "" {
			missing = append(missing, agent.MissingField{
				Field:    r.Field,
				Severity: r.Severity,
				Message:  r.Message,
			})
		}
	}

	total := len(rules)
	present := total - len(missing)
	if present < 0 {
		present = 0
	}
	score := 100
	if total > 0 {
		score = (present * 100) / total
	}
	return &agent.ComplianceResult{
		Score:        score,
		Missing:      missing,
		RulesVersion: a.version,
	}, nil
}

// load parses embedded YAML files exactly once.
func (a *rulesAgent) load() {
	a.once.Do(func() {
		a.byCategory = make(map[agent.ProductCategory][]fieldRule)

		common, err := loadFile("data/common.yaml")
		if err != nil {
			panic(fmt.Sprintf("rules: load common.yaml: %v", err))
		}
		a.version = common.Version
		a.common = common.Rules

		cats := map[agent.ProductCategory]string{
			agent.CategoryFood:        "data/food.yaml",
			agent.CategoryCosmetic:    "data/cosmetic.yaml",
			agent.CategoryElectronics: "data/electronics.yaml",
			agent.CategoryToy:         "data/toy.yaml",
		}
		for cat, path := range cats {
			rs, err := loadFile(path)
			if err != nil {
				panic(fmt.Sprintf("rules: load %s: %v", path, err))
			}
			a.byCategory[cat] = rs.Rules
		}
	})
}

func loadFile(path string) (*ruleSet, error) {
	data, err := ruleFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rs ruleSet
	if err := yaml.Unmarshal(data, &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}
