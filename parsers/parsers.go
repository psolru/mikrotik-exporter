package parsers

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	durationRegex     = regexp.MustCompile(`(?:(\d*)w)?(?:(\d*)d)?(?:(\d*)h)?(?:(\d*)m)?(?:(\d*)s)?(?:(\d*)ms)?`)
	durationParts     = [6]time.Duration{time.Hour * 168, time.Hour * 24, time.Hour, time.Minute, time.Second, time.Millisecond}
	wirelessRateRegex = regexp.MustCompile(`([\d.]+)Mbps.*`)

	errUnexpectedPartsCount  = errors.New("unexpected parts count after split")
	errUnexpectedRegexResult = errors.New("unexpected regex result")
)

const datetimeFormat = "Jan/02/2006 15:04:05"

func ParseCommaSeparatedValuesToFloat64(metric string) (float64, float64, error) {
	strs := strings.Split(metric, ",")
	if len(strs) == 0 || len(strs) < 2 {
		return 0, 0, errUnexpectedPartsCount
	}

	m1, err := strconv.ParseFloat(strs[0], 64)
	if err != nil {
		return math.NaN(), math.NaN(), err
	}

	m2, err := strconv.ParseFloat(strs[1], 64)
	if err != nil {
		return math.NaN(), math.NaN(), err
	}

	return m1, m2, nil
}

func ParseDuration(duration string) (float64, error) {
	reMatch := durationRegex.FindAllStringSubmatch(duration, -1)

	// should get one and only one match back on the regex
	if len(reMatch) != 1 {
		return 0, errUnexpectedRegexResult
	}

	var d time.Duration
	for i, match := range reMatch[0] {
		if len(match) == 0 || i == 0 {
			continue
		}

		v, err := strconv.Atoi(match)
		if err != nil {
			log.WithFields(log.Fields{
				"duration": duration,
				"value":    match,
				"error":    err,
			}).Error("failed to parse duration field value")
			return 0, err
		}

		d += time.Duration(v) * durationParts[i-1]
	}

	return d.Seconds(), nil
}

func ParseDatetime(datetime string) (time.Time, error) {
	t, err := time.Parse(datetimeFormat, datetime)
	if err != nil {
		log.WithFields(log.Fields{
			"datetime": datetime,
			"value":    t,
			"error":    err,
		}).Error("failed to parse datetime field value")
		return time.Time{}, err
	}

	return t, nil
}

func ParseWirelessRate(rate string) (float64, error) {
	reMatch := wirelessRateRegex.FindStringSubmatch(rate)

	// should get one and only one match back on the regex
	if len(reMatch) != 2 {
		return 0, errUnexpectedRegexResult
	}

	if len(reMatch[1]) == 0 {
		return 0, nil
	}

	v, err := strconv.ParseFloat(reMatch[1], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"wireless-rate": rate,
			"value":         reMatch[1],
			"error":         err,
		}).Error("failed to parse wireless rate field value")
		return 0, err
	}

	return v, nil
}
