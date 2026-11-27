package domain

import "fmt"

// SeriesID defines stable identities shared by producers and legacy decoding.
func SeriesID(mode Mode, index int) string {
	names := []string{"series-1", "series-2", "series-3", "series-4", "series-5", "series-6"}
	if mode == ModeOscillator {
		names = []string{"position", "velocity", "acceleration", "kinetic-energy", "potential-energy", "total-energy"}
	}
	if mode == ModePreparation {
		names = []string{"p", "t", "v", "l", "psi", "g", "e", "z"}
	}
	if index >= 0 && index < len(names) {
		return string(mode) + ":" + names[index]
	}
	return fmt.Sprintf("%s:series-%d", mode, index+1)
}

// BindSeries migrates legacy positional preferences only when an ID has no
// saved preference. New preferences follow the series identity across ordering.
func (s *Settings) BindSeries(series []Series) {
	if s.Series == nil {
		s.Series = make(map[string]SeriesSettings)
	}
	for i, item := range series {
		if _, exists := s.Series[item.ID]; !exists {
			pref := SeriesSettings{Visible: true, Scale: 1}
			if i < len(s.Visible) {
				pref.Visible, pref.Scale = s.Visible[i], s.Scales[i]
			}
			s.Series[item.ID] = pref
		}
	}
}
