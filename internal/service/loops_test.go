package service

import "testing"

func TestHexToRGB(t *testing.T) {
	r, g, b := hexToRGB("FF0000")
	if r != 255 || g != 0 || b != 0 {
		t.Fatalf("expected red 255,0,0 got %d,%d,%d", r, g, b)
	}

	r, g, b = hexToRGB("00FF00")
	if r != 0 || g != 255 || b != 0 {
		t.Fatalf("expected green 0,255,0 got %d,%d,%d", r, g, b)
	}

	r, g, b = hexToRGB("#0000FF")
	if r != 0 || g != 0 || b != 255 {
		t.Fatalf("expected blue 0,0,255 got %d,%d,%d", r, g, b)
	}

	r, g, b = hexToRGB("GARBAGE")
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("expected invalid hex to return 0,0,0 got %d,%d,%d", r, g, b)
	}
}

func TestShortMac(t *testing.T) {
	short := ShortMAC("AA:BB:CC:DD:EE:FF")
	if short != "EE:FF" {
		t.Fatalf("expected EE:FF got %s", short)
	}

	short = ShortMAC("11:22:33:44:55:66")
	if short != "55:66" {
		t.Fatalf("expected 55:66 got %s", short)
	}
}
