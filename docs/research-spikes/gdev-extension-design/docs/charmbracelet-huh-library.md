<!-- Source: https://pkg.go.dev/github.com/charmbracelet/huh/v2 -->
<!-- Retrieved: 2026-05-12 -->

# Huh Library Documentation

## Overview

**Huh** is a powerful Go library for building interactive terminal-based forms and prompts. It's built on [Bubble Tea](https://github.com/charmbracelet/bubbletea) and provides both standalone mode and first-class Bubble Tea integration.

**Key Features:**
- Simple, intuitive API for building forms
- Multiple field types (Input, Text, Select, MultiSelect, Confirm, FilePicker, Note)
- Form validation and error handling
- Dynamic forms that adapt based on previous answers
- Accessibility mode for screen readers
- Multiple built-in themes (Charm, Dracula, Catppuccin, Base16, Default)
- Standalone field execution
- Full Bubble Tea integration

---

## Core Concepts

### Forms, Groups, and Fields

Forms are built hierarchically:
- **Form**: Top-level container managing the entire interaction
- **Group**: Logical sections within a form (like "pages")
- **Field**: Individual input controls (Input, Select, Confirm, etc.)

```go
form := huh.NewForm(
    huh.NewGroup(
        huh.NewSelect[string]().
            Title("Choose your burger").
            Options(...).
            Value(&burger),
    ),
    huh.NewGroup(
        huh.NewInput().
            Title("What's your name?").
            Value(&name),
    ),
)

err := form.Run()
```

---

## Field Types

### 1. Input - Single Line Text

```go
huh.NewInput().
    Title("What's for lunch?").
    Prompt("?").
    Placeholder("e.g., Pizza").
    Value(&lunch).
    Validate(isFood)
```

**Methods:**
- `Title(string)` / `TitleFunc(func() string, binding)`
- `Prompt(string)` - Input prompt character
- `Placeholder(string)` / `PlaceholderFunc(...)`
- `CharLimit(int)` - Character limit
- `EchoMode(EchoMode)` - Normal, Password, or None
- `Suggestions([]string)` / `SuggestionsFunc(...)`
- `Value(*string)` - Store result
- `Validate(func(string) error)`

### 2. Text - Multi-line Text

```go
huh.NewText().
    Title("Tell me a story.").
    Lines(10).
    CharLimit(400).
    Value(&story).
    Validate(checkForPlagiarism)
```

### 3. Select - Single Choice

```go
huh.NewSelect[string]().
    Title("Pick a country.").
    Options(
        huh.NewOption("United States", "US"),
        huh.NewOption("Germany", "DE"),
    ).
    Value(&country)
```

### 4. MultiSelect - Multiple Choices

```go
huh.NewMultiSelect[string]().
    Title("Toppings").
    Options(
        huh.NewOption("Lettuce", "lettuce").Selected(true),
        huh.NewOption("Tomatoes", "tomatoes").Selected(true),
        huh.NewOption("Cheese", "cheese"),
    ).
    Limit(4).
    Value(&toppings)
```

### 5. Confirm - Yes/No

```go
huh.NewConfirm().
    Title("Are you sure?").
    Affirmative("Yes!").
    Negative("No.").
    Value(&confirm)
```

### 6. FilePicker - File Selection

```go
huh.NewFilePicker().
    Title("Choose a file").
    CurrentDirectory(".").
    AllowedTypes([]string{".txt", ".md"}).
    Value(&filePath)
```

### 7. Note - Read-only Display

```go
huh.NewNote().
    Title("Important Notice").
    Description("This is important information").
    Next(true).
    NextLabel("Continue")
```

---

## Validation

### Built-in Validators

```go
field.Validate(huh.ValidateNotEmpty())
field.Validate(huh.ValidateMinLength(5))
field.Validate(huh.ValidateMaxLength(50))
field.Validate(huh.ValidateLength(5, 50))
field.Validate(huh.ValidateOneOf("option1", "option2"))
```

### Custom Validation

```go
huh.NewInput().
    Title("What's your name?").
    Value(&name).
    Validate(func(str string) error {
        if str == "Frank" {
            return errors.New("Sorry, we don't serve that name.")
        }
        return nil
    })
```

---

## Dynamic Forms

Create forms that change based on user input using `Func` variants:

```go
var country string
var state string

form := huh.NewForm(
    huh.NewGroup(
        huh.NewSelect[string]().
            Title("Country").
            Options(huh.NewOptions("United States", "Canada")...).
            Value(&country),
        
        huh.NewSelect[string]().
            TitleFunc(func() string {
                if country == "Canada" {
                    return "Province"
                }
                return "State"
            }, &country).
            OptionsFunc(func() []huh.Option[string] {
                opts := fetchStatesForCountry(country)
                return huh.NewOptions(opts...)
            }, &country).
            Value(&state),
    ),
)
```

**Dynamic method variants:**
- `TitleFunc(func() string, binding)` instead of `Title(string)`
- `DescriptionFunc(func() string, binding)` instead of `Description(string)`
- `PlaceholderFunc(func() string, binding)` instead of `Placeholder(string)`
- `SuggestionsFunc(func() []string, binding)` instead of `Suggestions([]string)`
- `OptionsFunc(func() []Option[T], binding)` instead of `Options(...Option[T])`

---

## Accessibility Mode

Enable accessibility for screen reader compatibility:

```go
accessibleMode := os.Getenv("ACCESSIBLE") != ""
form.WithAccessible(accessibleMode)
```

**Features in accessible mode:**
- No TUI graphics
- Standard text prompts
- Better screen reader compatibility
- Simplified input/output

---

## Theming

### Built-in Themes

```go
form.WithTheme(huh.ThemeCharm(isDark))
form.WithTheme(huh.ThemeDracula(isDark))
form.WithTheme(huh.ThemeCatppuccin(isDark))
form.WithTheme(huh.ThemeBase16(isDark))
form.WithTheme(huh.ThemeBase())
```

---

## Form Configuration

```go
form.
    WithTheme(theme).
    WithKeyMap(keymap).
    WithHeight(30).
    WithWidth(80).
    WithLayout(huh.LayoutColumns(2)).
    WithShowErrors(true).
    WithShowHelp(true).
    WithAccessible(false).
    WithTimeout(30 * time.Second)
```

### Layout Options

```go
form.WithLayout(huh.LayoutColumns(1))  // Single column (default)
form.WithLayout(huh.LayoutColumns(2))  // Two columns
form.WithLayout(huh.LayoutGrid(rows, columns))  // Grid layout
```

---

## Bubble Tea Integration

Use forms as models in Bubble Tea applications:

```go
type Model struct {
    form *huh.Form
}

func (m Model) Init() tea.Cmd {
    return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    form, cmd := m.form.Update(msg)
    if f, ok := form.(*huh.Form); ok {
        m.form = f
    }
    return m, cmd
}

func (m Model) View() string {
    if m.form.State == huh.StateCompleted {
        class := m.form.GetString("class")
        return fmt.Sprintf("You selected: %s", class)
    }
    return m.form.View()
}
```

---

## Standalone Field Execution

```go
var name string
err := huh.NewInput().
    Title("What's your name?").
    Value(&name).
    Run()
```

---

## Bonus: Spinner

```go
err := spinner.New().
    Title("Making your burger...").
    Action(makeBurger).
    Run()
```

---

## Module Information

- **Version:** v2.0.0 (pre-release)
- **License:** MIT
- **Repository:** github.com/charmbracelet/huh
