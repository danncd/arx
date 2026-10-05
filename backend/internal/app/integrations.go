package app

import (
	"arx/internal/integrations/catalog"
	mcp "arx/internal/integrations/mcp"
	skills "arx/internal/integrations/skills"
	"context"
	"errors"
)

func (s *Service) MCPState() []mcp.Server                       { return s.mcp.State() }
func (s *Service) SkillsState() []skills.Skill                  { return s.skills.State() }
func (s *Service) SkillDetail(id string) (skills.Detail, error) { return s.skills.Detail(id) }
func (s *Service) SkillRead(id, path string) (string, error)    { return s.skills.Read(id, path) }
func (s *Service) changeIntegrations(change func() error) error {
	return s.Service.WithState(func(active bool, conversation string, closed bool) error {
		if closed {
			return errors.New("Application is closed")
		}
		if active {
			return errors.New("Stop the current reply before changing integrations")
		}
		return s.tools.Idle(change)
	})
}
func (s *Service) SaveMCP(c mcp.Configuration) ([]mcp.Server, error) {
	var result []mcp.Server
	err := s.changeIntegrations(func() error { var err error; result, err = s.mcp.Save(c); return err })
	return result, err
}
func (s *Service) RemoveMCP(id string) ([]mcp.Server, error) {
	var result []mcp.Server
	err := s.changeIntegrations(func() error { var err error; result, err = s.mcp.Remove(id); return err })
	return result, err
}
func (s *Service) TestMCP(ctx context.Context, id string, configuration *mcp.Configuration) ([]mcp.Server, error) {
	var result []mcp.Server
	err := s.changeIntegrations(func() error {
		var err error
		if configuration != nil {
			result, err = mcp.Probe(ctx, *configuration)
		} else {
			result, err = s.mcp.Test(ctx, id, true)
		}
		return err
	})
	return result, err
}
func (s *Service) SaveSkill(e skills.Edit) ([]skills.Skill, error) {
	var result []skills.Skill
	err := s.changeIntegrations(func() error { var err error; result, err = s.skills.Save(e); return err })
	return result, err
}
func (s *Service) RemoveSkill(id string) ([]skills.Skill, error) {
	var result []skills.Skill
	err := s.changeIntegrations(func() error { var err error; result, err = s.skills.Remove(id); return err })
	return result, err
}
func (s *Service) integrationGuidance() string {
	return catalog.Guidance(s.skills.State(), s.mcp.State())
}
