package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)
	itemStyle = lipgloss.NewStyle().PaddingLeft(4)
)

type commitType struct {
	title, desc string
}

func (c commitType) Title() string       { return c.title }
func (c commitType) Description() string { return c.desc }
func (c commitType) FilterValue() string { return c.title }

type model struct {
	stagedFiles  []string
	commitTypes  list.Model
	textInput    textinput.Model
	selectedType string
	state        int // 0: select type, 1: enter message, 2: confirm
	err          error
}

func getGitStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

func createCommit(commitType string, message string) error {
	fullMessage := fmt.Sprintf("%s: %s", commitType, message)
	cmd := exec.Command("git", "commit", "-m", fullMessage)
	return cmd.Run()
}

func initialModel() (model, error) {
	stagedFiles, err := getGitStagedFiles()
	if err != nil {
		return model{}, err
	}

	commitTypes := []list.Item{
		commitType{title: "feat", desc: "A new feature"},
		commitType{title: "fix", desc: "A bug fix"},
		commitType{title: "docs", desc: "Documentation only changes"},
		commitType{title: "style", desc: "Changes that do not affect the meaning of the code"},
		commitType{title: "refactor", desc: "A code change that neither fixes a bug nor adds a feature"},
		commitType{title: "perf", desc: "A code change that improves performance"},
		commitType{title: "test", desc: "Adding missing tests or correcting existing tests"},
		commitType{title: "chore", desc: "Changes to the build process or auxiliary tools"},
	}

	l := list.New(commitTypes, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select commit type"

	ti := textinput.New()
	ti.Placeholder = "Enter commit message"
	ti.Focus()
	ti.CharLimit = 80
	ti.Width = 60

	return model{
		stagedFiles: stagedFiles,
		commitTypes: l,
		textInput:   ti,
		state:       0,
	}, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			switch m.state {
			case 0: // select type
				if i, ok := m.commitTypes.SelectedItem().(commitType); ok {
					m.selectedType = i.title
					m.state = 1
				}
			case 1: // enter message
				if m.textInput.Value() != "" {
					m.state = 2
				}
			case 2: // confirm
				err := createCommit(m.selectedType, m.textInput.Value())
				if err != nil {
					m.err = err
					return m, tea.Quit
				}
				return m, tea.Quit
			}
		}
	}

	switch m.state {
	case 0:
		var cmd tea.Cmd
		m.commitTypes, cmd = m.commitTypes.Update(msg)
		return m, cmd
	case 1:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if len(m.stagedFiles) == 0 {
		return "No files staged for commit. Use 'git add' to stage files.\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var s string

	// Show staged files
	s += titleStyle.Render("Staged Files") + "\n"
	for _, file := range m.stagedFiles {
		s += itemStyle.Render(file) + "\n"
	}
	s += "\n"

	switch m.state {
	case 0:
		// Select commit type
		s += m.commitTypes.View()
	case 1:
		// Enter commit message
		s += titleStyle.Render("Commit Message") + "\n"
		s += fmt.Sprintf("Type: %s\n\n", m.selectedType)
		s += m.textInput.View()
	case 2:
		// Confirm
		s += titleStyle.Render("Confirm Commit") + "\n"
		s += fmt.Sprintf("Type: %s\n", m.selectedType)
		s += fmt.Sprintf("Message: %s\n\n", m.textInput.Value())
		s += "Press Enter to commit or q to quit"
	}

	return appStyle.Render(s)
}

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Printf("Error initializing: %v\n", err)
		os.Exit(1)
	}

	if len(m.stagedFiles) == 0 {
		fmt.Println("No files staged for commit. Use 'git add' to stage files.")
		os.Exit(0)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	if m.state == 2 {
		fmt.Println("Commit successful!")
	}
}

