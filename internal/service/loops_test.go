package service

import "testing"

func TestHexToRGB(t *testing.T) {

	tests := []struct {
		colorHex string
		want     [3]int
	}{
		{"FF0000", [3]int{255, 0, 0}},
		{"00FF00", [3]int{0, 255, 0}},
		{"0000FF", [3]int{0, 0, 255}},
		{"#FF0000", [3]int{255, 0, 0}},
		{"#00FF00", [3]int{0, 255, 0}},
		{"#0000FF", [3]int{0, 0, 255}},
		{"GARBAGE", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		r, g, b := hexToRGB(tt.colorHex)
		if r != tt.want[0] || g != tt.want[1] || b != tt.want[2] {
			t.Errorf("hexToRGB(%s) = %d,%d,%d; want %d,%d,%d",
				tt.colorHex, r, g, b, tt.want[0], tt.want[1], tt.want[2])
		}
	}
}

func TestShortMac(t *testing.T) {

	tests := []struct {
		mac  string
		want string
	}{
		{"AA:BB:CC:DD:EE:FF", "EE:FF"},
		{"11:22:33:44:55:66", "55:66"},
		{"GARBAGE", "RBAGE"},
	}

	for _, tt := range tests {
		short := ShortMAC(tt.mac)
		if short != tt.want {
			t.Errorf("ShortMAC(%s) = %s; want %s", tt.mac, short, tt.want)
		}
	}

}
