package discovery

import (
	"os"

	"arx/internal/provider"
)

func ResolveProfile(spec string, models []ModelInfo) (provider.Profile, error) {
	prof, err := provider.Parse(spec)
	if err != nil {
		return provider.Profile{}, err
	}
	for _, m := range models {
		if m.Provider != prof.Provider.Name || m.Model != prof.Model {
			continue
		}
		prof.Tools = m.Tools
		prof.ToolsKnown = m.ToolsKnown
		if m.MaxOutput > 0 && m.MaxOutput < prof.MaxTokens {
			prof.MaxTokens = m.MaxOutput
		}
		break
	}
	return prof, nil
}

func DefaultSpec(models []ModelInfo) string {
	if spec := os.Getenv("ARX_MODEL"); spec != "" {
		return spec
	}
	if len(models) == 0 {
		return ""
	}
	if p, err := provider.GetProvider("deepseek"); err == nil && p.Key() != "" {
		for _, m := range models {
			if m.Spec == "deepseek/deepseek-flash" {
				return m.Spec
			}
		}
	}
	return models[0].Spec
}
