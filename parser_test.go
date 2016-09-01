package cron

import (
	"reflect"
	"testing"
	"time"
)

func TestRange(t *testing.T) {
	ranges := []struct {
		expr     string
		min, max uint
		expected uint64
	}{
		{"5", 0, 7, 1 << 5},
		{"0", 0, 7, 1 << 0},
		{"7", 0, 7, 1 << 7},

		{"5-5", 0, 7, 1 << 5},
		{"5-6", 0, 7, 1<<5 | 1<<6},
		{"5-7", 0, 7, 1<<5 | 1<<6 | 1<<7},

		{"5-6/2", 0, 7, 1 << 5},
		{"5-7/2", 0, 7, 1<<5 | 1<<7},
		{"5-7/1", 0, 7, 1<<5 | 1<<6 | 1<<7},

		{"*", 1, 3, 1<<1 | 1<<2 | 1<<3 | starBit},
		{"*/2", 1, 3, 1<<1 | 1<<3 | starBit},
	}

	for _, c := range ranges {
		actual := getRange(c.expr, bounds{c.min, c.max, nil})
		if actual != c.expected {
			t.Errorf("%s => (expected) %d != %d (actual)", c.expr, c.expected, actual)
		}
	}
}

func TestField(t *testing.T) {
	fields := []struct {
		expr     string
		min, max uint
		expected uint64
	}{
		{"5", 1, 7, 1 << 5},
		{"5,6", 1, 7, 1<<5 | 1<<6},
		{"5,6,7", 1, 7, 1<<5 | 1<<6 | 1<<7},
		{"1,5-7/2,3", 1, 7, 1<<1 | 1<<5 | 1<<7 | 1<<3},
	}

	for _, c := range fields {
		actual := getField(c.expr, bounds{c.min, c.max, nil})
		if actual != c.expected {
			t.Errorf("%s => (expected) %d != %d (actual)", c.expr, c.expected, actual)
		}
	}
}

func TestBits(t *testing.T) {
	allBits := []struct {
		r        bounds
		expected uint64
	}{
		{minutes, 0xfffffffffffffff | starBit}, // 0-59: 60 ones
		{hours, 0xffffff | starBit},            // 0-23: 24 ones
		{dom, 0xfffffffe | starBit},            // 1-31: 31 ones, 1 zero
		{months, 0x1ffe | starBit},             // 1-12: 12 ones, 1 zero
		{dow, 0x7f | starBit},                  // 0-6: 7 ones
	}

	for _, c := range allBits {
		actual := all(c.r)
		if c.expected != actual {
			t.Errorf("%d-%d/%d => (expected) %b != %b (actual)",
				c.r.min, c.r.max, 1, c.expected, actual)
		}
	}

	bits := []struct {
		min, max, step uint
		expected       uint64
	}{
		{0, 0, 1, 0x1},
		{1, 1, 1, 0x2},
		{1, 5, 2, 0x2a}, // 101010
		{1, 4, 2, 0xa},  // 1010
	}

	for _, c := range bits {
		actual := getBits(c.min, c.max, c.step)
		if c.expected != actual {
			t.Errorf("%d-%d/%d => (expected) %b != %b (actual)",
				c.min, c.max, c.step, c.expected, actual)
		}
	}
}

func TestMultiBits(t *testing.T) {
	allBits := []struct {
		r        bounds
		expected []uint64
	}{
		{bounds{0, 99, nil}, []uint64{0xfffffffffffffff | starBit, 0xffffffffff}},
		{bounds{100, 199, nil}, []uint64{0xfffffffffffffff | starBit, 0xffffffffff}},
	}

	for _, c := range allBits {
		actual := mall(c.r)
		if !reflect.DeepEqual(c.expected, actual) {
			t.Errorf("%d-%d/%d => (expected) %b != %b (actual)",
				c.r.min, c.r.max, 1, c.expected, actual)
		}
	}

	b2 := bounds{0, 119, nil}
	b3 := bounds{0, 179, nil}

	bits := []struct {
		r              bounds
		min, max, step uint
		expected       []uint64
	}{
		{b2, 0, 59, 1, []uint64{0xfffffffffffffff, 0}},
		{b2, 0, 60, 1, []uint64{0xfffffffffffffff, 0x1}},
		{b2, 0, 61, 1, []uint64{0xfffffffffffffff, 0x3}},
		{b2, 0, 62, 1, []uint64{0xfffffffffffffff, 0x7}},
		{b2, 0, 99, 1, []uint64{0xfffffffffffffff, 0xffffffffff}},
		{b2, 0, 99, 2, []uint64{0x555555555555555, 0x5555555555}},
		{b2, 0, 119, 1, []uint64{0xfffffffffffffff, 0xfffffffffffffff}},
		{b2, 0, 119, 2, []uint64{0x555555555555555, 0x555555555555555}},
		{b2, 60, 60, 1, []uint64{0, 0x1}},
		{b2, 60, 61, 1, []uint64{0, 0x3}},
		{b2, 60, 62, 1, []uint64{0, 0x7}},
		{b2, 60, 99, 1, []uint64{0, 0xffffffffff}},
		{b2, 60, 99, 2, []uint64{0, 0x5555555555}},
		{b2, 60, 119, 1, []uint64{0, 0xfffffffffffffff}},
		{b2, 60, 119, 2, []uint64{0, 0x555555555555555}},
		{b3, 60, 120, 1, []uint64{0, 0xfffffffffffffff, 0x1}},
		{b3, 120, 120, 1, []uint64{0, 0, 0x1}},
		{b3, 0, 120, 1, []uint64{0xfffffffffffffff, 0xfffffffffffffff, 0x1}},
		{b3, 40, 140, 1, []uint64{0xfffff0000000000, 0xfffffffffffffff, 0x1fffff}},
	}

	for _, c := range bits {
		actual := getMultiBits(c.r, c.min, c.max, c.step)
		if !reflect.DeepEqual(c.expected, actual) {
			t.Errorf("%d-%d/%d => (expected) %x != %x (actual)",
				c.min, c.max, c.step, c.expected, actual)
		}
	}

	b99 := bounds{0, 99, nil}
	evenBits := getMultiBits(b99, 0, 99, 2)
	oddBits := getMultiBits(b99, 1, 99, 2)

	for i := 0; i <= 99; i += 2 {
		if mhas(b99, evenBits, i) != true {
			t.Errorf("0-99/2 expected mhas %d to be true", i)
		}
		if mhas(b99, evenBits, i+1) != false {
			t.Errorf("0-99/2 expected mhas %d to be false", i+1)
		}
	}

	for i := 0; i <= 99; i += 2 {
		if mhas(b99, oddBits, i) != false {
			t.Errorf("1-99/2 expected mhas %d to be false", i)
		}
		if mhas(b99, oddBits, i+1) != true {
			t.Errorf("1-99/2 expected mhas %d to be true", i+1)
		}
	}
}

func TestSpecSchedule(t *testing.T) {
	entries := []struct {
		expr     string
		expected Schedule
	}{
		{"* 5 * * * *", &SpecSchedule{all(seconds), 1 << 5, all(hours), all(dom), all(months), all(dow), all(weeksOfYear), mall(years)}},
		{"@every 5m", ConstantDelaySchedule{time.Duration(5) * time.Minute}},
	}

	for _, c := range entries {
		actual, err := Parse(c.expr)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("%s => (expected) %b != %b (actual)", c.expr, c.expected, actual)
		}
	}
}
