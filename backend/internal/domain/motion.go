package domain

import "math"

// Время пути: разгон до максимальной скорости, затем равномерное движение. Торможение по условию мгновенное.
func travelTime(distance, speed, maximum, acceleration float64) float64 {
	distance = math.Abs(distance)
	t := math.Max(0, (maximum-speed)/acceleration)
	ramp := speed*t + acceleration*t*t/2
	if distance <= ramp {
		return (math.Sqrt(speed*speed+2*acceleration*distance) - speed) / acceleration
	}
	return t + (distance-ramp)/maximum
}

func travelDistance(dt, speed, maximum, acceleration float64) (float64, float64) {
	ramp := math.Min(dt, math.Max(0, (maximum-speed)/acceleration))
	d := speed*ramp + acceleration*ramp*ramp/2
	next := math.Min(maximum, speed+acceleration*ramp)
	return d + next*(dt-ramp), next
}

func sign(n float64) int {
	if n > 1e-8 {
		return 1
	}
	if n < -1e-8 {
		return -1
	}
	return 0
}

func (e *Elevator) beginDirection(direction int) {
	if direction == 0 {
		return
	}
	if e.Direction != direction {
		// Первая поездка и каждый разворот увеличивают счётчик; простой не сбрасывает направление.
		e.Trips++
		e.EmptyTrips++
		e.tripLoaded = false
		e.Direction = direction
		e.Speed = 0
	}
	if len(e.Passengers) > 0 {
		e.markLoaded()
	}
}

func (e *Elevator) markLoaded() {
	if e.Trips > 0 && !e.tripLoaded {
		// Пустой подъезд и движение с пассажирами в том же направлении — одна поездка, уже не холостая.
		e.EmptyTrips--
		e.tripLoaded = true
	}
}

func floorPosition(floor int) float64 { return float64(floor-1) * FloorHeight }
