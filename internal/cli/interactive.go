// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package cli provides interactive command-line interface functionality for Karei.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/databases"
	"github.com/janderssonse/karei/internal/patterns"
)

const (
	// ManagerTypeTheme represents theme management operations.
	ManagerTypeTheme = "theme"
	// ManagerTypeFont represents font management operations.
	ManagerTypeFont = "font"
)

func getTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
}

func getHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)
}

// InteractiveSetup holds user selections from interactive setup.
type InteractiveSetup struct {
	Theme     string
	Font      string
	Apps      []string
	Languages []string
	Databases []string
	Groups    []string
	Confirmed bool
}

func (app *CLI) runFirstTimeSetup(ctx context.Context) error {
	fmt.Print(getTitleStyle().Render("◈ Welcome to Karei! ◈"))
	fmt.Println()
	fmt.Println("Let's set up your beautiful Ubuntu desktop...")
	fmt.Println()

	setup := &InteractiveSetup{}

	// Theme selection
	themeForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("◈ Choose your theme").
				Description("This will style your entire desktop").
				Options(
					huh.NewOption("▪ Tokyo Night", "tokyo-night"),
					huh.NewOption("▫ Catppuccin", "catppuccin"),
					huh.NewOption("◦ Nord", "nord"),
					huh.NewOption("▸ Everforest", "everforest"),
					huh.NewOption("■ Gruvbox Dark", "gruvbox"),
					huh.NewOption("□ Gruvbox Light", "gruvbox-light"),
					huh.NewOption("◈ Rose Pine", "rose-pine"),
					huh.NewOption("◉ Kanagawa", "kanagawa"),
				).
				Value(&setup.Theme),
		),
	)

	if err := themeForm.Run(); err != nil {
		return err
	}

	// Font selection
	fontForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("◈ Choose your coding font").
				Description("For terminal and code editor").
				Options(
					huh.NewOption("▸ CaskaydiaMono Nerd Font", "CaskaydiaMono"),
					huh.NewOption("▪ FiraMono Nerd Font", "FiraMono"),
					huh.NewOption("◈ JetBrainsMono Nerd Font", "JetBrainsMono"),
					huh.NewOption("▫ MesloLGS Nerd Font", "MesloLGS"),
					huh.NewOption("◉ Berkeley Mono", "BerkeleyMono"),
				).
				Value(&setup.Font),
		),
	)

	if err := fontForm.Run(); err != nil {
		return err
	}

	// App groups selection
	groupForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("⬛ Select app groups to install").
				Description("Choose categories of apps you want").
				Options(
					huh.NewOption("▸ Development Tools", "development"),
					huh.NewOption("◦ Browsers", "browsers"),
					huh.NewOption("◈ Communication", "communication"),
					huh.NewOption("▫ Media Apps", "media"),
					huh.NewOption("▪ Productivity", "productivity"),
					huh.NewOption("◉ Graphics Tools", "graphics"),
					huh.NewOption("■ Utilities", "utilities"),
					huh.NewOption("□ Terminal Tools", "terminal"),
				).
				Value(&setup.Groups),
		),
	)

	if err := groupForm.Run(); err != nil {
		return err
	}

	// Language selection
	langForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("▸ Select programming languages").
				Description("Languages to install via mise").
				Options(
					huh.NewOption("▪ Node.js", "nodejs"),
					huh.NewOption("▫ Python", "python"),
					huh.NewOption("◦ Go", "golang"),
					huh.NewOption("◈ Ruby", "ruby"),
					huh.NewOption("▸ Rust", "rust"),
					huh.NewOption("◉ Elixir", "elixir"),
					huh.NewOption("■ Java", "java"),
				).
				Value(&setup.Languages),
		),
	)

	if err := langForm.Run(); err != nil {
		return err
	}

	// Database selection
	dbForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("◦ Select databases").
				Description("Databases to run in Docker containers").
				Options(
					huh.NewOption("▸ MySQL", "mysql"),
					huh.NewOption("▪ Redis", "redis"),
					huh.NewOption("◉ PostgreSQL", "postgresql"),
				).
				Value(&setup.Databases),
		),
	)

	if err := dbForm.Run(); err != nil {
		return err
	}

	// Confirmation
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("▸ Ready to transform your system?").
				Description(fmt.Sprintf(
					"Theme: %s\nFont: %s\nGroups: %s\nLanguages: %s\nDatabases: %s",
					setup.Theme,
					setup.Font,
					strings.Join(setup.Groups, ", "),
					strings.Join(setup.Languages, ", "),
					strings.Join(setup.Databases, ", "),
				)).
				Value(&setup.Confirmed),
		),
	)

	if err := confirmForm.Run(); err != nil {
		return err
	}

	if !setup.Confirmed {
		fmt.Println("Setup cancelled.")

		return nil
	}

	return app.executeSetup(ctx, setup)
}

func (app *CLI) executeSetup(ctx context.Context, setup *InteractiveSetup) error {
	fmt.Print(getHeaderStyle().Render("▸ Installing your beautiful desktop..."))
	fmt.Println()

	app.applyTheme(ctx, setup.Theme)
	app.applyFont(ctx, setup.Font)
	app.installAppGroups(ctx, setup.Groups)
	app.installLanguages(ctx, setup.Languages)
	app.installDatabases(ctx, setup.Databases)

	fmt.Println()
	fmt.Print(getHeaderStyle().Render("✓ Karei setup complete! Enjoy your beautiful desktop!"))
	fmt.Println()

	return nil
}

func (app *CLI) applyTheme(ctx context.Context, theme string) {
	if theme == "" {
		return
	}

	fmt.Printf("◈ Applying theme: %s\n", theme)

	themeManager := patterns.NewThemeManager(app.verbose)
	if err := themeManager.Apply(ctx, theme); err != nil {
		fmt.Printf("⚠ Theme error: %v\n", err)
	}
}

func (app *CLI) applyFont(ctx context.Context, font string) {
	if font == "" {
		return
	}

	fmt.Printf("◈ Installing font: %s\n", font)

	fontManager := patterns.NewFontManager(app.verbose)
	if err := fontManager.Apply(ctx, font); err != nil {
		fmt.Printf("⚠ Font error: %v\n", err)
	}
}

func (app *CLI) installAppGroups(ctx context.Context, groups []string) {
	if len(groups) == 0 {
		return
	}

	appManager := apps.NewManager(app.verbose)

	for _, group := range groups {
		fmt.Printf("⬛ Installing %s apps...\n", group)

		if err := appManager.InstallGroup(ctx, group); err != nil {
			fmt.Printf("⚠ Group %s error: %v\n", group, err)
		}
	}
}

func (app *CLI) installLanguages(ctx context.Context, languages []string) {
	if len(languages) == 0 {
		return
	}

	appManager := apps.NewManager(app.verbose)

	for _, lang := range languages {
		fmt.Printf("▸ Installing %s...\n", lang)

		if err := appManager.InstallLanguage(ctx, lang, "latest"); err != nil {
			fmt.Printf("⚠ Language %s error: %v\n", lang, err)
		}
	}
}

func (app *CLI) installDatabases(ctx context.Context, databaseList []string) {
	if len(databaseList) == 0 {
		return
	}

	dbManager := databases.NewManager(app.verbose)

	fmt.Printf("◦ Installing databases...\n")

	if err := dbManager.InstallDatabases(ctx, databaseList); err != nil {
		fmt.Printf("⚠ Database error: %v\n", err)
	}
}

func (app *CLI) runAppSelector(ctx context.Context) error {
	var selectedApps []string

	// Get all available apps
	allApps := apps.ListApps("")
	options := make([]huh.Option[string], len(allApps))

	for i, appItem := range allApps {
		emoji := app.selectEmojiForGroup(appItem.Group)
		options[i] = huh.NewOption(
			fmt.Sprintf("%s %s", emoji, appItem.Name),
			appItem.Name,
		)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("⬛ Select apps to install").
				Description("Choose individual apps to install").
				Options(options...).
				Value(&selectedApps),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if len(selectedApps) == 0 {
		fmt.Println("No apps selected.")

		return nil
	}

	var confirmed bool

	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("▸ Install selected apps?").
				Description("Installing: " + strings.Join(selectedApps, ", ")).
				Value(&confirmed),
		),
	)

	if err := confirmForm.Run(); err != nil {
		return err
	}

	if !confirmed {
		fmt.Println("Installation cancelled.")

		return nil
	}

	// Install selected apps
	appManager := apps.NewManager(app.verbose)

	for _, appName := range selectedApps {
		fmt.Printf("⬛ Installing %s...\n", appName)

		if err := appManager.InstallApp(ctx, appName); err != nil {
			fmt.Printf("⚠ Failed to install %s\n", appName)
		}
	}

	fmt.Println("✓ App installation complete!")

	return nil
}

func (app *CLI) selectEmojiForGroup(group string) string {
	symbols := map[string]string{
		"development":   "▸",
		"browsers":      "◦",
		"communication": "◈",
		"media":         "▫",
		"productivity":  "▪",
		"graphics":      "◉",
		"utilities":     "■",
		"terminal":      "□",
	}

	if symbol, exists := symbols[group]; exists {
		return symbol
	}

	return "⬛"
}

func (app *CLI) createUniversalManager(managerType string, verbose bool) any {
	switch managerType {
	case ManagerTypeTheme:
		return app.getThemeManager(verbose)
	case ManagerTypeFont:
		return app.getFontManager(verbose)
	default:
		return nil
	}
}

func (app *CLI) getThemeManager(_ bool) any {
	// Return theme manager - simplified for now
	return struct {
		Apply func(string) error
	}{
		Apply: func(theme string) error {
			fmt.Printf("Applied theme: %s\n", theme)

			return nil
		},
	}
}

func (app *CLI) getFontManager(_ bool) any {
	// Return font manager - simplified for now
	return struct {
		Apply func(string) error
	}{
		Apply: func(font string) error {
			fmt.Printf("Applied font: %s\n", font)

			return nil
		},
	}
}
