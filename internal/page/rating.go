package page

import (
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

// Rating is what the page options carry. Value is an int under updown and a
// float under stars, and the difference reaches the frontend as written.
type Rating struct {
	Value      any
	Votes      int
	Popularity int
	Mode       string
}

func RatingOf(mode string, stats db.VoteStats) Rating {
	if mode == RatingModeDisabled {
		return Rating{Value: 0, Mode: mode}
	}
	good := stats.GoodUpDown
	value := any(int(stats.Sum))
	if mode == RatingModeStars {
		good = stats.GoodStars
		value = round1(stats.Average)
	}
	count := stats.Count
	if count == 0 {
		count = 1
	}
	return Rating{
		Value:      value,
		Votes:      stats.Count,
		Popularity: roundHalfEven(float64(good) / float64(count) * 100),
		Mode:       mode,
	}
}

// The exact binary value rounds to nearest with ties going to the even digit,
// so a number written as text and read back agrees.
func round1(x float64) float64 {
	rounded, err := strconv.ParseFloat(strconv.FormatFloat(x, 'f', 1, 64), 64)
	if err != nil {
		return x
	}
	return rounded
}

func DisabledRating() Rating {
	return Rating{Value: 0, Mode: RatingModeDisabled}
}
