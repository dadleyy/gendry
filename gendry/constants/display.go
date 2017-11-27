package constants

const (
	// GoodCoverageAmount is the float value that will determine when the display api renders green badges.
	GoodCoverageAmount = 80.00

	// DefaultShieldStyle is the style used in shield.io requests if the user has not provided one.
	DefaultShieldStyle = "flat-square"

	// ShieldConfigTemplate defines the string formatting used for shield text.
	ShieldConfigTemplate = "%s-%.2f%%-%s"

	// ShieldURLTemplate defines where the shields api lives.
	ShieldURLTemplate = "https://img.shields.io/badge/%s.svg"
)
