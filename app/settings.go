package app

type FontSize float32

const (
	FontSmall  FontSize = 16
	FontMedium FontSize = 24
	FontLarge  FontSize = 32
)

type Settings struct {
	// Use foundations as unit for the grid, instead of world units (meters)
	TickFoundation bool
	// Font size, for text in the scene
	FontSize FontSize
}

var settings = Settings{
	TickFoundation: true,
	FontSize:       FontMedium,
}
