package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

func buildAgentSkillCmd() *cobra.Command {
	skill := &cobra.Command{
		Use:   "skill",
		Short: "Manage workspace skills in config.yaml",
	}
	skill.AddCommand(buildAgentSkillListCmd())
	skill.AddCommand(buildAgentSkillAddCmd())
	skill.AddCommand(buildAgentSkillRemoveCmd())
	skill.AddCommand(buildAgentSkillAuditCmd())
	skill.AddCommand(buildAgentSkillInstallCmd())
	skill.AddCommand(buildAgentSkillPublishCmd())
	return skill
}

type SkillAuditResult struct {
	Skill  string   `json:"skill"`
	Status string   `json:"status"`
	Issues []string `json:"issues,omitempty"`
	Fix    string   `json:"fix,omitempty"`
}

type SkillPublishResult struct {
	WorkspaceID string `json:"workspace_id"`
	Skill       string `json:"skill"`
	Destination string `json:"destination"`
}

type SkillInstallResult struct {
	WorkspaceID string `json:"workspace_id"`
	Skill       string `json:"skill"`
	Source      string `json:"source"`
}

func buildAgentSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "list <id>",
		Short:        "List skills configured for a workspace",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			skills, err := fetchWorkspaceSkills(client, args[0])
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(skills)
			}
			if len(skills) == 0 {
				fmt.Println("(no skills configured)")
				return nil
			}
			for _, s := range skills {
				fmt.Println(s)
			}
			return nil
		},
	}
}

func buildAgentSkillAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "add <id> <skill>",
		Short:        "Add a skill to config.yaml if it is not already present",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			skills, raw, err := fetchWorkspaceSkillsWithRaw(client, args[0])
			if err != nil {
				return err
			}
			next, changed := upsertSkill(skills, args[1], true)
			if !changed {
				fmt.Printf("Skill %q already present on %s\n", args[1], shortID(args[0]))
				return nil
			}
			updated, err := replaceSkillsInConfig(raw, next)
			if err != nil {
				return err
			}
			if err := client.PutWorkspaceFile(args[0], "config.yaml", updated); err != nil {
				return err
			}
			fmt.Printf("Added skill %q to %s\n", args[1], shortID(args[0]))
			return nil
		},
	}
}

func buildAgentSkillRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "remove <id> <skill>",
		Short:        "Remove a skill from config.yaml if it is present",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			skills, raw, err := fetchWorkspaceSkillsWithRaw(client, args[0])
			if err != nil {
				return err
			}
			next, changed := upsertSkill(skills, args[1], false)
			if !changed {
				fmt.Printf("Skill %q not found on %s\n", args[1], shortID(args[0]))
				return nil
			}
			updated, err := replaceSkillsInConfig(raw, next)
			if err != nil {
				return err
			}
			if err := client.PutWorkspaceFile(args[0], "config.yaml", updated); err != nil {
				return err
			}
			fmt.Printf("Removed skill %q from %s\n", args[1], shortID(args[0]))
			return nil
		},
	}
}

func buildAgentSkillAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "audit <id>",
		Short:        "Audit configured skills for missing files and metadata",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			results, err := auditWorkspaceSkills(client, args[0])
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(results)
			}

			allPass := true
			for _, result := range results {
				switch result.Status {
				case "PASS":
					fmt.Printf("[PASS] %s\n", result.Skill)
				default:
					allPass = false
					fmt.Printf("[FAIL] %s\n", result.Skill)
					for _, issue := range result.Issues {
						fmt.Printf("  - %s\n", issue)
					}
					if result.Fix != "" {
						fmt.Printf("  Fix: %s\n", result.Fix)
					}
				}
			}
			if !allPass {
				return fmt.Errorf("one or more skills failed audit")
			}
			fmt.Printf("All %d skills passed audit\n", len(results))
			return nil
		},
	}
}

func buildAgentSkillInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "install <id> <source-path>",
		Short:        "Install a local skill folder into a workspace",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			result, err := installWorkspaceSkillFromDir(client, args[0], args[1])
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(result)
			}
			fmt.Printf("Installed skill %q into %s from %s\n", result.Skill, shortID(result.WorkspaceID), result.Source)
			return nil
		},
	}
}

func buildAgentSkillPublishCmd() *cobra.Command {
	var destination string

	cmd := &cobra.Command{
		Use:          "publish <id> <skill>",
		Short:        "Export a workspace skill to a local directory",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if destination == "" {
				return fmt.Errorf("--to is required")
			}
			client := NewPlatformClient(baseURL())
			result, err := publishWorkspaceSkillToDir(client, args[0], args[1], destination)
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(result)
			}
			fmt.Printf("Published skill %q from %s to %s\n", result.Skill, shortID(result.WorkspaceID), result.Destination)
			return nil
		},
	}
	cmd.Flags().StringVar(&destination, "to", "", "Destination directory for the exported skill")
	return cmd
}

func fetchWorkspaceSkills(client *PlatformClient, id string) ([]string, error) {
	skills, _, err := fetchWorkspaceSkillsWithRaw(client, id)
	return skills, err
}

func fetchWorkspaceSkillsWithRaw(client *PlatformClient, id string) ([]string, string, error) {
	file, err := client.GetWorkspaceFile(id, "config.yaml")
	if err != nil {
		return nil, "", err
	}
	skills, err := parseSkillsFromConfig(file.Content)
	if err != nil {
		return nil, "", err
	}
	return skills, file.Content, nil
}

func parseSkillsFromConfig(content string) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, fmt.Errorf("parse config.yaml: %w", err)
	}
	doc := root.Content
	if len(doc) == 0 {
		return []string{}, nil
	}
	mapping := doc[0]
	if mapping.Kind != yaml.MappingNode {
		return []string{}, nil
	}
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		key := mapping.Content[i]
		val := mapping.Content[i+1]
		if key.Value != "skills" || val.Kind != yaml.SequenceNode {
			continue
		}
		skills := make([]string, 0, len(val.Content))
		for _, item := range val.Content {
			if item != nil {
				skills = append(skills, item.Value)
			}
		}
		return skills, nil
	}
	return []string{}, nil
}

func auditWorkspaceSkills(client *PlatformClient, id string) ([]SkillAuditResult, error) {
	skills, err := fetchWorkspaceSkills(client, id)
	if err != nil {
		return nil, err
	}

	results := make([]SkillAuditResult, 0, len(skills))
	for _, skill := range skills {
		res := SkillAuditResult{Skill: skill, Status: "PASS"}

		file, err := client.GetWorkspaceFile(id, "skills/"+skill+"/SKILL.md")
		if err != nil {
			res.Status = "FAIL"
			res.Issues = append(res.Issues, "missing skills/"+skill+"/SKILL.md")
			res.Fix = "Create skills/" + skill + "/SKILL.md with YAML frontmatter and instructions."
			results = append(results, res)
			continue
		}

		issues := auditSkillMarkdown(file.Content)
		if len(issues) > 0 {
			res.Status = "FAIL"
			res.Issues = append(res.Issues, issues...)
			res.Fix = "Add name, description, and version to SKILL.md frontmatter."
		}
		results = append(results, res)
	}

	return results, nil
}

func installWorkspaceSkillFromDir(client *PlatformClient, workspaceID, sourcePath string) (*SkillInstallResult, error) {
	skillDir := filepath.Clean(sourcePath)
	if info, err := os.Stat(skillDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("source path must be a directory: %s", sourcePath)
	}

	skillName := filepath.Base(skillDir)
	files, err := collectFilesFromDir(skillDir)
	if err != nil {
		return nil, err
	}
	if _, ok := files["SKILL.md"]; !ok {
		return nil, fmt.Errorf("source skill is missing SKILL.md")
	}

	skills, raw, err := fetchWorkspaceSkillsWithRaw(client, workspaceID)
	if err != nil {
		return nil, err
	}
	next, _ := upsertSkill(skills, skillName, true)
	updated, err := replaceSkillsInConfig(raw, next)
	if err != nil {
		return nil, err
	}

	for relPath, content := range files {
		if err := client.PutWorkspaceFile(workspaceID, filepath.ToSlash(filepath.Join("skills", skillName, relPath)), content); err != nil {
			return nil, err
		}
	}
	if err := client.PutWorkspaceFile(workspaceID, "config.yaml", updated); err != nil {
		return nil, err
	}

	return &SkillInstallResult{
		WorkspaceID: workspaceID,
		Skill:       skillName,
		Source:      sourcePath,
	}, nil
}

func publishWorkspaceSkillToDir(client *PlatformClient, workspaceID, skillName, destination string) (*SkillPublishResult, error) {
	bundle, err := client.ExportBundle(workspaceID)
	if err != nil {
		return nil, err
	}

	var skill *WorkspaceBundleSkill
	for i := range bundle.Skills {
		if bundle.Skills[i].ID == skillName {
			skill = &bundle.Skills[i]
			break
		}
	}
	if skill == nil {
		return nil, fmt.Errorf("skill %q not found in exported bundle", skillName)
	}

	destDir := filepath.Join(destination, skillName)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("create destination directory: %w", err)
	}
	for relPath, content := range skill.Files {
		outPath := filepath.Join(destDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return nil, fmt.Errorf("create parent directory: %w", err)
		}
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write skill file: %w", err)
		}
	}

	return &SkillPublishResult{
		WorkspaceID: workspaceID,
		Skill:       skillName,
		Destination: destDir,
	}, nil
}

func collectFilesFromDir(dir string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read source skill files: %w", err)
	}
	return files, nil
}

func auditSkillMarkdown(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []string{"SKILL.md is empty"}
	}

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return []string{fmt.Sprintf("failed to parse SKILL.md frontmatter: %v", err)}
	}
	if len(root.Content) == 0 {
		return []string{"SKILL.md is missing frontmatter"}
	}

	node := root.Content[0]
	if node.Kind != yaml.MappingNode {
		return []string{"SKILL.md frontmatter must be a YAML mapping"}
	}

	fields := map[string]bool{}
	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]
		if val.Kind == yaml.ScalarNode && strings.TrimSpace(val.Value) != "" {
			fields[key.Value] = true
		}
	}

	issues := make([]string, 0, 3)
	for _, field := range []string{"name", "description", "version"} {
		if !fields[field] {
			issues = append(issues, fmt.Sprintf("frontmatter missing %q", field))
		}
	}
	return issues
}

func replaceSkillsInConfig(content string, skills []string) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return "", fmt.Errorf("parse config.yaml: %w", err)
	}
	doc := root.Content
	if len(doc) == 0 || doc[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("config.yaml must contain a top-level mapping")
	}
	mapping := doc[0]

	var skillsKey *yaml.Node
	var skillsVal *yaml.Node
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == "skills" {
			skillsKey = mapping.Content[i]
			skillsVal = mapping.Content[i+1]
			break
		}
	}

	if skillsKey == nil {
		skillsKey = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "skills"}
		skillsVal = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		mapping.Content = append(mapping.Content, skillsKey, skillsVal)
	} else {
		skillsVal.Kind = yaml.SequenceNode
		skillsVal.Tag = "!!seq"
		skillsVal.Content = nil
	}

	for _, skill := range skills {
		skillsVal.Content = append(skillsVal.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: skill,
		})
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return "", fmt.Errorf("marshal config.yaml: %w", err)
	}
	return string(out), nil
}

func upsertSkill(skills []string, target string, add bool) ([]string, bool) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return skills, false
	}
	exists := false
	for _, s := range skills {
		if s == trimmed {
			exists = true
			break
		}
	}
	if add {
		if exists {
			return skills, false
		}
		return append(skills, trimmed), true
	}
	if !exists {
		return skills, false
	}
	next := make([]string, 0, len(skills)-1)
	for _, s := range skills {
		if s != trimmed {
			next = append(next, s)
		}
	}
	return next, true
}
