package files

import (
	"bytes"
	"encoding/csv"
	"strconv"

	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
)

func Report(s domain.Session) []byte {
	var b bytes.Buffer
	b.WriteString("\xef\xbb\xbf")
	w := csv.NewWriter(&b)
	w.Comma = ';'
	f := func(n float64) string { return strconv.FormatFloat(n, 'f', 3, 64) }
	_ = w.Write([]string{"Показатель", "Значение"})
	rows := [][]string{
		{"Модельное время, с", f(s.Time)},
		{"Поездок всего", strconv.Itoa(s.Statistics.Trips)},
		{"Доставлено людей", strconv.Itoa(s.Statistics.Transported)},
		{"Завершённых ожиданий", strconv.Itoa(s.Statistics.WaitCount)},
		{"Среднее ожидание, с", f(s.Statistics.WaitAverage)},
		{"Максимальное ожидание, с", f(s.Statistics.WaitMax)},
		{"Суммарное ожидание, с", f(s.Statistics.WaitSum)},
		{"Пожарных тревог", strconv.Itoa(s.Statistics.FireCount)},
		{"Длительность тревог, с", f(s.Statistics.FireDuration)},
	}
	_ = w.WriteAll(rows)
	_ = w.Write([]string{})
	_ = w.Write([]string{"Лифт", "Поездки", "Холостые поездки", "Холостые, %", "Доставлено"})
	for _, e := range s.Elevators {
		percent := 0.0
		if e.Trips > 0 {
			percent = 100 * float64(e.EmptyTrips) / float64(e.Trips)
		}
		_ = w.Write([]string{strconv.Itoa(e.ID), strconv.Itoa(e.Trips), strconv.Itoa(e.EmptyTrips), f(percent), strconv.Itoa(e.Transported)})
	}
	w.Flush()
	return b.Bytes()
}
