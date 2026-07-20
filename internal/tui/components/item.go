// Package components holds the reusable UI pieces the views compose with:
// menus, the command palette, modals, forms, confirmations, toasts, the status
// bar and the help overlay.
//
// Components own presentation and local cursor state only. They never call the
// service and never emit domain intents — the owning view reads their state
// (Current, Answer, Values) and emits the intent. That keeps every component
// reusable and keeps intent dispatch in one place.
package components

// Item is the generic row type shared by the menu and the palette, so an
// action defined once renders identically in both.
type Item struct {
	ID    string
	Title string
	Desc  string
	// Group is the section header this item sits under.
	Group string
	// Key is the direct binding hint shown on the right, if any.
	Key string
	// Badge renders after the title, e.g. "PRO".
	Badge string
	// Disabled items are shown but cannot be selected — discoverability
	// without a broken screen.
	Disabled bool
}
