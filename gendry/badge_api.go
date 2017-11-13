package gendry

const (
	defaultShieldStyle   = "flat-square"
	shieldConfigTemplate = "%s-%.2f%%-%s"
	shieldURLTemplate    = "https://img.shields.io/badge/%s.svg"
)

// NewBadgeAPI returns a new APIEndpoint capable of returning badge data from shields.io.
func NewBadgeAPI() APIEndpoint {
	return &badgeAPI{}
}

// BadgeAPI is resposnible for writing the svg badge result from shields.io given a report name.
type badgeAPI struct {
	notImplementedRoute
}
