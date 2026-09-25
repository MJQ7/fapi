package cli

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// form is a list of labelled text fields, one of them focused (receiving
// typing). The text fields are Bubbles' textinput components.
type form struct {
	title   string
	labels  []string
	inputs  []textinput.Model
	focused int
	err     string // fapi's message after a failed submit
	saving  bool   // submitted, waiting for fapi
}

// formField describes one field of a new form.
type formField struct {
	label       string
	value       string
	placeholder string
}

func newForm(title string, fields ...formField) form {
	newForm := form{title: title}
	for _, field := range fields {
		input := textinput.New()
		input.Prompt = ""
		input.Placeholder = field.placeholder
		input.SetValue(field.value)
		input.SetWidth(80) // wide enough for the sample response bodies
		newForm.labels = append(newForm.labels, field.label)
		newForm.inputs = append(newForm.inputs, input)
	}
	return newForm
}

// value returns the text in field index, without surrounding spaces.
func (f *form) value(index int) string {
	return strings.TrimSpace(f.inputs[index].Value())
}

// focusField moves the focus to field index. It returns the command that
// makes the cursor blink.
func (f *form) focusField(index int) tea.Cmd {
	f.inputs[f.focused].Blur()
	f.focused = index
	return f.inputs[index].Focus()
}

// formKeyResult says what a key press did to the form.
type formKeyResult int

const (
	formEditing   formKeyResult = iota // still being filled in
	formSubmitted                      // enter on the last field
	formCancelled                      // esc
)

// handleKey moves between fields (tab, shift+tab, up, down, enter), submits
// with enter on the last field, cancels with esc, and passes any other key
// to the focused field.
func (f *form) handleKey(key tea.KeyPressMsg) (formKeyResult, tea.Cmd) {
	last := len(f.inputs) - 1

	switch key.String() {
	case "esc":
		return formCancelled, nil
	case "tab", "down":
		return formEditing, f.focusField(min(f.focused+1, last))
	case "shift+tab", "up":
		return formEditing, f.focusField(max(f.focused-1, 0))
	case "enter":
		if f.focused == last {
			return formSubmitted, nil
		}
		return formEditing, f.focusField(f.focused + 1)
	}
	return formEditing, f.updateInput(key)
}

// updateInput passes a message (a key, or the cursor's blink timer) to the
// focused field.
func (f *form) updateInput(msg tea.Msg) tea.Cmd {
	var command tea.Cmd
	f.inputs[f.focused], command = f.inputs[f.focused].Update(msg)
	return command
}

// view shows the form's fields, its error and its keys.
func (f *form) view() string {
	var view strings.Builder
	view.WriteString(titleStyle.Render(f.title) + "\n\n")

	for index, input := range f.inputs {
		label := f.labels[index]
		if index == f.focused {
			label = selectedStyle.Render(label)
		}
		view.WriteString(label + "\n  " + input.View() + "\n\n")
	}

	if f.saving {
		view.WriteString(faintStyle.Render("Saving…") + "\n\n")
	}
	if f.err != "" {
		view.WriteString(errorStyle.Render(f.err) + "\n\n")
	}
	view.WriteString(faintStyle.Render("tab/↑/↓ move between fields · enter next field, or save on the last · esc cancel"))
	return view.String()
}
