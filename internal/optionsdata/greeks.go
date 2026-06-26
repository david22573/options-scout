// Package optionsdata — Greeks helpers.
package optionsdata

// DeltaClassify returns the delta bucket for a contract.
// near-the-money = |delta| in [0.30, 0.60]
func DeltaClassify(c *Contract) string {
	d := c.Delta
	if d < 0 {
		d = -d
	}
	switch {
	case d >= 0.60:
		return "deep-itm"
	case d >= 0.30:
		return "near-atm"
	case d >= 0.10:
		return "otm"
	default:
		return "far-otm"
	}
}
