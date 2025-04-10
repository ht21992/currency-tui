package main

import (
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

var (
	infoColor      = color.New(color.FgCyan).Add(color.Bold)
	errorColor     = color.New(color.FgRed).Add(color.Bold)
	successColor   = color.New(color.FgGreen).Add(color.Bold)
	whiteColor     = color.New(color.FgWhite).Add(color.Bold)
	selectionColor = color.New(color.FgHiMagenta).Add(color.Bold)
)

type model struct {
	choices     []string         // available currencies
	menuOptions []string         // available menu options
	cursor      int              // which currency is selected
	selected    map[int]struct{} // which currencies are selected
	amount      string           // amount to convert
	floatAmount float64          // float amount
	from        string           // currency to convert from
	to          string           // currency to convert to
	results     string           // conversion results
	state       string           // current state (amount, from, to, results)
	err         error            // any errors
	loading     bool             // loading
}

func initialModel() model {
	return model{
		choices:     []string{},
		selected:    make(map[int]struct{}),
		state:       "menu",
		loading:     false,
		menuOptions: []string{"Conversion", "List", "Quit"},
		amount:      "",
		floatAmount: 0.0,
		cursor:      0,
		from:        "",
		to:          "",
		results:     "",
	}
}

type currencyListMsg struct {
	Currencies []string
	Err        error
}

type menu struct {
	menuOptions []string
}

func (m model) Init() tea.Cmd {
	return mainMenu()
}

func mainMenu() tea.Cmd {
	return func() tea.Msg {
		options := []string{"Conversion", "List", "Quit"}
		return menu{menuOptions: options}
	}
}

func loadCurrencies() tea.Cmd {
	return func() tea.Msg {
		currencies, err := fetchSupportedCurrencies("USD") // default base
		return currencyListMsg{currencies, err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case menu:
		m.state = "menu"
		m.menuOptions = msg.menuOptions

	case currencyListMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.choices = msg.Currencies
		m.state = "amount"
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {

		case "backspace":
			if m.state == "list" {
				m.state = "menu"
				m.cursor = 0
				return m, nil
			}
			if m.state == "amount" {
				m.amount = trimLastChar(m.amount)
				m.cursor = 0
				return m, nil
			}

		case "ctrl+c", "esc":
			return m, tea.Quit

		case "up":
			if m.state == "menu" {
				if m.cursor > 0 {
					m.cursor--
				}
			} else if m.cursor >= 10 {
				m.cursor -= 10
			}

		case "down":
			if m.state == "menu" {
				if m.cursor < len(m.menuOptions)-1 {
					m.cursor++
				}
			} else if m.cursor+10 < len(m.choices) {
				m.cursor += 10
			}

		case "left":
			if m.cursor > 0 {
				m.cursor--
			}

		case "right":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			switch m.state {

			case "menu":
				switch m.cursor {

				case 0: // Conversion
					m.state = "amount"
					m.cursor = 0
					return m, loadCurrencies()

				case 1: // Show list
					m.state = "list"
					m.cursor = 0
					return m, nil

				case 2: // Quit
					return m, tea.Quit
				} // end of switch m.cursor
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
	clearScreen := "\033[H\033[2J" // ANSI escape code to clear the terminal
	s := clearScreen + "\nCurrency Converter\n\n"

	if m.loading {
		s += "Loading supported currencies...\n"
		return s

	}

	if m.err != nil {
		s += errorColor.Sprintf("Error: %v\n", m.err)
		s += "Press esc or ctr + c to quit\n"
		return s
	}

	switch m.state {
	case "list":
		s += "Available Currencies:\n\n"
		var currencies []string
		favorites := []string{"USD", "EUR", "GBP"}
		currencies, _ = getCurrenciesFromRedis("supported_currencies")

		if len(currencies) > 0 {
			currencies = sortCurrencies(currencies, favorites)
			for i, currency := range currencies {
				s += whiteColor.Sprintf("%-4s", currency)
				// Insert a newline after every 10 items
				if (i+1)%10 == 0 {
					s += "\n"
				}
			}
		} else {
			s += errorColor.Sprintf("Nothing found in cache. Try to use conversion at least once\n")
		}
		s += infoColor.Sprintf("\n\nPress backspace button to go back\n")

	case "menu":
		s += "Select an option:\n\n"
		for i, option := range m.menuOptions {
			if m.cursor == i {
				s += selectionColor.Sprintf("> %s\n", option)
			} else {
				s += whiteColor.Sprintf("  %s\n", option)
			}
		}

	case "amount":
		if m.amount == "" {
			s += clearScreen + "Enter amount to convert: " + infoColor.Sprintf("\n\nPress enter when done\n")
		} else {
			s += "\n" + m.amount + "\n\n"
		}
	case "from":
		s += successColor.Sprintf("Amount to convert: %s\n\n", m.amount)
		s += "Select currency to convert FROM:\n\n"
		for i, choice := range m.choices {
			if m.cursor == i {
				s += selectionColor.Sprintf("> %-4s ", choice)
			} else {
				s += infoColor.Sprintf("  %-4s ", choice)
			}
			// Insert a newline after every 10 items
			if (i+1)%10 == 0 {
				s += "\n"
			}

		}

	case "to":
		s += successColor.Sprintf("Amount: %s\nFrom: %s\n\n", m.amount, m.from)
		s += "Select currency to convert TO:\n\n"
		for i, choice := range m.choices {
			if m.cursor == i {
				s += selectionColor.Sprintf("> %-4s ", choice)
			} else {
				s += infoColor.Sprintf("  %-4s ", choice)
			}
			// Insert a newline after every 10 items
			if (i+1)%10 == 0 {
				s += "\n"
			}

		}

	case "results":
		s += m.results + "\n\n"
		s += infoColor.Sprintf("Press enter to start over\n")
	}

	s += "\n\nPress esc or ctrl + c to quit.\n"

	return s
}

func main() {
	InitRedis()
	// Initialize our application
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		errorColor.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
