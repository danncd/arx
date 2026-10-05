package skills

import (
	"arx/internal/integrations"
	"arx/internal/platform/atomicfile"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

type Manager struct {
	mu              sync.Mutex
	path, directory string
	skills          []Skill
}

func Open(directory string) (*Manager, error) {
	m := &Manager{path: filepath.Join(directory, "skills.json"), directory: filepath.Join(directory, "skills")}
	if err := os.MkdirAll(m.directory, 0700); err != nil {
		return nil, err
	}
	if err := atomicfile.ReadJSON(m.path, &m.skills); err != nil {
		return nil, err
	}
	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		p := filepath.Join(m.directory, "memo")
		if err := os.MkdirAll(p, 0700); err != nil {
			return nil, err
		}
		if _, err := os.Stat(filepath.Join(p, "SKILL.md")); os.IsNotExist(err) {
			if err := atomicfile.Replace(filepath.Join(p, "SKILL.md"), []byte(memoManifest), false); err != nil {
				return nil, err
			}
		}
		if _, err := m.Save(Edit{ID: "memo", Path: p, Enabled: true, Activation: "auto", Dependencies: []string{"memo"}}); err != nil {
			return nil, err
		}
	}
	return m, nil
}
func (m *Manager) index(id string) int {
	return slices.IndexFunc(m.skills, func(s Skill) bool { return s.ID == id })
}
func (m *Manager) State() []Skill {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := integrations.Copy(m.skills)
	if out == nil {
		return []Skill{}
	}
	for i := range out {
		out[i].Error = ""
		p, err := root(out[i].Path)
		if err == nil {
			var b string
			b, err = readFile(p, "SKILL.md")
			if err == nil {
				out[i].Name, out[i].Description, err = Validate(b)
			}
		}
		if err != nil {
			out[i].Error = err.Error()
		}
		if out[i].Usage.Activity == nil {
			out[i].Usage.Activity = []integrations.Activity{}
		}
	}
	return out
}
func (m *Manager) Detail(id string) (Detail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.detail(id)
}
func (m *Manager) detail(id string) (Detail, error) {
	i := m.index(id)
	if i < 0 {
		return Detail{}, errors.New("Skill was not found")
	}
	s := integrations.Copy(m.skills[i])
	p, err := root(s.Path)
	if err != nil {
		return Detail{}, err
	}
	body, err := readFile(p, "SKILL.md")
	if err != nil {
		return Detail{}, err
	}
	name, description, validationErr := Validate(body)
	if validationErr != nil {
		s.Error = validationErr.Error()
	} else {
		s.Name = name
		s.Description = description
	}
	list, err := files(p)
	return Detail{Skill: s, Instructions: body, Files: list}, err
}
func (m *Manager) Save(edit Edit) ([]Skill, error) {
	if !integrations.Identifier.MatchString(edit.ID) {
		return nil, errors.New("Enter a valid skill ID")
	}
	if edit.Activation != "auto" && edit.Activation != "explicit" {
		return nil, errors.New("Choose a skill activation mode")
	}
	if len(edit.Dependencies) > 32 {
		return nil, errors.New("Too many skill dependencies")
	}
	for _, d := range edit.Dependencies {
		if !integrations.Identifier.MatchString(d) {
			return nil, errors.New("Invalid MCP dependency")
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.index(edit.ID)
	if i < 0 && len(m.skills) >= 64 {
		return nil, errors.New("Choose up to 64 skills")
	}
	p := edit.Path
	created := false
	if p == "" {
		if i >= 0 {
			p = m.skills[i].Path
		} else {
			p = filepath.Join(m.directory, edit.ID)
			if _, err := os.Stat(p); err == nil {
				return nil, errors.New("Skill folder already exists; import it instead")
			}
			if err := os.Mkdir(p, 0700); err != nil {
				return nil, err
			}
			created = true
		}
	}
	cleanup := func() {
		if created {
			os.Remove(filepath.Join(p, "SKILL.md"))
			os.Remove(p)
		}
	}
	resolved, err := root(p)
	if err != nil {
		cleanup()
		return nil, err
	}
	p = resolved
	for j, s := range m.skills {
		if j != i && s.Path == p {
			cleanup()
			return nil, errors.New("Skill folder is already registered")
		}
	}
	body := edit.Instructions
	if body == "" {
		body, err = readFile(p, "SKILL.md")
		if err != nil {
			cleanup()
			return nil, err
		}
	}
	name, description, err := Validate(body)
	if err != nil {
		cleanup()
		return nil, err
	}
	next := integrations.Copy(m.skills)
	skill := Skill{ID: edit.ID, Name: name, Description: description, Path: p, Enabled: edit.Enabled, Activation: edit.Activation, Dependencies: edit.Dependencies}
	if skill.Dependencies == nil {
		skill.Dependencies = []string{}
	}
	if i >= 0 {
		skill.Usage = next[i].Usage
		next[i] = skill
	} else {
		next = append(next, skill)
	}

	target := filepath.Join(p, "SKILL.md")
	if info, e := os.Lstat(target); e == nil && info.Mode()&os.ModeSymlink != 0 {
		cleanup()
		return nil, errors.New("SKILL.md must not be a symbolic link")
	}
	old, oldErr := os.ReadFile(target)
	if edit.Instructions != "" {
		if err := atomicfile.Replace(target, []byte(body), false); err != nil {
			cleanup()
			return nil, err
		}
	}
	if err := atomicfile.WriteJSON(m.path, next); err != nil {
		if oldErr == nil {
			_ = atomicfile.Replace(target, old, false)
		}
		cleanup()
		return nil, err
	}
	m.skills = next
	return integrations.Copy(next), nil
}
func (m *Manager) Remove(id string) ([]Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.index(id)
	if i < 0 {
		return nil, errors.New("Skill was not found")
	}
	next := slices.Delete(integrations.Copy(m.skills), i, i+1)
	if err := atomicfile.WriteJSON(m.path, next); err != nil {
		return nil, err
	}
	m.skills = next
	return integrations.Copy(next), nil
}
func (m *Manager) Read(id, path string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.index(id)
	if i < 0 {
		return "", errors.New("Skill is unavailable")
	}
	p, err := root(m.skills[i].Path)
	if err != nil {
		return "", err
	}
	return readFile(p, path)
}
func (m *Manager) Load(id, conversation string, count bool) (Detail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.index(id)
	if i < 0 || !m.skills[i].Enabled {
		return Detail{}, errors.New("Skill is unavailable")
	}
	d, err := m.detail(id)
	if d.Error != "" {
		return Detail{}, errors.New(d.Error)
	}
	if err != nil {
		return Detail{}, err
	}
	if count {
		next := integrations.Copy(m.skills)
		next[i].Usage.Record("Loaded", conversation, "Succeeded")
		if err := atomicfile.WriteJSON(m.path, next); err != nil {
			return Detail{}, fmt.Errorf("Could not save skill usage: %w", err)
		}
		m.skills = next
		d.Usage = next[i].Usage
	}
	return d, nil
}
