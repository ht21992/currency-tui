package main

import (
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

var (
	infoColor    = color.New(color.FgCyan).Add(color.Bold)
	errorColor   = color.New(color.FgRed).Add(color.Bold)
	successColor = color.New(color.FgGreen).Add(color.Bold)
)

type model struct {
	choices     []string         // available currencies
	cursor      int              // which currency is selected
	selected    map[int]struct{} // which currencies are selected
	amount      string           // amount to convert
	floatAmount float64          // float amount
	from        string           // currency to convert from
	to          string           // currency to convert to
	results     string           // conversion results
	state       string           // current state (amount, from, to, results)
	err         error            // any errors
}

func initialModel() model {
	return model{
		choices:  []string{"USD", "EUR", "JPY", "GBP", "AUD", "CAD", "CHF", "CNY", "SEK", "NZD"},
		selected: make(map[int]struct{}),
		state:    "amount",
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now"
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			switch m.state {
			case "amount":
				value, err := strconv.ParseFloat(m.amount, 64)
				if err != nil {
					errorColor.Println("Invalid Amount")
					m.amount = ""
					m.state = "amount"
					m.cursor = 0
					m.floatAmount = 0.0
					return m, nil
				}
				m.floatAmount = float64(value)
				m.state = "from"

			case "from":
				m.from = m.choices[m.cursor]
				m.state = "to"
				m.cursor = 0 // Reset cursor for next selection
			case "to":
				m.to = m.choices[m.cursor]
				m.state = "results"

				rate, err := getExchangeRate(m.from, m.to)
				if err != nil {
					errorColor.Println(err)
					return m, nil
				}

				converted := m.floatAmount * rate
				m.results = successColor.Sprintf("\nConverted %s %s to %s: %.2f",
					m.amount, m.from, m.to, converted)

			case "results":
				// Reset to start over
				return initialModel(), nil
			}

		default:
			if m.state == "amount" {
				// Append the key pressed to the amount string
				m.amount += msg.String()
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Currency Converter\n\n"

	switch m.state {
	case "amount":
		s += "Enter amount to convert: " + m.amount + "\n\n"
		s += "Press enter when done\n"

	case "from":
		s += successColor.Sprintf("Amount to convert: %s\n\n", m.amount)
		s += "Select currency to convert FROM:\n\n"
		for i, choice := range m.choices {
			cursor := " " // no cursor
			if m.cursor == i {
				cursor = ">" // cursor!
			}
			s += infoColor.Sprintf("%s %s\n", cursor, choice)
		}

	case "to":
		s += successColor.Sprintf("Amount: %s\nFrom: %s\n\n", m.amount, m.from)
		s += "Select currency to convert TO:\n\n"
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s += infoColor.Sprintf("%s %s\n", cursor, choice)
		}

	case "results":
		s += m.results + "\n\n"
		s += "Press enter to start over\n"
	}

	s += "\nPress esc or ctrl + c to quit.\n"
	return s
}

func main() {
	// Initialize our application
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		errorColor.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
